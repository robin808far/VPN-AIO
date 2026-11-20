package sub

import (
	"encoding/json"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/mhsanaei/3x-ui/v2/database"
	"github.com/mhsanaei/3x-ui/v2/database/model"
)

func TestHostsReplaceEndpointsWithoutChangingQuotaOrStoredInbound(t *testing.T) {
	newSubTestDB(t)
	ib := seedSubInbound(t, model.VLESS, 41942, "original", subClient("hosts-user", "hosts-sub"))
	stream := `{"network":"ws","security":"tls","wsSettings":{"path":"/old"},"tlsSettings":{"serverName":"old.example","settings":{"fingerprint":"chrome"}},"externalProxy":[{"dest":"legacy.example","port":443,"forceTls":"same","remark":"old"}]}`
	if err := database.GetDB().Model(ib).Update("stream_settings", stream).Error; err != nil {
		t.Fatal(err)
	}
	seedTraffic(t, ib.Id, "hosts-user", 10, 20, 1000, 0)
	rows := []map[string]any{
		{"group_id": "edge-b", "inbound_id": ib.Id, "sort_order": 2, "remark": "B", "address": "b.example", "port": 9443, "security": "same", "sni": "b.sni", "path": "/new"},
		{"group_id": "edge-a", "inbound_id": ib.Id, "sort_order": 1, "remark": "A", "address": "a.example", "port": 8443, "security": "same", "sni": "a.sni", "path": "/new"},
	}
	for _, row := range rows {
		if err := database.GetDB().Table("hosts").Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	links, _, usage, err := subSvc().GetSubs("hosts-sub", "panel.example")
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 2 {
		t.Fatalf("links = %q, want two host entries", links)
	}
	for i, want := range []string{"a.example:8443", "b.example:9443"} {
		u, err := url.Parse(links[i])
		if err != nil {
			t.Fatal(err)
		}
		if u.Host != want || u.Query().Get("path") != "/new" || u.Query().Get("sni") != strings.Split(want, ".")[0]+".sni" {
			t.Fatalf("host %d: %s", i, links[i])
		}
	}
	if usage.Up != 10 || usage.Down != 20 || usage.Total != 1000 {
		t.Fatalf("host fanout multiplied usage: %#v", usage)
	}
	var stored model.Inbound
	if err := database.GetDB().First(&stored, ib.Id).Error; err != nil {
		t.Fatal(err)
	}
	if stored.StreamSettings != stream {
		t.Fatalf("render mutated stored stream: %s", stored.StreamSettings)
	}
}

func TestHostsJSONClashOverridesAndDisabledClientBinding(t *testing.T) {
	newSubTestDB(t)
	client := subClient("bound-user", "bound-sub")
	ib := seedSubInbound(t, model.VLESS, 41943, "source", client)
	stream := `{"network":"ws","security":"tls","wsSettings":{"path":"/base"},"tlsSettings":{"serverName":"original.sni","settings":{"fingerprint":"chrome"}}}`
	if err := database.GetDB().Model(ib).Update("stream_settings", stream).Error; err != nil {
		t.Fatal(err)
	}
	seedTraffic(t, ib.Id, "bound-user", 11, 22, 999, 0)
	h := &model.Host{GroupId: "host-json", InboundId: ib.Id, Remark: "json-edge", Address: "edge.example", Port: 8443, Security: "same", Sni: "edge.sni", Fingerprint: "firefox", HostHeader: "http.example", Path: "/edge", SockoptParams: `{"tcpFastOpen":true}`, MuxParams: `{"enabled":true,"concurrency":4}`, MihomoIpVersion: "ipv4-prefer", AllowInsecure: true}
	h.PinnedPeerCertSha256 = []string{strings.Repeat("11", 32)}
	if err := database.GetDB().Create(h).Error; err != nil {
		t.Fatal(err)
	}
	jsonService := NewSubJsonService("", "", "", "", subSvc())
	body, header, err := jsonService.GetJson("bound-sub", "panel.example")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatalf("JSON: %v %s", err, body)
	}
	proxy := doc["outbounds"].([]any)[0].(map[string]any)
	ss := proxy["streamSettings"].(map[string]any)
	tls := ss["tlsSettings"].(map[string]any)
	if tls["serverName"] != "edge.sni" || tls["fingerprint"] != "firefox" || tls["allowInsecure"] != nil || tls["pinnedPeerCertSha256"] != strings.Repeat("11", 32) || ss["sockopt"].(map[string]any)["tcpFastOpen"] != true || proxy["mux"].(map[string]any)["concurrency"] != float64(4) {
		t.Fatalf("JSON host overrides missing: %s", body)
	}
	if header != "upload=11; download=22; total=999; expire=0" {
		t.Fatalf("incorrect usage: %s", header)
	}
	t.Run("pinned core accepts Host JSON", func(t *testing.T) {
		if runtime.GOOS != "linux" {
			t.Skip("pinned Linux core check")
		}
		core, err := filepath.Abs(filepath.Join("..", "corebundle", "core", runtime.GOARCH, "xray"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(core); err != nil {
			t.Skip("build the pinned core before running its config validation")
		}
		file := filepath.Join(t.TempDir(), "host-client.json")
		if err := os.WriteFile(file, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		assets, _ := filepath.Abs(filepath.Join("..", "corebundle", "core"))
		cmd := exec.Command(core, "run", "-test", "-config", file)
		cmd.Env = append(os.Environ(), "XRAY_LOCATION_ASSET="+assets)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("pinned core rejected Host profile: %v\n%s", err, output)
		}
	})
	body, _, err = NewSubClashService(subSvc()).GetClash("bound-sub", "panel.example")
	if err != nil {
		t.Fatal(err)
	}
	var clash struct {
		Proxies []map[string]any `yaml:"proxies"`
	}
	if err := yaml.Unmarshal([]byte(body), &clash); err != nil {
		t.Fatal(err)
	}
	if len(clash.Proxies) != 1 || clash.Proxies[0]["server"] != "edge.example" || clash.Proxies[0]["ip-version"] != "ipv4-prefer" || clash.Proxies[0]["skip-cert-verify"] != true {
		t.Fatalf("Clash host missing: %s", body)
	}
	client["hostRuleIds"] = []int{h.Id}
	settings, _ := json.Marshal(map[string]any{"clients": []any{client}})
	if err := database.GetDB().Model(ib).Update("settings", string(settings)).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(h).Update("is_disabled", true).Error; err != nil {
		t.Fatal(err)
	}
	links, _, usage, err := subSvc().GetSubs("bound-sub", "panel.example")
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 0 || usage.Up != 11 || usage.Down != 22 || usage.Total != 999 {
		t.Fatalf("disabled binding fell back or lost accounting: %v %#v", links, usage)
	}
	body, header, err = jsonService.GetJson("bound-sub", "panel.example")
	if err != nil {
		t.Fatal(err)
	}
	if body != "[]" || header != "upload=11; download=22; total=999; expire=0" {
		t.Fatalf("closed JSON binding leaked an endpoint or lost quota: %s %s", body, header)
	}
	body, header, err = NewSubClashService(subSvc()).GetClash("bound-sub", "panel.example")
	if err != nil {
		t.Fatal(err)
	}
	clash.Proxies = nil
	if err := yaml.Unmarshal([]byte(body), &clash); err != nil {
		t.Fatal(err)
	}
	if len(clash.Proxies) != 0 || header != "upload=11; download=22; total=999; expire=0" {
		t.Fatalf("closed Clash binding leaked an endpoint or lost quota: %s %s", body, header)
	}
}
