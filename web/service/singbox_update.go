package service

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// UpdateSingbox fetches the latest SagerNet/sing-box release for this OS/arch,
// installs the binary into the daemon bin/ folder and restarts every running
// singbox inbound process on the new build. The Xray core equivalent is
// deliberately locked (it is a pinned, patched fork); sing-box is a plain
// upstream binary, so updating it in place is safe and reversible by re-running
// the panel installer.
func (s *SingboxService) UpdateSingbox() (string, error) {
	tag, err := latestSingboxTag()
	if err != nil {
		return "", err
	}
	version := strings.TrimPrefix(tag, "v")
	asset := fmt.Sprintf("sing-box-%s-linux-%s.tar.gz", version, singboxArch())
	url := fmt.Sprintf("https://github.com/SagerNet/sing-box/releases/download/%s/%s", tag, asset)

	bin, err := downloadSingboxBinary(url, asset)
	if err != nil {
		return "", err
	}

	dest := daemonBin("sing-box")
	tmp := dest + ".new"
	if err := os.WriteFile(tmp, bin, 0o755); err != nil {
		return "", fmt.Errorf("write sing-box binary: %w", err)
	}
	defer os.Remove(tmp)
	// Replacing the managed build with an official archive without stats would
	// silently remove accounting. Verify support before replacing or stopping it.
	if err := requireSingboxStatsBuild(tmp); err != nil {
		return "", err
	}
	// Atomic replace so a crash mid-update can never leave a truncated binary.
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("replace sing-box binary: %w", err)
	}

	clearDaemonVersion("sing-box")

	// Restart every running singbox inbound onto the new binary.
	_ = s.StopServices()
	if err := s.EnsureServicesRunning(); err != nil {
		return version, fmt.Errorf("installed %s but restart failed: %w", version, err)
	}
	return version, nil
}

func requireSingboxStatsBuild(path string) error {
	cmd := exec.Command(path, "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cannot inspect sing-box build: %w", err)
	}
	data, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Contains(data, []byte("vpn-aio-limits")) {
		return fmt.Errorf("this sing-box build lacks VPN-AIO speed/IP/stream enforcement; install a VPN-AIO managed build")
	}
	for _, line := range strings.Split(string(output), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "Tags:") {
			continue
		}
		for _, tag := range strings.Split(strings.TrimSpace(strings.SplitN(line, ":", 2)[1]), ",") {
			if strings.TrimSpace(tag) == "with_v2ray_api" {
				return nil
			}
		}
	}
	return fmt.Errorf("this sing-box build lacks with_v2ray_api, which managed accounts require for traffic accounting; update using a VPN-AIO bundle with stats support")
}

func singboxArch() string {
	switch runtime.GOARCH {
	case "arm64":
		return "arm64"
	case "386":
		return "386"
	default:
		return "amd64"
	}
}

func latestSingboxTag() (string, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/repos/SagerNet/sing-box/releases/latest", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("query sing-box releases: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("sing-box release lookup: HTTP %d", resp.StatusCode)
	}
	var body struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("parse sing-box release: %w", err)
	}
	if body.TagName == "" {
		return "", fmt.Errorf("sing-box release lookup: empty tag_name")
	}
	return body.TagName, nil
}

// downloadSingboxBinary fetches the release tarball and pulls out the sing-box
// binary, which sits at <asset-dirname>/sing-box inside the archive.
func downloadSingboxBinary(url, asset string) ([]byte, error) {
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("download sing-box: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download sing-box: HTTP %d for %s", resp.StatusCode, asset)
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("open sing-box tarball: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("sing-box binary not found in %s", asset)
		}
		if err != nil {
			return nil, fmt.Errorf("read sing-box tarball: %w", err)
		}
		if filepath.Base(hdr.Name) == "sing-box" && hdr.Typeflag == tar.TypeReg {
			out, err := io.ReadAll(io.LimitReader(tr, 256<<20))
			if err != nil {
				return nil, fmt.Errorf("read sing-box binary: %w", err)
			}
			return out, nil
		}
	}
}
