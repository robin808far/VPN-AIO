package service

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mhsanaei/3x-ui/v2/config"
	"github.com/mhsanaei/3x-ui/v2/logger"
)

// MigrateLegacyLayout moves an install made before the product rename onto the
// VPN-AIO names. It must run as root at panel startup BEFORE any core, daemon,
// control socket or log consumer starts, so every generated config is written
// under the new paths. Idempotent: once nothing with the legacy name is left,
// it does nothing. The legacy names themselves live only in config/legacy.go.
func MigrateLegacyLayout() {
	if os.Geteuid() != 0 {
		return
	}
	// Stop daemons a previous panel launched from the legacy dirs/bundles first,
	// so nothing holds the trees being renamed and no orphan keeps its ports.
	if commandExists("pkill") {
		_ = exec.Command("pkill", "-KILL", "-f", config.LegacyProcessPattern).Run()
	}
	res := migrateLegacyIn("/", config.LegacyScanDirs, extraLegacyFiles())
	for _, m := range res.moved {
		logger.Infof("rename migration: %s -> %s", m[0], m[1])
	}
	for _, w := range res.warnings {
		logger.Warning("rename migration:", w)
	}
	// The old menu command: the new one is (re)installed by ensureMenuInstalled.
	if err := os.Remove(config.LegacyMenuPath); err == nil {
		logger.Info("rename migration: removed legacy menu", config.LegacyMenuPath)
	}
	if _, err := os.Stat(filepath.Join("/etc/systemd/system", config.LegacySystemdUnit)); err == nil {
		logger.Warning("rename migration: legacy systemd unit", config.LegacySystemdUnit,
			"still exists; re-run `--systemd` to install the VPN-AIO unit, then remove it")
	}
}

// extraLegacyFiles lists the panel-owned files that sit next to the binary or
// in the DB/log folders and were named after the old product.
func extraLegacyFiles() []string {
	var out []string
	for _, cur := range []string{
		ControlSocketPath(),
		filepath.Join(config.GetLogFolder(), "vpn-aio.log"),
	} {
		dir, base := filepath.Split(cur)
		if legacy := config.LegacyName(base); legacy != base {
			out = append(out, filepath.Join(dir, legacy))
		}
	}
	return out
}

type legacyMigrateResult struct {
	moved    [][2]string
	warnings []string
}

// migrateLegacyIn does the work under root (a temp dir in tests). Every direct
// entry of each scan dir whose name carries the legacy token, and every file in
// extra, is renamed to the current name. A target that already exists is never
// overwritten: the legacy copy is left alone and reported.
func migrateLegacyIn(root string, scanDirs, extra []string) legacyMigrateResult {
	var res legacyMigrateResult
	var candidates []string
	for _, d := range scanDirs {
		entries, err := os.ReadDir(filepath.Join(root, d))
		if err != nil {
			continue
		}
		for _, e := range entries {
			if config.HasLegacyToken(e.Name()) {
				candidates = append(candidates, filepath.Join(root, d, e.Name()))
			}
		}
	}
	for _, f := range extra {
		candidates = append(candidates, filepath.Join(root, f))
	}
	for _, old := range candidates {
		if _, err := os.Lstat(old); err != nil {
			continue
		}
		dst := filepath.Join(filepath.Dir(old), config.CurrentName(filepath.Base(old)))
		if dst == old {
			continue
		}
		if _, err := os.Lstat(dst); err == nil {
			res.warnings = append(res.warnings, fmt.Sprintf("%s and %s both exist; keeping both, using the new one", old, dst))
			continue
		}
		if err := movePath(old, dst); err != nil {
			res.warnings = append(res.warnings, fmt.Sprintf("could not move %s to %s: %v", old, dst, err))
			continue
		}
		res.moved = append(res.moved, [2]string{old, dst})
	}
	return res
}

// movePath renames, falling back to copy+remove across filesystems.
func movePath(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := copyTree(src, dst); err != nil {
		_ = os.RemoveAll(dst)
		return err
	}
	return os.RemoveAll(src)
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			link, lerr := os.Readlink(p)
			if lerr != nil {
				return lerr
			}
			return os.Symlink(strings.ReplaceAll(link, config.LegacyToken, config.CurrentToken), target)
		case info.IsDir():
			return os.MkdirAll(target, info.Mode().Perm())
		case !info.Mode().IsRegular():
			return nil // sockets/fifos are recreated by their owners
		}
		in, err := os.Open(p)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			out.Close()
			return err
		}
		return out.Close()
	})
}

// removeLegacyLeftovers is the uninstall half: every pre-rename path that the
// startup migration could not move (both names existed) is removed too.
func removeLegacyLeftovers(r *UninstallReport) {
	for _, d := range config.LegacyScanDirs {
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if config.HasLegacyToken(e.Name()) {
				removePath(r, filepath.Join(d, e.Name()))
			}
		}
	}
	removePath(r, config.LegacyMenuPath)
}
