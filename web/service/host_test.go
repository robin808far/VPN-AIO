package service

import (
	"encoding/json"
	"net/netip"
	"reflect"
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v2/database"
	"github.com/mhsanaei/3x-ui/v2/database/model"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"gorm.io/gorm"
)

func TestHostsSharedGroupCannotBeChangedThroughOneGrantedInbound(t *testing.T) {
	newInboundDB(t)
	db := database.GetDB()
	root := &model.User{IsSuperAdmin: true, Enable: true}
	admin := &model.User{Username: "hosts-admin", Enable: true, Permissions: model.PermAccessInbounds | model.PermEditInbound}
	if err := db.Create(admin).Error; err != nil {
		t.Fatal(err)
	}
	a := &model.Inbound{UserId: 1, Port: 41951, Tag: "host-owner-a", Protocol: model.VLESS, Settings: `{"clients":[]}`}
	b := &model.Inbound{UserId: 1, Port: 41952, Tag: "host-owner-b", Protocol: model.VLESS, Settings: `{"clients":[]}`}
	for _, ib := range []*model.Inbound{a, b} {
		if err := db.Create(ib).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&model.InboundAccess{UserId: admin.Id, InboundId: a.Id}).Error; err != nil {
		t.Fatal(err)
	}
	svc := HostService{}
	rows, err := svc.Save(root, "", &model.HostGroup{Host: model.Host{Remark: "shared", Security: "same"}, InboundIds: []int{a.Id, b.Id}, Hosts: []string{"cdn.example:443"}})
	if err != nil {
		t.Fatal(err)
	}
	before := rows[0].GroupId
	if _, err := svc.Save(admin, before, &model.HostGroup{Host: model.Host{Remark: "stolen"}, InboundIds: []int{a.Id}}); err == nil {
		t.Fatal("partial grant changed a shared group")
	}
	for _, action := range []string{"enable", "delete", "reorder"} {
		if err := svc.Mutate(admin, []string{before}, action, false); err == nil {
			t.Fatalf("partial grant allowed %s", action)
		}
	}
	groups, err := svc.List(admin)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || groups[0].Editable || !reflect.DeepEqual(groups[0].InboundIds, []int{a.Id}) || groups[0].Remark != "shared" {
		t.Fatalf("scoped groups leaked or changed: %#v", groups)
	}
	var remaining []model.Host
	if err := db.Where("group_id = ?", before).Find(&remaining).Error; err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 2 || remaining[0].Remark != "shared" || remaining[0].IsDisabled {
		t.Fatalf("denied operation partially applied: %#v", remaining)
	}
}

func TestHostsGroupEditDoesNotStealAnUnchangedEndpointBinding(t *testing.T) {
	newInboundDB(t)
	db := database.GetDB()
	ib := &model.Inbound{UserId: 1, Port: 41958, Tag: "host-match", Protocol: model.VLESS}
	if err := db.Create(ib).Error; err != nil {
		t.Fatal(err)
	}
	svc := HostService{}
	root := &model.User{IsSuperAdmin: true, Enable: true}
	request := &model.HostGroup{Host: model.Host{Remark: "group"}, InboundIds: []int{ib.Id}, Hosts: []string{"a.example:443", "b.example:443"}}
	rows, err := svc.Save(root, "", request)
	if err != nil {
		t.Fatal(err)
	}
	aID, group := rows[0].Id, rows[0].GroupId
	request.Hosts = []string{"c.example:443", "a.example:443"}
	rows, err = svc.Save(root, group, request)
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range rows {
		if h.Address == "a.example" && h.Id != aID {
			t.Fatalf("replacement endpoint stole unchanged binding %d: %#v", aID, rows)
		}
	}
	var kept model.Host
	if err := db.First(&kept, aID).Error; err != nil {
		t.Fatal(err)
	}
	if kept.Address != "a.example" {
		t.Fatalf("binding %d now selects %s", aID, kept.Address)
	}
}

