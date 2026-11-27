package service

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v2/database/model"
)

func TestSingboxAuditBundledTUICAndAnyTLSRuntime(t *testing.T) {
	exact := os.Getenv("VPN_AIO_TEST_SINGBOX")
	if exact == "" {
		t.Skip("requires the bundled sing-box build with with_v2ray_api")
	}

	for _, tc := range []struct {
		name     string
		protocol model.Protocol
		network  string
	}{
		{name: "tuic", protocol: model.TUIC, network: "udp"},
		{name: "anytls", protocol: model.ANYTLS, network: "tcp"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			singboxAuditSetup(t)
			certPath, keyPath := singboxAuditTLSCertificate(t)
			userName := "runtime-" + tc.name + "@example.test"
			password := "runtime-password-" + tc.name
			port := singboxAuditPort(t, tc.network)
			user := map[string]any{"name": userName, "password": password}
			outbound := map[string]any{
				"type": string(tc.protocol), "tag": "test-proxy",
				"server": "127.0.0.1", "server_port": port,
				"password": password,
				"tls":      map[string]any{"enabled": true, "insecure": true},
			}
			if tc.protocol == model.TUIC {
				user["uuid"] = "11111111-1111-4111-8111-111111111111"
				outbound["uuid"] = user["uuid"]
			}
			settings := map[string]any{
				"type": string(tc.protocol), "listen": "127.0.0.1", "listen_port": port,
				"users": []any{user},
				"tls":   map[string]any{"enabled": true, "certificate_path": certPath, "key_path": keyPath},
			}
			settingsBytes, err := json.Marshal(settings)
			if err != nil {
				t.Fatal(err)
			}
			inbound, _, err := (&InboundService{}).AddInbound(&model.Inbound{
				Core: "singbox", Protocol: tc.protocol, Enable: true, TrafficReset: "never",
				Settings: string(settingsBytes),
			})
			if err != nil {
				t.Fatalf("add managed %s inbound: %v", tc.name, err)
			}
			t.Cleanup(func() { procMgr.Stop(singboxProcName(inbound.Id)) })

			target := http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, "protocol-runtime-ok")
			})}
			targetListener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			targetPort := targetListener.Addr().(*net.TCPAddr).Port
			go func() { _ = target.Serve(targetListener) }()
			t.Cleanup(func() { _ = target.Close() })

			socksPort := singboxAuditPort(t, "tcp")
			clientConfig := map[string]any{
				"log": map[string]any{"level": "error"},
				"inbounds": []any{map[string]any{
					"type": "socks", "tag": "test-socks", "listen": "127.0.0.1", "listen_port": socksPort,
				}},
				"outbounds": []any{outbound},
				"route":     map[string]any{"final": "test-proxy"},
			}
			clientBytes, err := json.MarshalIndent(clientConfig, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			clientPath := filepath.Join(t.TempDir(), "client.json")
			if err := os.WriteFile(clientPath, clientBytes, 0o600); err != nil {
				t.Fatal(err)
			}
			if out, err := exec.Command(exact, "check", "-c", clientPath).CombinedOutput(); err != nil {
				t.Fatalf("%s client config rejected: %v: %s", tc.name, err, out)
			}
			client := exec.Command(exact, "run", "-c", clientPath)
			client.Stdout, client.Stderr = io.Discard, io.Discard
			if err := client.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				_ = client.Process.Kill()
				_ = client.Wait()
			})

			socksAddr := net.JoinHostPort("127.0.0.1", strconv.Itoa(socksPort))
			deadline := time.Now().Add(5 * time.Second)
			for {
				conn, err := net.DialTimeout("tcp", socksAddr, 100*time.Millisecond)
				if err == nil {
					_ = conn.Close()
					break
				}
				if time.Now().After(deadline) {
					t.Fatalf("%s client SOCKS listener did not start", tc.name)
				}
				time.Sleep(25 * time.Millisecond)
			}

			conn, err := net.DialTimeout("tcp", socksAddr, time.Second)
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			_ = conn.SetDeadline(time.Now().Add(15 * time.Second))
			if _, err := conn.Write([]byte{5, 1, 0}); err != nil {
				t.Fatal(err)
			}
			greeting := make([]byte, 2)
			if _, err := io.ReadFull(conn, greeting); err != nil || greeting[0] != 5 || greeting[1] != 0 {
				t.Fatalf("%s SOCKS greeting: %v %v", tc.name, greeting, err)
			}
			request := []byte{5, 1, 0, 1, 127, 0, 0, 1, byte(targetPort >> 8), byte(targetPort)}
			if _, err := conn.Write(request); err != nil {
				t.Fatal(err)
			}
			response := make([]byte, 10)
			if _, err := io.ReadFull(conn, response); err != nil || response[1] != 0 {
				t.Fatalf("%s SOCKS connect: %v %v", tc.name, response, err)
			}
			if _, err := io.WriteString(conn, "GET / HTTP/1.1\r\nHost: fixture\r\nConnection: close\r\n\r\n"); err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(conn)
			if err != nil || !strings.Contains(string(body), "protocol-runtime-ok") {
				t.Fatalf("%s proxied HTTP request failed: %v: %s", tc.name, err, body)
			}
		})
	}
}

