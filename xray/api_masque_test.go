package xray

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestMasqueLiveAccountAddRemoveAndReenable(t *testing.T) {
	binaryPath := os.Getenv("VPN_AIO_XRAY_TEST_BINARY")
	if binaryPath == "" {
		t.Skip("set VPN_AIO_XRAY_TEST_BINARY to the pinned core executable")
	}
	dir := t.TempDir()
	cert, key := filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem")
	if out, err := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:2048", "-nodes", "-days", "1", "-subj", "/CN=localhost", "-keyout", key, "-out", cert).CombinedOutput(); err != nil {
		t.Fatalf("certificate: %v: %s", err, out)
	}
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "live-masque-ok") }))
	defer httpServer.Close()
	_, destPort, err := net.SplitHostPort(httpServer.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	freePort := func() int {
		t.Helper()
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer listener.Close()
		return listener.Addr().(*net.TCPAddr).Port
	}
	apiPort, serverPort := freePort(), freePort()
	serverConfig := map[string]any{
		"log":       map[string]any{"loglevel": "warning"},
		"api":       map[string]any{"tag": "api", "listen": fmt.Sprintf("127.0.0.1:%d", apiPort), "services": []string{"HandlerService"}},
		"inbounds":  []any{map[string]any{"tag": "masque-live", "listen": "127.0.0.1", "port": serverPort, "protocol": "masque", "settings": map[string]any{"clients": []any{}, "address": []string{"198.18.71.1/24"}, "mtu": 1280}, "streamSettings": map[string]any{"network": "masque", "security": "tls", "tlsSettings": map[string]any{"alpn": []string{"h2"}, "certificates": []any{map[string]any{"certificateFile": cert, "keyFile": key}}}}}},
		"outbounds": []any{map[string]any{"tag": "redirect", "protocol": "freedom", "settings": map[string]any{"redirect": "127.0.0.1:" + destPort, "finalRules": []any{map[string]any{"action": "allow"}}}}},
		"routing":   map[string]any{"rules": []any{map[string]any{"type": "field", "network": "tcp", "outboundTag": "redirect"}}},
	}
	serverConfigPath := filepath.Join(dir, "server.json")
	writeConfig(t, serverConfigPath, serverConfig)
	server := exec.Command(binaryPath, "run", "-c", serverConfigPath)
	server.Stdout, server.Stderr = io.Discard, io.Discard
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Process.Kill(); _ = server.Wait() })
	waitTCP(t, fmt.Sprintf("127.0.0.1:%d", apiPort))
	var api XrayAPI
	if err := api.Init(apiPort); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(api.Close)
	requestThrough := func(password string) (string, error) {
		t.Helper()
		proxyPort := freePort()
		clientConfig := map[string]any{
			"log":       map[string]any{"loglevel": "warning"},
			"inbounds":  []any{map[string]any{"listen": "127.0.0.1", "port": proxyPort, "protocol": "socks", "settings": map[string]any{"auth": "noauth"}}},
			"outbounds": []any{map[string]any{"tag": "proxy", "protocol": "masque", "settings": map[string]any{"address": "127.0.0.1", "port": serverPort}, "streamSettings": map[string]any{"network": "masque", "security": "tls", "masqueSettings": map[string]any{"user": "live@masque", "pass": password}, "tlsSettings": map[string]any{"serverName": "localhost", "allowInsecure": true, "alpn": []string{"h2"}}}}},
		}
		path := filepath.Join(dir, "client.json")
		writeConfig(t, path, clientConfig)
		client := exec.Command(binaryPath, "run", "-c", path)
		client.Stdout, client.Stderr = io.Discard, io.Discard
		if err := client.Start(); err != nil {
			return "", err
		}
		defer func() { _ = client.Process.Kill(); _ = client.Wait() }()
		if err := waitTCPResult(fmt.Sprintf("127.0.0.1:%d", proxyPort)); err != nil {
			return "", err
		}
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", proxyPort), time.Second)
		if err != nil {
			return "", err
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		if _, err = conn.Write([]byte{5, 1, 0}); err != nil {
			return "", err
		}
		greeting := make([]byte, 2)
		if _, err = io.ReadFull(conn, greeting); err != nil {
			return "", err
		}
		if greeting[1] != 0 {
			return "", fmt.Errorf("SOCKS auth method %d", greeting[1])
		}
		portBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(portBytes, 80)
		if _, err = conn.Write(append([]byte{5, 1, 0, 1, 192, 0, 2, 1}, portBytes...)); err != nil {
			return "", err
		}
		var reply [4]byte
		if _, err = io.ReadFull(conn, reply[:]); err != nil {
			return "", err
		}
		if reply[1] != 0 {
			return "", fmt.Errorf("SOCKS connect code %d", reply[1])
		}
		switch reply[3] {
		case 1:
			_, err = io.CopyN(io.Discard, conn, 6)
		case 4:
			_, err = io.CopyN(io.Discard, conn, 18)
		case 3:
			var n [1]byte
			_, err = io.ReadFull(conn, n[:])
			if err == nil {
				_, err = io.CopyN(io.Discard, conn, int64(n[0])+2)
			}
		default:
			return "", fmt.Errorf("SOCKS address type %d", reply[3])
		}
		if err != nil {
			return "", err
		}
		if _, err = io.WriteString(conn, "GET / HTTP/1.1\r\nHost: live.test\r\nConnection: close\r\n\r\n"); err != nil {
			return "", err
		}
		response, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: http.MethodGet})
		if err != nil {
			return "", err
		}
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		return string(body), err
	}
	user := map[string]any{"email": "live@masque", "password": "first-secret"}
	if err := api.AddUser("masque", "masque-live", user); err != nil {
		t.Fatalf("live add: %v", err)
	}
	if got, err := requestThrough("first-secret"); err != nil || got != "live-masque-ok" {
		t.Fatalf("new account request = %q, %v", got, err)
	}
	if err := api.RemoveUser("masque-live", "live@masque"); err != nil {
		t.Fatalf("live remove: %v", err)
	}
	if got, err := requestThrough("first-secret"); err == nil && got == "live-masque-ok" {
		t.Fatal("removed MASQUE account still authenticated")
	}
	user["password"] = "rotated-secret"
	if err := api.AddUser("masque", "masque-live", user); err != nil {
		t.Fatalf("re-enable: %v", err)
	}
	if got, err := requestThrough("rotated-secret"); err != nil || got != "live-masque-ok" {
		t.Fatalf("re-enabled account request = %q, %v", got, err)
	}
}

func writeConfig(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func waitTCP(t *testing.T, address string) {
	t.Helper()
	if err := waitTCPResult(address); err != nil {
		t.Fatal(err)
	}
}

func waitTCPResult(address string) error {
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(25 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for %s", address)
}