func TestHostsWireguardConfigUsesIPv6EndpointAndKeepsDeviceKeys(t *testing.T) {
	newInboundDB(t)
	server, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	client, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	settings, _ := json.Marshal(map[string]any{"pubKey": server.PublicKey().String(), "clients": []any{map[string]any{"email": "host-wg", "devices": []any{map[string]any{"privKey": client.String(), "pubKey": client.PublicKey().String()}}, "addresses": []string{"10.8.1.2/32"}}}})
	ib := &model.Inbound{UserId: 1, Port: 41959, Tag: "host-ipv6", Protocol: model.WireGuard, Settings: string(settings)}
	if err := database.GetDB().Create(ib).Error; err != nil {
		t.Fatal(err)
	}
	h := &model.Host{GroupId: "wg-ipv6", InboundId: ib.Id, Remark: "IPv6", Address: "2001:db8::20", Port: 8443, Security: "same"}
	if err := database.GetDB().Create(h).Error; err != nil {
		t.Fatal(err)
	}
	configs, err := RenderWireguardXrayConfigs(ib, "host-wg", "fallback.example")
	if err != nil {
		t.Fatal(err)
	}
	if len(configs) != 1 {
		t.Fatalf("configs = %d, want one", len(configs))
	}
	for _, value := range []string{"PrivateKey = " + client.String(), "PublicKey = " + server.PublicKey().String(), "Address = 10.8.1.2/32", "Endpoint = [2001:db8::20]:8443"} {
		if !strings.Contains(configs[0].Config, value) {
			t.Fatalf("config lost %q: %s", value, configs[0].Config)
		}
	}
	if ib.Settings != string(settings) {
		t.Fatal("Host export rewrote device credentials")
	}
}

func TestHostsImportedBindingsAreRemappedAndUnknownBindingsStayClosed(t *testing.T) {
	newInboundDB(t)
	db := database.GetDB()
	ib := &model.Inbound{UserId: 1, Port: 41955, Tag: "host-import", Protocol: model.VLESS, Settings: `{"clients":[{"email":"selected","id":"11111111-2222-3333-4444-555555555555","hostRuleIds":[500]},{"email":"deleted","hostRuleIds":[501]}]}`,
		Hosts: []*model.Host{{Id: 500, GroupId: "source-group", Remark: "imported", Address: "import.example", Port: 443}}, HostRules: map[string][]int{"selected": {500}, "deleted": {501}}}
	if err := db.Create(ib).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Transaction(func(tx *gorm.DB) error { return importInboundHosts(tx, ib) }); err != nil {
		t.Fatal(err)
	}
	var hosts []model.Host
	if err := db.Where("inbound_id = ?", ib.Id).Find(&hosts).Error; err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 1 || hosts[0].Id == 500 {
		t.Fatalf("import reused source row identity: %#v", hosts)
	}
	var settings struct {
		Clients []model.Client `json:"clients"`
	}
	if err := json.Unmarshal([]byte(ib.Settings), &settings); err != nil {
		t.Fatal(err)
	}
	if len(settings.Clients) != 2 || !reflect.DeepEqual(settings.Clients[0].HostRuleIds, []int{hosts[0].Id}) {
		t.Fatalf("selected Host was not remapped: %s", ib.Settings)
	}
	svc := HostService{}
	renders, err := svc.Renders([]*model.Inbound{ib}, "raw", "panel.example")
	if err != nil {
		t.Fatal(err)
	}
	if len(renders) != 1 || !renders[0].Allows(settings.Clients[0]) || renders[0].Allows(settings.Clients[1]) {
		t.Fatal("imported deleted Host binding fell back")
	}
}