func TestSingboxAuditBundledHysteria2ShadowsocksVMessAndTrojanRuntime(t *testing.T) {
	exact := os.Getenv("VPN_AIO_TEST_SINGBOX")
	if exact == "" {
		t.Skip("requires the bundled sing-box build with with_v2ray_api")
	}

	for _, tc := range []struct {
		name     string
		protocol model.Protocol
		network  string
	}{
		{name: "hysteria2", protocol: model.Hysteria2, network: "udp"},
		{name: "shadowsocks", protocol: model.Shadowsocks, network: "tcp"},
		{name: "vmess", protocol: model.VMESS, network: "tcp"},
		{name: "trojan", protocol: model.Trojan, network: "tcp"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			singboxAuditSetup(t)
			userName := "runtime-" + tc.name + "@example.test"
			password := "runtime-password-" + tc.name
			port := singboxAuditPort(t, tc.network)
			user := map[string]any{"name": userName}
			inboundSettings := map[string]any{
				"type": string(tc.protocol), "listen": "127.0.0.1", "listen_port": port,
				"users": []any{user},
			}
			outbound := map[string]any{
				"type": string(tc.protocol), "tag": "test-proxy",
				"server": "127.0.0.1", "server_port": port,
			}

			switch tc.protocol {
			case model.Hysteria2:
				certPath, keyPath := singboxAuditTLSCertificate(t)
				user["password"] = password
				inboundSettings["tls"] = map[string]any{"enabled": true, "certificate_path": certPath, "key_path": keyPath}
				outbound["password"] = password
				outbound["tls"] = map[string]any{"enabled": true, "insecure": true}
			case model.Shadowsocks:
				method := "aes-128-gcm"
				user["password"] = password
				inboundSettings["method"] = method
				outbound["method"], outbound["password"] = method, password
			case model.VMESS:
				user["uuid"] = "11111111-1111-4111-8111-111111111111"
				outbound["uuid"] = user["uuid"]
			case model.Trojan:
				certPath, keyPath := singboxAuditTLSCertificate(t)
				user["password"] = password
				inboundSettings["tls"] = map[string]any{"enabled": true, "certificate_path": certPath, "key_path": keyPath}
				outbound["password"] = password
				outbound["tls"] = map[string]any{"enabled": true, "insecure": true}
			}

			settingsBytes, err := json.Marshal(inboundSettings)
			if err != nil {
				t.Fatal(err)
			}
			inbound, _, err := (&InboundService{}).AddInbound(&model.Inbound{
				Core: "singbox", Protocol: tc.protocol, Enable: true, TrafficReset: "never",
				Settings: string(settingsBytes),
			})
			if err != nil {
				t.Fatalf("add managed %s inbound: %v", tc.name, err)
			}
			t.Cleanup(func() { _ = procMgr.Stop(singboxProcName(inbound.Id)) })

			target := http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, "protocol-runtime-ok")
			})}
			targetListener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			targetPort := targetListener.Addr().(*net.TCPAddr).Port
			go func() { _ = target.Serve(targetListener) }()
			t.Cleanup(func() { _ = target.Close() })

			socksPort := singboxAuditPort(t, "tcp")
			clientConfig := map[string]any{
				"log": map[string]any{"level": "error"},
				"inbounds": []any{map[string]any{
					"type": "socks", "tag": "test-socks", "listen": "127.0.0.1", "listen_port": socksPort,
				}},
				"outbounds": []any{outbound},
				"route":     map[string]any{"final": "test-proxy"},
			}
			clientBytes, err := json.MarshalIndent(clientConfig, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			clientPath := filepath.Join(t.TempDir(), "client.json")
			if err := os.WriteFile(clientPath, clientBytes, 0o600); err != nil {
				t.Fatal(err)
			}
			if out, err := exec.Command(exact, "check", "-c", clientPath).CombinedOutput(); err != nil {
				t.Fatalf("%s client config rejected: %v: %s", tc.name, err, out)
			}
			client := exec.Command(exact, "run", "-c", clientPath)
			client.Stdout, client.Stderr = io.Discard, io.Discard
			if err := client.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				_ = client.Process.Kill()
				_ = client.Wait()
			})

			socksAddr := net.JoinHostPort("127.0.0.1", strconv.Itoa(socksPort))
			deadline := time.Now().Add(5 * time.Second)
			for {
				conn, err := net.DialTimeout("tcp", socksAddr, 100*time.Millisecond)
				if err == nil {
					_ = conn.Close()
					break
				}
				if time.Now().After(deadline) {
					t.Fatalf("%s client SOCKS listener did not start", tc.name)
				}
				time.Sleep(25 * time.Millisecond)
			}

			conn, err := net.DialTimeout("tcp", socksAddr, time.Second)
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			_ = conn.SetDeadline(time.Now().Add(15 * time.Second))
			if _, err := conn.Write([]byte{5, 1, 0}); err != nil {
				t.Fatal(err)
			}
			greeting := make([]byte, 2)
			if _, err := io.ReadFull(conn, greeting); err != nil || greeting[0] != 5 || greeting[1] != 0 {
				t.Fatalf("%s SOCKS greeting: %v %v", tc.name, greeting, err)
			}
			request := []byte{5, 1, 0, 1, 127, 0, 0, 1, byte(targetPort >> 8), byte(targetPort)}
			if _, err := conn.Write(request); err != nil {
				t.Fatal(err)
			}
			response := make([]byte, 10)
			if _, err := io.ReadFull(conn, response); err != nil || response[1] != 0 {
				t.Fatalf("%s SOCKS connect: %v %v", tc.name, response, err)
			}
			if _, err := io.WriteString(conn, "GET / HTTP/1.1\r\nHost: fixture\r\nConnection: close\r\n\r\n"); err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(conn)
			if err != nil || !strings.Contains(string(body), "protocol-runtime-ok") {
				t.Fatalf("%s proxied HTTP request failed: %v: %s", tc.name, err, body)
			}
		})
	}
}

