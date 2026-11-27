package service

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v2/database"
)

func TestSingboxAuditBundledRuntimeEnforcesSidecarSpeedIPAndStreams(t *testing.T) {
	exact := os.Getenv("VPN_AIO_TEST_SINGBOX")
	if exact == "" {
		t.Skip("requires the VPN-AIO patched sing-box build")
	}
	t.Setenv("VPNAIO_BIN_FOLDER", t.TempDir())
	singboxAuditSetup(t)
	RegisterSpeedLimitInvalidation()

	inbound := singboxAuditInbound(t)
	inbound.IPLimitStrategy = "reject"
	svc := &InboundService{}
	saved, _, err := svc.AddInbound(inbound)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = procMgr.Stop(singboxProcName(saved.Id)) })
	clients, err := svc.GetClients(saved)
	if err != nil || len(clients) != 1 {
		t.Fatalf("managed clients: %d, %v", len(clients), err)
	}
	account, err := (&AccountService{}).GetAccountByEmail(clients[0].Email)
	if err != nil || account == nil {
		t.Fatalf("managed account: %+v, %v", account, err)
	}

	var targetHits atomic.Int32
	ipHeld := make(chan struct{})
	ipRelease := make(chan struct{})
	streamHeld := make(chan struct{})
	streamRelease := make(chan struct{})
	var ipHeldOnce sync.Once
	var streamHeldOnce sync.Once
	target := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetHits.Add(1)
		if r.URL.Path == "/hold" {
			if r.Header.Get("X-Test-Hold") == "ip" {
				ipHeldOnce.Do(func() { close(ipHeld) })
				select {
				case <-ipRelease:
				case <-r.Context().Done():
					return
				}
			} else {
				streamHeldOnce.Do(func() { close(streamHeld) })
				select {
				case <-streamRelease:
				case <-r.Context().Done():
					return
				}
			}
		}
		w.Header().Set("Connection", "close")
		if r.URL.Path == "/download" {
			_, _ = w.Write(make([]byte, 4096))
			return
		}
		_, _ = io.WriteString(w, "sidecar-runtime-ok")
	})}
	targetListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	targetPort := targetListener.Addr().(*net.TCPAddr).Port
	go func() { _ = target.Serve(targetListener) }()
	t.Cleanup(func() { _ = target.Close() })
	targetURL := "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(targetPort))

	setPolicy := func(down, up, ips, streams int) {
		t.Helper()
		account.LimitIP = ips
		account.StreamLimit = streams
		if down < 0 {
			account.SpeedLimitDown = nil
		} else {
			account.SpeedLimitDown = &down
		}
		if up < 0 {
			account.SpeedLimitUp = nil
		} else {
			account.SpeedLimitUp = &up
		}
		if err := database.GetDB().Save(account).Error; err != nil {
			t.Fatal(err)
		}
		MarkSpeedLimitsDirty()
		WriteSpeedLimitsIfDirty()
	}

	startClient := func(sourceIP string) (*http.Client, func()) {
		t.Helper()
		port := singboxAuditPort(t, "tcp")
		config := map[string]any{
			"log": map[string]any{"level": "error"},
			"inbounds": []any{map[string]any{
				"type": "mixed", "listen": "127.0.0.1", "listen_port": port,
			}},
			"outbounds": []any{map[string]any{
				"type": "vless", "tag": "test-vless", "server": "127.0.0.1",
				"server_port": saved.Port, "uuid": clients[0].ID,
				"inet4_bind_address": sourceIP,
			}},
			"route": map[string]any{"final": "test-vless"},
		}
		configBytes, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		configPath := filepath.Join(t.TempDir(), "client.json")
		if err := os.WriteFile(configPath, configBytes, 0o600); err != nil {
			t.Fatal(err)
		}
		if out, err := exec.Command(exact, "check", "-c", configPath).CombinedOutput(); err != nil {
			t.Fatalf("client config rejected: %v: %s", err, out)
		}
		process := exec.Command(exact, "run", "-c", configPath)
		process.Stdout, process.Stderr = io.Discard, io.Discard
		if err := process.Start(); err != nil {
			t.Fatal(err)
		}
		var stopOnce sync.Once
		stop := func() {
			stopOnce.Do(func() {
				if process.Process != nil {
					_ = process.Process.Kill()
					_ = process.Wait()
				}
			})
		}
		t.Cleanup(stop)
		address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
		deadline := time.Now().Add(5 * time.Second)
		for {
			conn, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
			if err == nil {
				_ = conn.Close()
				break
			}
			if time.Now().After(deadline) {
				stop()
				t.Fatalf("client mixed listener failed to start: %v", err)
			}
			time.Sleep(25 * time.Millisecond)
		}
		proxy := &url.URL{Scheme: "http", Host: net.JoinHostPort("127.0.0.1", strconv.Itoa(port))}
		client := &http.Client{
			Transport: &http.Transport{Proxy: http.ProxyURL(proxy), DisableKeepAlives: true},
			Timeout:   20 * time.Second,
		}
		return client, stop
	}

	getBodyWithHeader := func(client *http.Client, path, hold string) ([]byte, error) {
		req, err := http.NewRequest(http.MethodGet, targetURL+path, nil)
		if err != nil {
			return nil, err
		}
		if hold != "" {
			req.Header.Set("X-Test-Hold", hold)
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		return io.ReadAll(resp.Body)
	}
	getBody := func(client *http.Client, path string) ([]byte, error) {
		return getBodyWithHeader(client, path, "")
	}

	// Rate controls are in KB/s in the panel and bytes/s in the sidecar. A 1 KiB/s
	// cap should make a 4 KiB response take several seconds, not a loopback instant.
	setPolicy(1, 1, 0, 0)
	client, stop := startClient("127.0.0.1")
	started := time.Now()
	body, err := getBody(client, "/download")
	elapsed := time.Since(started)
	if err != nil || len(body) != 4096 {
		t.Fatalf("limited transfer body=%d err=%v", len(body), err)
	}
	if elapsed < 2*time.Second {
		t.Fatalf("4 KiB transfer under 1 KiB/s policy completed too quickly: %v", elapsed)
	}
	client.CloseIdleConnections()
	stop()

	// Two loopback source addresses let this test exercise the source-IP cap through
	// the actual VLESS listener, rather than only unit-testing the tracker state.
	setPolicy(-1, -1, 1, 0)
	firstClient, firstStop := startClient("127.0.0.1")
	firstDone := make(chan error, 1)
	go func() {
		_, err := getBodyWithHeader(firstClient, "/hold", "ip")
		firstDone <- err
	}()
	select {
	case <-ipHeld:
	case <-time.After(5 * time.Second):
		t.Fatal("first IP-cap stream never reached the holding target")
	}
	secondClient, secondStop := startClient("127.0.0.2")
	secondBody, err := getBody(secondClient, "/quick")
	secondClient.CloseIdleConnections()
	secondStop()
	if strings.Contains(string(secondBody), "sidecar-runtime-ok") || targetHits.Load() != 2 {
		t.Fatal("second source IP exceeded the account IP cap")
	}
	if got := targetHits.Load(); got != 2 { // speed transfer + the one admitted IP-cap request
		t.Fatalf("target received %d requests, want 2; rejected IP may have reached target", got)
	}
	close(ipRelease)
	if err := <-firstDone; err != nil {
		t.Fatalf("first IP-cap stream failed: %v", err)
	}
	firstClient.CloseIdleConnections()
	firstStop()

	// With a one-stream cap, keep the first routed request alive at the target and
	// verify another TCP stream from the same source cannot be routed until release.
	setPolicy(-1, -1, 0, 1)
	streamClient, streamStop := startClient("127.0.0.1")
	streamDone := make(chan error, 1)
	go func() {
		_, err := getBodyWithHeader(streamClient, "/hold", "stream")
		streamDone <- err
	}()
	select {
	case <-streamHeld:
	case <-time.After(5 * time.Second):
		t.Fatal("first stream never reached the holding target")
	}
	secondStreamClient := *streamClient
	secondStreamClient.Timeout = 2 * time.Second
	secondBody, _ = getBodyWithHeader(&secondStreamClient, "/hold", "stream")
	if strings.Contains(string(secondBody), "sidecar-runtime-ok") || targetHits.Load() != 3 {
		t.Fatal("second concurrent stream was admitted above StreamLimit=1")
	}
	if got := targetHits.Load(); got != 3 {
		t.Fatalf("blocked second stream reached target; target requests=%d, want 3", got)
	}
	close(streamRelease)
	select {
	case err := <-streamDone:
		if err != nil {
			t.Fatalf("first held stream failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("first held stream did not finish after release")
	}
	body, err = getBody(streamClient, "/quick")
	if err != nil || !strings.Contains(string(body), "sidecar-runtime-ok") {
		t.Fatalf("stream slot was not released: %v: %s", err, body)
	}
	streamClient.CloseIdleConnections()
	streamStop()
}
