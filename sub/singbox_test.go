package sub

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v2/database/model"
)

func singboxLinkFixture(t *testing.T, typ string, block map[string]any) *model.Inbound {
	t.Helper()
	block["type"], block["listen_port"] = typ, 34443
	raw, err := json.Marshal(block)
	if err != nil {
		t.Fatal(err)
	}
	return &model.Inbound{Core: "singbox", Protocol: model.Protocol(typ), Port: 34443, Listen: "vpn.example", Settings: string(raw)}
}

func TestSingboxLinkUserSelectionAndWebSocket(t *testing.T) {
	inbound := singboxLinkFixture(t, "vless", map[string]any{
		"users":     []map[string]any{{"name": "alice", "uuid": "alice-id"}, {"name": "bob", "uuid": "bob-id"}},
		"transport": map[string]any{"type": "ws", "path": "/custom?ed=2048", "headers": map[string]any{"Host": []string{"cdn.example"}}},
	})
	link, err := url.Parse(subSvc().genSingboxLink(inbound, " BOB "))
	if err != nil {
		t.Fatal(err)
	}
	if link.User.Username() != "bob-id" || link.Query().Get("path") != "/custom?ed=2048" || link.Query().Get("host") != "cdn.example" {
		t.Fatalf("wrong user/transport: %s", link)
	}
	for _, email := range []string{"", "unknown"} {
		if got := subSvc().genSingboxLink(inbound, email); got != "" {
			t.Fatalf("unknown identity received a link: %s", got)
		}
	}
}

func TestSingboxLinkRealityAndDisabledTLS(t *testing.T) {
	block := map[string]any{
		"users": []map[string]any{{"name": "alice", "uuid": "alice-id"}},
		"tls": map[string]any{"enabled": true, "server_name": "target.example", "reality": map[string]any{
			"enabled": true, "private_key": "AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE", "short_id": []string{"0123"},
		}},
	}
	inbound := singboxLinkFixture(t, "vless", block)
	link, err := url.Parse(subSvc().genSingboxLink(inbound, "alice"))
	if err != nil {
		t.Fatal(err)
	}
	if link.Query().Get("security") != "reality" || link.Query().Get("pbk") == "" || link.Query().Get("sid") != "0123" || link.Query().Get("sni") != "target.example" {
		t.Fatalf("missing REALITY: %s", link)
	}
	if strings.Contains(link.String(), "AQEBAQEBAQ") {
		t.Fatal("server private key exported")
	}
	block["tls"].(map[string]any)["enabled"] = false
	link, err = url.Parse(subSvc().genSingboxLink(singboxLinkFixture(t, "vless", block), "alice"))
	if err != nil || link.Query().Get("security") != "none" || link.Query().Get("pbk") != "" {
		t.Fatalf("disabled TLS exported: %v %v", link, err)
	}
}

func TestSingboxLinkPasswordsAndOptionalObfs(t *testing.T) {
	for _, typ := range []string{"trojan", "hysteria2", "anytls"} {
		t.Run(typ, func(t *testing.T) {
			password := "p:@/?#% فارسی"
			inbound := singboxLinkFixture(t, typ, map[string]any{
				"users": []map[string]any{{"name": "alice", "password": password}}, "tls": map[string]any{"enabled": true},
			})
			link, err := url.Parse(subSvc().genSingboxLink(inbound, "alice"))
			if err != nil || link.User == nil || link.User.Username() != password {
				t.Fatalf("password URI did not round-trip: %v %v", link, err)
			}
			if typ == "hysteria2" && link.Query().Has("obfs-password") {
				t.Fatal("absent obfs exported")
			}
		})
	}
}

func TestSingboxLinkVMessTransportAndTUICSettings(t *testing.T) {
	inbound := singboxLinkFixture(t, "vmess", map[string]any{
		"users":     []map[string]any{{"name": "alice", "uuid": "alice-id"}},
		"transport": map[string]any{"type": "ws", "path": "/saved", "headers": map[string]any{"host": "cdn.example"}},
	})
	link := subSvc().genSingboxLink(inbound, "alice")
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(link, "vmess://"))
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if err := json.Unmarshal(decoded, &config); err != nil {
		t.Fatal(err)
	}
	if config["net"] != "ws" || config["path"] != "/saved" || config["host"] != "cdn.example" || config["tls"] != "" {
		t.Fatalf("incomplete vmess: %s", decoded)
	}
	tuic := singboxLinkFixture(t, "tuic", map[string]any{
		"users":              []map[string]any{{"name": "alice", "uuid": "alice-id", "password": "pass"}},
		"congestion_control": "bbr", "tls": map[string]any{"enabled": true, "alpn": []string{"h3", "custom"}},
	})
	parsed, err := url.Parse(subSvc().genSingboxLink(tuic, "alice"))
	if err != nil || parsed.Query().Get("congestion_control") != "bbr" || parsed.Query().Get("alpn") != "h3,custom" {
		t.Fatalf("TUIC settings lost: %v %v", parsed, err)
	}
}