func TestSingboxAuditBundledShadowTLSV3StartsListener(t *testing.T) {
	if os.Getenv("VPN_AIO_TEST_SINGBOX") == "" {
		t.Skip("requires the bundled sing-box build with with_v2ray_api")
	}
	singboxAuditSetup(t)

	handshakeTarget := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "handshake-target")
	}))
	t.Cleanup(handshakeTarget.Close)
	handshakePort := handshakeTarget.Listener.Addr().(*net.TCPAddr).Port
	port := singboxAuditPort(t, "tcp")
	settings, err := json.Marshal(map[string]any{
		"type": "shadowtls", "version": 3, "listen": "127.0.0.1", "listen_port": port,
		"users":     []any{map[string]any{"name": "shadowtls-runtime@example.test", "password": "shadowtls-runtime-password"}},
		"handshake": map[string]any{"server": "127.0.0.1", "server_port": handshakePort},
	})
	if err != nil {
		t.Fatal(err)
	}
	inbound, _, err := (&InboundService{}).AddInbound(&model.Inbound{
		Core: "singbox", Protocol: model.Protocol("shadowtls"), Enable: true,
		TrafficReset: "never", Settings: string(settings),
	})
	if err != nil {
		t.Fatalf("add valid ShadowTLS v3 inbound: %v", err)
	}
	t.Cleanup(func() { procMgr.Stop(singboxProcName(inbound.Id)) })

	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	deadline := time.Now().Add(5 * time.Second)
	for {
		conn, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("ShadowTLS v3 config passed save but listener did not bind: %v", err)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func TestSingboxAuditBundledShadowTLSV3AuthenticatedDetourRuntime(t *testing.T) {
	exact := os.Getenv("VPN_AIO_TEST_SINGBOX")
	if exact == "" {
		t.Skip("requires the bundled sing-box build with with_v2ray_api")
	}
	singboxAuditSetup(t)

	handshakeTarget := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "shadowtls-handshake-target")
	}))
	t.Cleanup(handshakeTarget.Close)
	handshakePort := handshakeTarget.Listener.Addr().(*net.TCPAddr).Port
	shadowTLSPort := singboxAuditPort(t, "tcp")
	detourPort := singboxAuditPort(t, "tcp")
	clientPort := singboxAuditPort(t, "tcp")
	userName := "shadowtls-auth@example.test"
	shadowTLSPassword := "shadowtls-user-password"
	shadowsocksPassword := "shadowtls-detour-password"
	method := "aes-128-gcm"

	settings, err := json.Marshal(map[string]any{
		"type": "shadowtls", "listen": "127.0.0.1", "listen_port": shadowTLSPort,
		"version":   3,
		"users":     []any{map[string]any{"name": userName, "password": shadowTLSPassword}},
		"handshake": map[string]any{"server": "127.0.0.1", "server_port": handshakePort},
	})
	if err != nil {
		t.Fatal(err)
	}
	inbound, _, err := (&InboundService{}).AddInbound(&model.Inbound{
		Core: "singbox", Protocol: model.Protocol("shadowtls"), Enable: true,
		TrafficReset: "never", Settings: string(settings),
	})
	if err != nil {
		t.Fatalf("add managed ShadowTLS v3 inbound: %v", err)
	}
	serverProcessName := singboxProcName(inbound.Id)
	t.Cleanup(func() { _ = procMgr.Stop(serverProcessName) })
	if err := procMgr.Stop(serverProcessName); err != nil {
		t.Fatalf("stop single-inbound process before assembling detour fixture: %v", err)
	}

	serverConfigPath := filepath.Join(singboxService.configDir(inbound.Id), "config.json")
	serverBytes, err := os.ReadFile(serverConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	var serverConfig map[string]any
	if err := json.Unmarshal(serverBytes, &serverConfig); err != nil {
		t.Fatal(err)
	}
	inbounds, ok := serverConfig["inbounds"].([]any)
	if !ok || len(inbounds) != 1 {
		t.Fatalf("expected the generated managed config to contain one inbound, got %#v", serverConfig["inbounds"])
	}
	shadowTLSInbound, ok := inbounds[0].(map[string]any)
	if !ok {
		t.Fatalf("generated ShadowTLS inbound has type %T", inbounds[0])
	}
	shadowTLSInbound["detour"] = "shadowtls-detour"
	inbounds = append(inbounds, map[string]any{
		"type": "shadowsocks", "tag": "shadowtls-detour",
		"listen": "127.0.0.1", "listen_port": detourPort,
		"method": method, "password": shadowsocksPassword,
	})
	serverConfig["inbounds"] = inbounds
	outbounds, ok := serverConfig["outbounds"].([]any)
	if !ok || len(outbounds) == 0 {
		t.Fatalf("generated managed config has no outbound: %#v", serverConfig["outbounds"])
	}
	directOutbound, ok := outbounds[0].(map[string]any)
	if !ok {
		t.Fatalf("generated direct outbound has type %T", outbounds[0])
	}
	directOutbound["tag"] = "direct"
	serverRoute, _ := serverConfig["route"].(map[string]any)
	if serverRoute == nil {
		serverRoute = map[string]any{}
	}
	serverRules, _ := serverRoute["rules"].([]any)
	serverRules = append([]any{map[string]any{
		"inbound": []any{"shadowtls-detour"}, "outbound": "direct",
	}}, serverRules...)
	serverRoute["rules"], serverRoute["final"] = serverRules, "direct"
	serverConfig["route"] = serverRoute
	serverBytes, err = json.MarshalIndent(serverConfig, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(serverConfigPath, serverBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(exact, "check", "-c", serverConfigPath).CombinedOutput(); err != nil {
		t.Fatalf("managed ShadowTLS+detour server config rejected: %v: %s", err, out)
	}

	startCore := func(label, configPath string) *exec.Cmd {
		t.Helper()
		cmd := exec.Command(exact, "run", "-c", configPath)
		cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
		if err := cmd.Start(); err != nil {
			t.Fatalf("start %s sing-box process: %v", label, err)
		}
		t.Cleanup(func() {
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
				_ = cmd.Wait()
			}
		})
		return cmd
	}
	waitForTCP := func(label string, port int) {
		t.Helper()
		address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
		deadline := time.Now().Add(5 * time.Second)
		for {
			conn, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
			if err == nil {
				_ = conn.Close()
				return
			}
			if time.Now().After(deadline) {
				t.Fatalf("%s listener did not start: %v", label, err)
			}
			time.Sleep(25 * time.Millisecond)
		}
	}

	startCore("managed ShadowTLS server", serverConfigPath)
	waitForTCP("ShadowTLS server", shadowTLSPort)

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "shadowtls-authenticated-detour-ok")
	}))
	t.Cleanup(target.Close)

	clientConfigForPassword := func(password string) map[string]any {
		return map[string]any{
			"log": map[string]any{"level": "error"},
			"inbounds": []any{map[string]any{
				"type": "mixed", "tag": "test-proxy", "listen": "127.0.0.1", "listen_port": clientPort,
			}},
			"outbounds": []any{
				map[string]any{
					"type": "shadowsocks", "tag": "ss-out",
					"server": "127.0.0.1", "server_port": shadowTLSPort,
					"method": method, "password": shadowsocksPassword,
					"detour": "shadowtls-out",
				},
				map[string]any{
					"type": "shadowtls", "tag": "shadowtls-out",
					"server": "127.0.0.1", "server_port": shadowTLSPort,
					"version": 3, "password": password,
					"tls": map[string]any{"enabled": true, "server_name": "localhost", "insecure": true},
				},
				map[string]any{"type": "direct", "tag": "direct"},
			},
			"route": map[string]any{"final": "ss-out"},
		}
	}

	runClient := func(label, password string) (*http.Client, *exec.Cmd) {
		t.Helper()
		configBytes, err := json.MarshalIndent(clientConfigForPassword(password), "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		configPath := filepath.Join(t.TempDir(), label+".json")
		if err := os.WriteFile(configPath, configBytes, 0o600); err != nil {
			t.Fatal(err)
		}
		if out, err := exec.Command(exact, "check", "-c", configPath).CombinedOutput(); err != nil {
			t.Fatalf("%s client config rejected: %v: %s", label, err, out)
		}
		clientProcess := startCore(label+" client", configPath)
		waitForTCP(label+" proxy", clientPort)
		proxyURL := &url.URL{Scheme: "http", Host: net.JoinHostPort("127.0.0.1", strconv.Itoa(clientPort))}
		return &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}, Timeout: 10 * time.Second}, clientProcess
	}
	stopClient := func(client *http.Client, process *exec.Cmd) {
		client.CloseIdleConnections()
		_ = process.Process.Kill()
		_ = process.Wait()
	}

	client, validClientProcess := runClient("shadowtls-valid", shadowTLSPassword)
	response, err := client.Get(target.URL)
	if err != nil {
		t.Fatalf("authenticated ShadowTLS/ShadowSocks detour request: %v", err)
	}
	body, readErr := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if readErr != nil || !strings.Contains(string(body), "shadowtls-authenticated-detour-ok") {
		t.Fatalf("authenticated ShadowTLS traffic missed the detour target: status=%s err=%v body=%s", response.Status, readErr, body)
	}
	stopClient(client, validClientProcess)

	invalidClient, invalidClientProcess := runClient("shadowtls-invalid", "wrong-shadowtls-password")
	response, err = invalidClient.Get(target.URL)
	if err == nil {
		body, readErr = io.ReadAll(response.Body)
		_ = response.Body.Close()
		if readErr != nil {
			t.Fatalf("read wrong-password fallback response: %v", readErr)
		}
		if strings.Contains(string(body), "shadowtls-authenticated-detour-ok") {
			t.Fatal("wrong ShadowTLS password reached the authenticated detour")
		}
	}
	stopClient(invalidClient, invalidClientProcess)
}