func TestHostsSwitchingRealityToTLSDoesNotExportServerPrivateKey(t *testing.T) {
	ib := &model.Inbound{Protocol: model.VLESS, Port: 443, Settings: `{"clients":[]}`, StreamSettings: `{"security":"reality","realitySettings":{"privateKey":"server-private-secret","serverNames":["base.example"],"settings":{"publicKey":"public"}},"network":"tcp"}`}
	view, err := ProjectHost(ib, &model.Host{Security: "tls", Address: "cdn.example", Sni: "cdn.example"}, "fallback.example")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(view.StreamSettings, "server-private-secret") || strings.Contains(view.StreamSettings, "realitySettings") {
		t.Fatalf("Host export leaked server Reality settings: %s", view.StreamSettings)
	}
	if !strings.Contains(ib.StreamSettings, "server-private-secret") {
		t.Fatal("projection destroyed the server's key")
	}
}

func TestHostsPlaintextExportRemovesVisionFlowFromDetachedClient(t *testing.T) {
	ib := &model.Inbound{Protocol: model.VLESS, Port: 443, Settings: `{"flow":"xtls-rprx-vision","clients":[{"id":"11111111-2222-3333-4444-555555555555","flow":"xtls-rprx-vision"}]}`, StreamSettings: `{"network":"tcp","security":"reality","realitySettings":{"privateKey":"server-private"}}`}
	before := ib.Settings
	view, err := ProjectHost(ib, &model.Host{Security: "none", Address: "relay.example"}, "fallback.example")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(view.Settings, "xtls-rprx-vision") || strings.Contains(view.StreamSettings, "server-private") {
		t.Fatalf("plaintext Host carried TLS-only state: %s %s", view.Settings, view.StreamSettings)
	}
	if ib.Settings != before {
		t.Fatal("Host removed the runtime account's flow")
	}
}

func TestHostsAccountRulesSurvivePartialWritesAndCanBeCleared(t *testing.T) {
	svc := newAccountsDB(t)
	vless := seedInboundWithClients(t, model.VLESS, 41956, []map[string]any{{"id": "11111111-2222-3333-4444-555555555555", "email": "host-account", "enable": true, "hostRuleIds": []int{73}}})
	trojan := seedInboundWithClients(t, model.Trojan, 41957, []map[string]any{{"password": "keep-password", "email": "host-account", "enable": true, "hostRuleIds": []int{73}}})
	svc.MigrationAccounts()
	account, err := svc.GetAccountByEmail("host-account")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(account.HostRuleIds, []int{73}) {
		t.Fatalf("migration lost Host rules: %#v", account.HostRuleIds)
	}
	for _, clear := range []bool{false, true} {
		entry := map[string]any{"id": "11111111-2222-3333-4444-555555555555", "email": "host-account", "enable": true}
		if clear {
			entry["hostRuleIds"] = []int{}
		}
		writeClients(t, vless.Id, []map[string]any{entry})
		if _, err := svc.ApplyMemberships("host-account", []int{vless.Id}, nil, false); err != nil {
			t.Fatal(err)
		}
		for _, id := range []int{vless.Id, trojan.Id} {
			got := clientEntry(t, id, "host-account")["hostRuleIds"]
			if clear && got != nil {
				t.Fatalf("Host selection could not be cleared: %v", got)
			}
			if !clear && !reflect.DeepEqual(got, []any{float64(73)}) {
				t.Fatalf("partial edit erased Host selection: %v", got)
			}
		}
	}
}

