package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v2/database"
	"github.com/mhsanaei/3x-ui/v2/database/model"
)

func TestSingboxGroupsResolveManagedShadowTLSDetour(t *testing.T) {
	target := &model.Inbound{
		Id: 10, Tag: "inbound-127.0.0.1:443", Core: "singbox", Protocol: model.Shadowsocks, Enable: true,
		Settings: `{"type":"shadowsocks","users":[{"name":"target@example.test","password":"target-secret"}]}`,
	}
	front := &model.Inbound{
		Id: 11, Tag: "inbound-127.0.0.1:8443", Core: "singbox", Protocol: "shadowtls", Enable: true,
		Settings: `{"type":"shadowtls","version":3,"detour":"inbound-127.0.0.1:443","users":[{"name":"front@example.test","password":"front-secret"}]}`,
	}

	groups, err := singboxGroupsFromInbounds([]*model.Inbound{front, target})
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || len(groups[0].Members) != 2 || groups[0].Root.Id != target.Id {
		t.Fatalf("groups = %#v, want one group rooted at target", groups)
	}
	if got := singboxGroupIDs(groups[0]); len(got) != 2 || got[0] != target.Id || got[1] != front.Id {
		t.Fatalf("group ids = %v", got)
	}

	target.Tag = "inbound-127.0.0.1:9443"
	groups, err = singboxGroupsFromInbounds([]*model.Inbound{front, target})
	if err != nil || len(groups) != 1 || groups[0].Root.Id != target.Id {
		t.Fatalf("stale-tag relation was not recovered: groups=%#v err=%v", groups, err)
	}
}

func TestSingboxGroupsRejectInvalidDetourGraph(t *testing.T) {
	base := func(id int, tag string) *model.Inbound {
		return &model.Inbound{Id: id, Tag: tag, Core: "singbox", Protocol: model.Shadowsocks,
			Settings: `{"type":"shadowsocks","users":[{"name":"user` + string(rune('a'+id)) + `@example.test","password":"secret"}]}`}
	}
	for _, tc := range []struct {
		name string
		rows []*model.Inbound
		want string
	}{
		{name: "missing target", rows: []*model.Inbound{{Id: 1, Tag: "front", Core: "singbox", Protocol: "shadowtls", Settings: `{"type":"shadowtls","version":3,"detour":"missing","users":[{"name":"front@example.test","password":"secret"}]}`}}, want: "not found"},
		{name: "self", rows: []*model.Inbound{{Id: 1, Tag: "front", Core: "singbox", Protocol: "shadowtls", Settings: `{"type":"shadowtls","version":3,"detour":"front","users":[{"name":"front@example.test","password":"secret"}]}`}}, want: "itself"},
		{name: "cycle", rows: func() []*model.Inbound {
			a := base(1, "a")
			a.Protocol = "shadowtls"
			a.Settings = `{"type":"shadowtls","version":3,"detour":"b","users":[{"name":"a@example.test","password":"secret"}]}`
			b := base(2, "b")
			b.Protocol = "shadowtls"
			b.Settings = `{"type":"shadowtls","version":3,"detour":"a","users":[{"name":"b@example.test","password":"secret"}]}`
			return []*model.Inbound{a, b}
		}(), want: "cycle"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := singboxGroupsFromInbounds(tc.rows)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestSingboxMigrationBackfillsStableDetourID(t *testing.T) {
	singboxAuditSetup(t)
	db := database.GetDB()
	target := &model.Inbound{
		Core: "singbox", Protocol: model.Shadowsocks, Tag: "target-current",
		Settings: `{"type":"shadowsocks","users":[{"name":"target@example.test","password":"target-secret"}]}`,
	}
	frontSettings := `{"type":"shadowtls","version":3,"detour":"target-current","users":[{"name":"front@example.test","password":"front-secret"}]}`
	front := &model.Inbound{Core: "singbox", Protocol: "shadowtls", Tag: "front-current", Settings: frontSettings}
	if err := db.Create(target).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(front).Error; err != nil {
		t.Fatal(err)
	}

	updated, err := (&SingboxService{}).migrateDetourInboundIDs()
	if err != nil || updated != 1 {
		t.Fatalf("backfill updated=%d err=%v, want one relation", updated, err)
	}
	var saved model.Inbound
	if err := db.First(&saved, front.Id).Error; err != nil {
		t.Fatal(err)
	}
	if saved.SingboxDetourInboundID != target.Id || saved.Settings != frontSettings {
		t.Fatalf("backfilled inbound relation/settings = %d/%q", saved.SingboxDetourInboundID, saved.Settings)
	}
	updated, err = (&SingboxService{}).migrateDetourInboundIDs()
	if err != nil || updated != 0 {
		t.Fatalf("second backfill updated=%d err=%v, want idempotent no-op", updated, err)
	}
}

func TestSingboxRenderManagedDetourUsesCurrentTagsAndAllStatsUsers(t *testing.T) {
	singboxAuditSetup(t)
	target := &model.Inbound{
		Id: 10, Tag: "target-tag", SingboxAPIPort: 39001, Protocol: model.Shadowsocks,
		Settings: `{"type":"shadowsocks","users":[{"name":"target@example.test","password":"target-secret"}]}`,
	}
	front := &model.Inbound{
		Id: 11, Tag: "front-tag", Protocol: "shadowtls",
		Settings: `{"type":"shadowtls","version":3,"detour":"target-tag","users":[{"name":"front@example.test","password":"front-secret"}]}`,
	}
	content, err := (&SingboxService{}).renderConfigWithBase(target, []*model.Inbound{target, front}, DefaultSingboxBaseConfig)
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if err := json.Unmarshal(content, &config); err != nil {
		t.Fatal(err)
	}
	inbounds, ok := config["inbounds"].([]any)
	if !ok || len(inbounds) != 2 {
		t.Fatalf("inbounds = %#v", config["inbounds"])
	}
	var foundFront bool
	for _, raw := range inbounds {
		object := raw.(map[string]any)
		if object["tag"] == "front-tag" {
			foundFront = true
			if object["detour"] != "target-tag" {
				t.Fatalf("front detour = %#v, want target-tag", object["detour"])
			}
		}
	}
	if !foundFront {
		t.Fatal("rendered group omitted front inbound")
	}
	experimental := config["experimental"].(map[string]any)
	api := experimental["v2ray_api"].(map[string]any)
	stats := api["stats"].(map[string]any)
	if got := len(stats["inbounds"].([]any)); got != 2 {
		t.Fatalf("stats inbounds = %d, want 2", got)
	}
	if got := len(stats["users"].([]any)); got != 2 {
		t.Fatalf("stats users = %d, want 2", got)
	}
}