func TestSingboxAuditBundledShadowTLSV3ManagedMultiInboundRuntime(t *testing.T) {
	exact := os.Getenv("VPN_AIO_TEST_SINGBOX")
	if exact == "" {
		t.Skip("requires the bundled sing-box build with with_v2ray_api")
	}
	singboxAuditSetup(t)

	handshakeTarget := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "managed-chain-handshake")
	}))
	t.Cleanup(handshakeTarget.Close)
	handshakePort := handshakeTarget.Listener.Addr().(*net.TCPAddr).Port
	targetPort := singboxAuditPort(t, "tcp")
	frontPort := singboxAuditPort(t, "tcp")
	method := "aes-128-gcm"
	targetPassword := "managed-chain-target-password"
	frontPassword := "managed-chain-front-password"
	targetName := "managed-chain-target@example.test"
	frontName := "managed-chain-front@example.test"

	targetSettings, err := json.Marshal(map[string]any{
		"type": string(model.Shadowsocks), "listen": "127.0.0.1", "listen_port": targetPort,
		"method": method, "users": []any{map[string]any{"name": targetName, "password": targetPassword}},
	})
	if err != nil {
		t.Fatal(err)
	}
	target, _, err := (&InboundService{}).AddInbound(&model.Inbound{
		Core: "singbox", Protocol: model.Shadowsocks, Enable: true, TrafficReset: "never",
		Settings: string(targetSettings),
	})
	if err != nil {
		t.Fatalf("add managed Shadowsocks detour target: %v", err)
	}
	processName := singboxProcName(target.Id)
	t.Cleanup(func() { _ = procMgr.Stop(processName) })

	frontSettings, err := json.Marshal(map[string]any{
		"type": "shadowtls", "listen": "127.0.0.1", "listen_port": frontPort,
		"version": 3, "detour": target.Tag,
		"users":     []any{map[string]any{"name": frontName, "password": frontPassword}},
		"handshake": map[string]any{"server": "127.0.0.1", "server_port": handshakePort},
	})
	if err != nil {
		t.Fatal(err)
	}
	front, _, err := (&InboundService{}).AddInbound(&model.Inbound{
		Core: "singbox", Protocol: model.Protocol("shadowtls"), Enable: true, TrafficReset: "never",
		Settings: string(frontSettings),
	})
	if err != nil {
		t.Fatalf("add managed ShadowTLS front: %v", err)
	}
	if front.SingboxDetourInboundID != target.Id {
		t.Fatalf("front detour relation = %d, want target %d", front.SingboxDetourInboundID, target.Id)
	}

	serverConfigPath := filepath.Join(singboxService.configDir(target.Id), "config.json")
	serverBytes, err := os.ReadFile(serverConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	var serverConfig map[string]any
	if err := json.Unmarshal(serverBytes, &serverConfig); err != nil {
		t.Fatal(err)
	}
	serverInbounds, ok := serverConfig["inbounds"].([]any)
	if !ok || len(serverInbounds) != 2 {
		t.Fatalf("managed process has %d inbounds, want 2", len(serverInbounds))
	}
	frontFound := false
	for _, raw := range serverInbounds {
		object, _ := raw.(map[string]any)
		if object["tag"] == front.Tag {
			frontFound = true
			if object["detour"] != target.Tag {
				t.Fatalf("rendered front detour = %#v, want %q", object["detour"], target.Tag)
			}
		}
	}
	if !frontFound {
		t.Fatal("managed process config omitted the ShadowTLS front")
	}

	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "managed-shadowtls-chain-ok")
	}))
	t.Cleanup(targetServer.Close)

	clientPort := singboxAuditPort(t, "tcp")
	clientConfig := map[string]any{
		"log": map[string]any{"level": "error"},
		"inbounds": []any{map[string]any{
			"type": "mixed", "tag": "managed-chain-client", "listen": "127.0.0.1", "listen_port": clientPort,
		}},
		"outbounds": []any{
			map[string]any{
				"type": "shadowsocks", "tag": "managed-chain-ss-out",
				"server": "127.0.0.1", "server_port": frontPort,
				"method": method, "password": targetPassword,
				"detour": "managed-chain-shadowtls-out",
			},
			map[string]any{
				"type": "shadowtls", "tag": "managed-chain-shadowtls-out",
				"server": "127.0.0.1", "server_port": frontPort,
				"version": 3, "password": frontPassword,
				"tls": map[string]any{"enabled": true, "server_name": "localhost", "insecure": true},
			},
			map[string]any{"type": "direct", "tag": "direct"},
		},
		"route": map[string]any{"final": "managed-chain-ss-out"},
	}
	clientBytes, err := json.MarshalIndent(clientConfig, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	clientPath := filepath.Join(t.TempDir(), "managed-chain-client.json")
	if err := os.WriteFile(clientPath, clientBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(exact, "check", "-c", clientPath).CombinedOutput(); err != nil {
		t.Fatalf("managed chain client config rejected: %v: %s", err, out)
	}
	clientProcess := exec.Command(exact, "run", "-c", clientPath)
	clientProcess.Stdout, clientProcess.Stderr = io.Discard, io.Discard
	if err := clientProcess.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if clientProcess.Process != nil {
			_ = clientProcess.Process.Kill()
			_ = clientProcess.Wait()
		}
	})

	waitForTCP := func(label string, port int) {
		t.Helper()
		address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
		deadline := time.Now().Add(5 * time.Second)
		for {
			conn, dialErr := net.DialTimeout("tcp", address, 100*time.Millisecond)
			if dialErr == nil {
				_ = conn.Close()
				return
			}
			if time.Now().After(deadline) {
				t.Fatalf("%s listener did not start: %v", label, dialErr)
			}
			time.Sleep(25 * time.Millisecond)
		}
	}
	waitForTCP("managed ShadowTLS front", frontPort)
	waitForTCP("managed chain client", clientPort)

	proxyURL := &url.URL{Scheme: "http", Host: net.JoinHostPort("127.0.0.1", strconv.Itoa(clientPort))}
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}, Timeout: 10 * time.Second}
	response, err := client.Get(targetServer.URL)
	if err != nil {
		t.Fatalf("managed ShadowTLS multi-inbound request: %v", err)
	}
	body, readErr := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if readErr != nil || !strings.Contains(string(body), "managed-shadowtls-chain-ok") {
		t.Fatalf("managed ShadowTLS multi-inbound body: status=%s err=%v body=%s", response.Status, readErr, body)
	}
}

func singboxAuditPort(t *testing.T, network string) int {
	t.Helper()
	var address net.Addr
	if network == "udp" {
		listener, err := net.ListenPacket(network, "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer listener.Close()
		address = listener.LocalAddr()
	} else {
		listener, err := net.Listen(network, "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer listener.Close()
		address = listener.Addr()
	}
	_, portText, err := net.SplitHostPort(address.String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatal(err)
	}
	return port
}

func singboxAuditTLSCertificate(t *testing.T) (string, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	template := x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    now.Add(-time.Minute),
		NotAfter:     now.Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyBytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certPath := filepath.Join(t.TempDir(), "server.crt")
	keyPath := filepath.Join(t.TempDir(), "server.key")
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes}), 0o600); err != nil {
		t.Fatal(err)
	}
	return certPath, keyPath
}