func TestHostsEditsKeepBindingIDsAndDeletionFailsClosed(t *testing.T) {
	newInboundDB(t)
	db := database.GetDB()
	root := &model.User{IsSuperAdmin: true, Enable: true}
	ib := &model.Inbound{UserId: 1, Port: 41953, Tag: "host-bindings", Protocol: model.VLESS, Settings: `{"clients":[]}`}
	if err := db.Create(ib).Error; err != nil {
		t.Fatal(err)
	}
	svc := HostService{}
	request := &model.HostGroup{Host: model.Host{Remark: "edge"}, InboundIds: []int{ib.Id}, Hosts: []string{"one.example:443"}}
	rows, err := svc.Save(root, "", request)
	if err != nil {
		t.Fatal(err)
	}
	id, group := rows[0].Id, rows[0].GroupId
	request.Hosts = []string{"two.example:8443"}
	rows, err = svc.Save(root, group, request)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Id != id || rows[0].Address != "two.example" {
		t.Fatalf("edit replaced binding identity: %#v", rows)
	}
	if err := svc.Mutate(root, []string{group}, "delete", false); err != nil {
		t.Fatal(err)
	}
	renders, err := svc.Renders([]*model.Inbound{ib}, "raw", "panel.example")
	if err != nil {
		t.Fatal(err)
	}
	bound := model.Client{Email: "bound", HostRuleIds: []int{id}}
	if len(renders) != 1 || renders[0].Allows(bound) || !renders[0].Allows(model.Client{Email: "default"}) {
		t.Fatalf("deleted binding fell back to the public endpoint: %#v", renders)
	}
}

func TestHostsCIDRAndProjectionPreserveCredentialsAndServerConfig(t *testing.T) {
	for _, cidr := range []string{"198.51.100.0/29", "2001:db8::/125", "203.0.113.9/32", "2001:db8::1/128"} {
		prefix := netip.MustParsePrefix(cidr)
		ips, err := randomHostIPs(prefix, 100)
		if err != nil {
			t.Fatal(err)
		}
		want := 6
		if prefix.Addr().Is6() {
			want = 7
		}
		if prefix.Bits() == prefix.Addr().BitLen() {
			want = 1
		}
		if len(ips) != want {
			t.Fatalf("%s gave %d endpoints, want %d", cidr, len(ips), want)
		}
		seen := map[string]bool{}
		for _, ip := range ips {
			if seen[ip] || !prefix.Contains(netip.MustParseAddr(ip)) {
				t.Fatalf("invalid CIDR sample %q", ips)
			}
			seen[ip] = true
		}
	}
	ib := &model.Inbound{Port: 443, Protocol: model.VLESS, Settings: `{"clients":[{"email":"a","id":"11111111-2222-3333-4444-555555555555"}],"unknown":{"keep":true}}`, StreamSettings: `{"network":"ws","security":"tls","wsSettings":{"path":"/old"},"tlsSettings":{"serverName":"server.example","certificates":[{"certificateFile":"keep.pem"}]}}`}
	before := *ib
	view, err := ProjectHost(ib, &model.Host{Address: "cdn.example", Port: 8443, Security: "same", Sni: "client.example", Path: "/new"}, "fallback.example")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*ib, before) || view.Listen != "cdn.example" || view.Port != 8443 {
		t.Fatal("projection changed the original inbound")
	}
	var settings map[string]any
	if err := json.Unmarshal([]byte(view.Settings), &settings); err != nil {
		t.Fatal(err)
	}
	if settings["clients"].([]any)[0].(map[string]any)["id"] != "11111111-2222-3333-4444-555555555555" || settings["unknown"].(map[string]any)["keep"] != true {
		t.Fatalf("projection changed credentials or unknown keys: %s", view.Settings)
	}
}

func TestHostsRejectMalformedGroupsWithoutWritingRows(t *testing.T) {
	newInboundDB(t)
	db := database.GetDB()
	ib := &model.Inbound{UserId: 1, Port: 41954, Tag: "host-validation", Protocol: model.VLESS}
	if err := db.Create(ib).Error; err != nil {
		t.Fatal(err)
	}
	svc := HostService{}
	root := &model.User{IsSuperAdmin: true, Enable: true}
	for _, address := range []string{"https://bad.example/path", "good.example:70000", "good.example\nother.example", "[bad:ipv6]:443"} {
		_, err := svc.Save(root, "", &model.HostGroup{Host: model.Host{Remark: "bad"}, InboundIds: []int{ib.Id}, Hosts: []string{address}})
		if err == nil {
			t.Fatalf("accepted address %q", address)
		}
	}
	var count int64
	if err := db.Model(&model.Host{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("rejected requests wrote %d hosts", count)
	}
}
