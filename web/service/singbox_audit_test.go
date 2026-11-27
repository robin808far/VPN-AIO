package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v2/database"
	"github.com/mhsanaei/3x-ui/v2/database/model"
	"github.com/mhsanaei/3x-ui/v2/xray"
	"gorm.io/gorm"
)

func singboxAuditSetup(t *testing.T) {
	t.Helper()
	newNodeTestDB(t)
	previousRoot := singboxConfigRoot
	singboxConfigRoot = filepath.Join(t.TempDir(), "fresh", "singbox")
	t.Cleanup(func() { singboxConfigRoot = previousRoot })
	dir := t.TempDir()
	bin := filepath.Join(dir, "sing-box")
	if exact := os.Getenv("VPN_AIO_TEST_SINGBOX"); exact != "" {
		if err := os.Symlink(exact, bin); err != nil {
			t.Fatal(err)
		}
	} else {
		// An executable fixture pins error/transaction handling without requiring
		// the optional daemon. CI can also run every case with the bundled core.
		script := "#!/bin/sh\ncase \"$1\" in\ncheck) if grep -q review_invalid \"$3\"; then echo invalid-config; exit 1; fi;;\nversion) echo 'sing-box version 1.12.14';;\nrun) if grep -q review_run_fail \"$3\"; then echo failed-to-bind; exit 1; fi; exec sleep 600;;\nesac\n"
		if err := os.WriteFile(bin, []byte(script), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func singboxAuditInbound(t *testing.T) *model.Inbound {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return &model.Inbound{Core: "singbox", Protocol: model.VLESS, Enable: true,
		Listen: "127.0.0.1", Port: port, Tag: fmt.Sprintf("fixture-%d", port), TrafficReset: "never",
		Settings: fmt.Sprintf(`{"type":"vless","listen":"127.0.0.1","listen_port":%d,"users":[{"name":"alice-%d","uuid":"11111111-1111-4111-8111-%012x"}]}`, port, port, port)}
}

func TestSingboxAuditResellerCannotReadRawCredentials(t *testing.T) {
	for _, core := range []string{"", "singbox"} {
		inbound := &model.Inbound{Core: core, Protocol: "shadowtls",
			Settings: `{"users":[{"name":"alice","password":"private-credential"}],"password":"shared-secret","tls":{"private_key":"server-key"}}`}
		(&InboundService{}).FilterInboundForReseller(inbound, map[string]bool{"alice": true})
		for _, secret := range []string{"private-credential", "shared-secret", "server-key"} {
			if strings.Contains(inbound.Settings, secret) {
				t.Fatalf("reseller received %s", secret)
			}
		}
	}
}

func TestSingboxAuditRejectsUnsupportedPoliciesBeforePersistence(t *testing.T) {
	singboxAuditSetup(t)
	for _, policy := range []string{"multiplier", "clients"} {
		t.Run(policy, func(t *testing.T) {
			inbound := singboxAuditInbound(t)
			switch policy {
			case "multiplier":
				inbound.TrafficMultiplierEnable = true
			case "clients":
				inbound.Settings = `{"type":"vless","clients":[{"email":"alice","totalGB":1024}]}`
			}
			if _, _, err := (&InboundService{}).AddInbound(inbound); err == nil {
				t.Fatal("unsupported policy accepted")
			}
		})
	}
	var count int64
	if err := database.GetDB().Model(&model.Inbound{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("refused saves left %d inbounds: %v", count, err)
	}
}

func TestSingboxAuditPublishesSpeedIPAndStreamPolicies(t *testing.T) {
	singboxAuditSetup(t)
	inbound := singboxAuditInbound(t)
	inbound.SpeedLimitEnable = true
	inbound.SpeedLimitSeparate = true
	inbound.SpeedLimitDown = 1024
	inbound.SpeedLimitUp = 512
	inbound.IPLimit = 2
	inbound.IPLimitStrategy = "reject"
	saved, _, err := (&InboundService{}).AddInbound(inbound)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = procMgr.Stop(singboxProcName(saved.Id)) })
	clients, err := (&InboundService{}).GetClients(saved)
	if err != nil || len(clients) != 1 {
		t.Fatalf("clients: %d, %v", len(clients), err)
	}
	down, up, streams := 512, 256, 4
	account, err := (&AccountService{}).GetAccountByEmail(clients[0].Email)
	if err != nil || account == nil {
		t.Fatalf("account: %+v, %v", account, err)
	}
	account.SpeedLimitDown = &down
	account.SpeedLimitUp = &up
	account.LimitIP = 3
	account.StreamLimit = streams
	if err := database.GetDB().Save(account).Error; err != nil {
		t.Fatal(err)
	}
	policies := loadSpeedLimitPolicies()
	if len(policies) != 1 {
		t.Fatalf("policies = %d, want 1", len(policies))
	}
	data, err := speedLimitDocument(policies, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var document speedLimitDoc
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if len(document.Users) != 1 {
		t.Fatalf("users = %d, want 1", len(document.Users))
	}
	user := document.Users[0]
	if user.DownBps != 512*bytesPerKB || user.UpBps != 256*bytesPerKB || user.IPLimit != 3 || user.StreamLimit != streams {
		t.Fatalf("published policy = %+v", user)
	}
}

func TestSingboxAuditRejectsPhysicalDeviceLimit(t *testing.T) {
	limit := 2
	account := &model.Account{Email: "device-cap@example.test", UserLimitOverride: &limit}
	if err := validateSingboxAccount(account); err == nil || !strings.Contains(err.Error(), "cannot count physical devices") {
		t.Fatalf("physical device limit validation error = %v", err)
	}
}

func TestSingboxAuditShadowTLSManagedUsersRequireVersionThree(t *testing.T) {
	singboxAuditSetup(t)
	svc := &InboundService{}
	for _, tc := range []struct {
		name    string
		version string
		hasUser bool
	}{
		{name: "version 1", version: "1", hasUser: true},
		{name: "version 2", version: "2", hasUser: true},
		{name: "missing version", hasUser: true},
		{name: "string version", version: `"3"`, hasUser: true},
		{name: "fractional version", version: "3.0", hasUser: true},
		{name: "version 2 empty users", version: "2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			port := listener.Addr().(*net.TCPAddr).Port
			_ = listener.Close()

			versionField := ""
			if tc.version != "" {
				versionField = `"version":` + tc.version + ","
			}
			users := "[]"
			if tc.hasUser {
				users = fmt.Sprintf(`[{"name":"alice-%d@example.test","password":"managed-secret"}]`, port)
			}
			inbound := &model.Inbound{
				Core: "singbox", Protocol: "shadowtls", Listen: "127.0.0.1", Port: port,
				TrafficReset: "never",
				Settings:     fmt.Sprintf(`{"type":"shadowtls","listen":"127.0.0.1","listen_port":%d,%s"users":%s,"handshake":{"server":"example.test","server_port":443}}`, port, versionField, users),
			}
			if _, _, err := svc.AddInbound(inbound); err == nil || !strings.Contains(err.Error(), "managed accounts require version 3") {
				t.Fatalf("version %q was not rejected by managed-account validation: %v", tc.version, err)
			}
		})
	}

	var count int64
	if err := database.GetDB().Model(&model.Inbound{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("rejected ShadowTLS configs left %d inbounds: %v", count, err)
	}

	valid := &model.Inbound{Core: "singbox", Protocol: "shadowtls",
		Settings: `{"type":"shadowtls","version":3,"users":[{"name":"alice","password":"managed-secret"}]}`}
	if err := validateSingboxLimits(valid); err != nil {
		t.Fatalf("version 3 managed users were rejected: %v", err)
	}
}

func TestSingboxAuditSupportsInboundExpiryAndTrafficReset(t *testing.T) {
	singboxAuditSetup(t)
	svc := &InboundService{}

	expired := singboxAuditInbound(t)
	expired.ExpiryTime = time.Now().Add(-time.Minute).UnixMilli()
	savedExpired, _, err := svc.AddInbound(expired)
	if err != nil {
		t.Fatalf("expired inbound policy rejected: %v", err)
	}
	if savedExpired.Enable {
		t.Fatal("expired inbound remained enabled after save")
	}
	storedExpired, err := svc.GetInbound(savedExpired.Id)
	if err != nil || storedExpired.Enable {
		t.Fatalf("expired inbound state not persisted: %+v, %v", storedExpired, err)
	}

	reset := singboxAuditInbound(t)
	reset.TrafficReset = "daily"
	savedReset, _, err := svc.AddInbound(reset)
	if err != nil {
		t.Fatalf("daily traffic reset policy rejected: %v", err)
	}
	t.Cleanup(func() { _ = procMgr.Stop(singboxProcName(savedReset.Id)) })
	if err := database.GetDB().Model(&model.Inbound{}).Where("id = ?", savedReset.Id).
		Updates(map[string]any{"up": 512, "down": 256}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(&xray.ClientTraffic{}).Where("inbound_id = ?", savedReset.Id).
		Updates(map[string]any{"up": 512, "down": 256}).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.ResetInboundTraffic(savedReset.Id); err != nil {
		t.Fatal(err)
	}
	if err := svc.ResetAllClientTraffics(savedReset.Id); err != nil {
		t.Fatal(err)
	}
	var stored model.Inbound
	if err := database.GetDB().First(&stored, savedReset.Id).Error; err != nil || stored.Up != 0 || stored.Down != 0 {
		t.Fatalf("inbound traffic reset failed: %+v, %v", stored, err)
	}
	var traffic xray.ClientTraffic
	if err := database.GetDB().Where("inbound_id = ?", savedReset.Id).First(&traffic).Error; err != nil || traffic.Up != 0 || traffic.Down != 0 {
		t.Fatalf("client traffic reset failed: %+v, %v", traffic, err)
	}
	inbounds, err := svc.GetInboundsByTrafficReset("daily")
	if err != nil || len(inbounds) != 1 || inbounds[0].Id != savedReset.Id {
		t.Fatalf("daily reset scheduler cannot find sing-box inbound: %+v, %v", inbounds, err)
	}
}

func TestSingboxAuditCommittedSaveAndInvalidUpdatePreserveRuntime(t *testing.T) {
	singboxAuditSetup(t)
	svc := &InboundService{}
	inbound, _, err := svc.AddInbound(singboxAuditInbound(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = procMgr.Stop(singboxProcName(inbound.Id)) })
	path := filepath.Join((&SingboxService{}).configDir(inbound.Id), "config.json")
	working, err := os.ReadFile(path)
	if err != nil || !procMgr.IsRunning(singboxProcName(inbound.Id)) {
		t.Fatalf("save did not start committed row: %v", err)
	}
	stored, err := svc.GetInbound(inbound.Id)
	if err != nil || stored.Settings != inbound.Settings {
		t.Fatalf("wrong committed row: %v", err)
	}
	other, _, err := svc.AddInbound(singboxAuditInbound(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = procMgr.Stop(singboxProcName(other.Id)) })
	otherGen := procMgr.get(singboxProcName(other.Id)).gen
	updated := *inbound
	updated.Remark = "renamed"
	if _, _, err := svc.UpdateInbound(&updated); err != nil {
		t.Fatal(err)
	}
	if procMgr.get(singboxProcName(other.Id)).gen != otherGen {
		t.Fatal("unrelated sing-box process restarted")
	}
	updated.Settings = strings.TrimSuffix(updated.Settings, "}") + `,"review_invalid":true}`
	if _, _, err := svc.UpdateInbound(&updated); err == nil {
		t.Fatal("invalid update accepted")
	}
	current, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(current, working) {
		t.Fatal("invalid update replaced working file")
	}
	stored, err = svc.GetInbound(inbound.Id)
	if err != nil || stored.Settings != inbound.Settings {
		t.Fatal("invalid update reached DB")
	}
	if !procMgr.IsRunning(singboxProcName(inbound.Id)) {
		t.Fatal("invalid update stopped working process")
	}
	// A restart must return the actual check error and preserve a working file
	// even when invalid data entered through an external DB restore.
	if err := database.GetDB().Model(&model.Inbound{}).Where("id = ?", inbound.Id).Update("settings", updated.Settings).Error; err != nil {
		t.Fatal(err)
	}
	if err := (&SingboxService{}).RestartServices(); err == nil {
		t.Fatal("restart swallowed check failure")
	}
	current, err = os.ReadFile(path)
	if err != nil || !bytes.Equal(current, working) {
		t.Fatal("failed restart replaced working config")
	}
}

func TestSingboxAuditCommitFailureDoesNotStartService(t *testing.T) {
	singboxAuditSetup(t)
	if err := database.GetDB().Exec(`CREATE TRIGGER reject_singbox BEFORE INSERT ON inbounds BEGIN SELECT RAISE(ABORT, 'fixture commit failure'); END`).Error; err != nil {
		t.Fatal(err)
	}
	if _, _, err := (&InboundService{}).AddInbound(singboxAuditInbound(t)); err == nil {
		t.Fatal("failed transaction acknowledged")
	}
	files, err := filepath.Glob(filepath.Join(singboxBaseDir(), "server-*"))
	if err != nil || len(files) != 0 {
		t.Fatal("failed transaction generated runtime files")
	}
}

func TestSingboxAuditStartupFailureRestoresDatabaseAndProcess(t *testing.T) {
	if os.Getenv("VPN_AIO_TEST_SINGBOX") != "" {
		t.Skip("exercises an injected process-start failure with the executable fixture")
	}
	singboxAuditSetup(t)
	svc := &InboundService{}
	inbound, _, err := svc.AddInbound(singboxAuditInbound(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = procMgr.Stop(singboxProcName(inbound.Id)) })
	path := filepath.Join((&SingboxService{}).configDir(inbound.Id), "config.json")
	working, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	updated := *inbound
	updated.Settings = strings.TrimSuffix(updated.Settings, "}") + `,"review_run_fail":true}`
	if _, _, err := svc.UpdateInbound(&updated); err == nil || !strings.Contains(err.Error(), "exited during startup") {
		t.Fatalf("startup failure not returned: %v", err)
	}
	stored, err := svc.GetInbound(inbound.Id)
	if err != nil || stored.Settings != inbound.Settings {
		t.Fatal("failed startup left new row in database")
	}
	current, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(working, current) || !procMgr.IsRunning(singboxProcName(inbound.Id)) {
		t.Fatal("failed startup did not restore working file and process")
	}
}

func TestSingboxAuditPortEditReconcilesCommittedListener(t *testing.T) {
	singboxAuditSetup(t)
	svc := &InboundService{}
	inbound, _, err := svc.AddInbound(singboxAuditInbound(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = procMgr.Stop(singboxProcName(inbound.Id)) })
	replacement := singboxAuditInbound(t)
	updated := *inbound
	updated.Settings = replacement.Settings
	if _, _, err := svc.UpdateInbound(&updated); err != nil {
		t.Fatal(err)
	}
	stored, err := svc.GetInbound(inbound.Id)
	if err != nil || stored.Port != replacement.Port {
		t.Fatal("listener port was not committed")
	}
	if os.Getenv("VPN_AIO_TEST_SINGBOX") != "" {
		conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", replacement.Port))
		if err != nil {
			t.Fatalf("updated listener is not running: %v", err)
		}
		conn.Close()
		if conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", inbound.Port)); err == nil {
			conn.Close()
			t.Fatal("old listener remains active")
		}
	}
}

func TestSingboxAuditFreshBaseSaveAndNullRejection(t *testing.T) {
	singboxAuditSetup(t)
	if _, err := mergeBaseConfig("null"); err == nil {
		t.Fatal("null base accepted")
	}
	if err := baseConfigService.SaveBaseConfig(DefaultSingboxBaseConfig); err != nil {
		t.Fatalf("fresh base save: %v", err)
	}
	files, err := filepath.Glob(filepath.Join(singboxBaseDir(), ".check-*.json"))
	if err != nil || len(files) != 0 {
		t.Fatal("temporary validation files leaked")
	}
}

func TestSingboxAuditInboundQuotaStopsListener(t *testing.T) {
	singboxAuditSetup(t)
	inbound := singboxAuditInbound(t)
	inbound.Total = 1024
	saved, _, err := (&InboundService{}).AddInbound(inbound)
	if err != nil {
		t.Fatalf("supported inbound quota rejected: %v", err)
	}
	t.Cleanup(func() { _ = procMgr.Stop(singboxProcName(saved.Id)) })
	if !procMgr.IsRunning(singboxProcName(saved.Id)) {
		t.Fatal("listener did not start before quota depletion")
	}
	if err := database.GetDB().Model(&model.Inbound{}).Where("id = ?", saved.Id).
		Updates(map[string]any{"up": 512, "down": 512}).Error; err != nil {
		t.Fatal(err)
	}
	if err := singboxService.RefreshServices(); err != nil {
		t.Fatal(err)
	}
	stored, err := (&InboundService{}).GetInbound(saved.Id)
	if err != nil || stored.Enable {
		t.Fatalf("depleted inbound was not disabled: %+v, %v", stored, err)
	}
	if procMgr.IsRunning(singboxProcName(saved.Id)) {
		t.Fatal("depleted inbound listener kept running")
	}
}

func TestSingboxAuditDepletedSweepPreservesSingboxQuota(t *testing.T) {
	singboxAuditSetup(t)
	svc := &InboundService{}
	inbound, _, err := svc.AddInbound(singboxAuditInbound(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = procMgr.Stop(singboxProcName(inbound.Id)) })
	clients, err := svc.GetClients(inbound)
	if err != nil || len(clients) != 1 {
		t.Fatalf("users: %d, %v", len(clients), err)
	}
	email := clients[0].Email
	if _, _, err := svc.updateSingboxAccount(inbound, email, func(account *model.Account) {
		account.TotalGB = 1024
	}); err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(&xray.ClientTraffic{}).Where("email = ?", email).
		Updates(map[string]any{"up": 1024, "enable": false}).Error; err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.BulkUpdateClients(BulkClientUpdateRequest{
		Op: "addTraffic", AmountBytes: 1024,
		Targets: []BulkClientTarget{{InboundId: inbound.Id, Email: email}},
	}); err == nil {
		t.Fatal("unsupported sing-box bulk edit was acknowledged")
	}
	if err := svc.DelDepletedClients(-1); err == nil {
		t.Fatal("depleted-user sweep silently removed the sing-box account")
	}
	if _, err := svc.GetInbound(inbound.Id); err != nil {
		t.Fatalf("depleted sweep deleted the inbound: %v", err)
	}
	var traffic xray.ClientTraffic
	if err := database.GetDB().Where("email = ?", email).First(&traffic).Error; err != nil || traffic.Total != 1024 || traffic.Up != 1024 {
		t.Fatalf("depleted sweep changed the quota history: %+v, %v", traffic, err)
	}
}

func TestSingboxAuditMixedBulkRefusalRollsBackEarlierTargets(t *testing.T) {
	singboxAuditSetup(t)
	xrayInbound := seedTaggedInbound(t, model.VLESS, 34471, []map[string]any{
		{"id": "11111111-1111-4111-8111-111111111111", "email": "xray-first@example.test", "enable": true, "totalGB": float64(1024)},
	})
	seedTrafficRow(t, xray.ClientTraffic{InboundId: xrayInbound.Id, Email: "xray-first@example.test", Total: 1024, Enable: true})
	singboxInbound, _, err := (&InboundService{}).AddInbound(singboxAuditInbound(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = procMgr.Stop(singboxProcName(singboxInbound.Id)) })

	_, _, err = (&InboundService{}).BulkUpdateClients(BulkClientUpdateRequest{
		Op: "addTraffic", AmountBytes: 512,
		Targets: []BulkClientTarget{
			{InboundId: xrayInbound.Id, Email: "xray-first@example.test"},
			{InboundId: singboxInbound.Id, Email: "alice-" + fmt.Sprint(singboxInbound.Port)},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "not supported for sing-box") {
		t.Fatalf("mixed bulk request was not refused: %v", err)
	}
	var stored model.Inbound
	if err := database.GetDB().First(&stored, xrayInbound.Id).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stored.Settings, `"totalGB":1024`) {
		t.Fatalf("earlier Xray target committed despite sing-box refusal: %s", stored.Settings)
	}
	var traffic xray.ClientTraffic
	if err := database.GetDB().Where("email = ?", "xray-first@example.test").First(&traffic).Error; err != nil || traffic.Total != 1024 {
		t.Fatalf("bulk refusal changed traffic quota: %+v, %v", traffic, err)
	}
}

func TestAccountMembershipAdmissionRefusesSecondIkev2PSKAccount(t *testing.T) {
	svc := newInboundDB(t)
	accounts := &AccountService{}
	source := seedTaggedInbound(t, model.VLESS, 34473, []map[string]any{testClient("new-account@example.test")})
	ikev2 := seedTaggedInbound(t, model.IKEV2, 34474, []map[string]any{testClient("existing@example.test")})
	var settings map[string]any
	if err := json.Unmarshal([]byte(ikev2.Settings), &settings); err != nil {
		t.Fatal(err)
	}
	settings["authMode"] = "psk"
	encoded, err := json.Marshal(settings)
	if err != nil {
		t.Fatal(err)
	}
	ikev2.Settings = string(encoded)
	if err := database.GetDB().Save(ikev2).Error; err != nil {
		t.Fatal(err)
	}
	for _, inbound := range []*model.Inbound{source, ikev2} {
		if err := accounts.SyncInboundAccounts(database.GetDB(), inbound.Id); err != nil {
			t.Fatalf("sync inbound %d: %v", inbound.Id, err)
		}
	}
	account, err := accounts.GetAccountByEmail("new-account@example.test")
	if err != nil || account == nil {
		t.Fatalf("load account: %+v, %v", account, err)
	}

	if _, err := accounts.ApplyMemberships(account.Email, []int{source.Id, ikev2.Id}, nil, true); err == nil || !strings.Contains(err.Error(), "allows only one account") {
		t.Fatalf("second PSK IKEv2 account was admitted: %v", err)
	}
	memberships, err := accounts.GetMemberships(account.Id)
	if err != nil || len(memberships) != 1 || memberships[0].InboundId != source.Id {
		t.Fatalf("failed membership left partial rows: %+v, %v", memberships, err)
	}
	clients, err := svc.GetClients(ikev2)
	if err != nil || len(clients) != 1 || clients[0].Email != "existing@example.test" {
		t.Fatalf("failed admission changed the PSK inbound: %+v, %v", clients, err)
	}
}

func TestSingboxAuditManualTrafficAndBulkResellerQuotaRefreshImmediately(t *testing.T) {
	singboxAuditSetup(t)
	svc := &InboundService{}
	inbound, _, err := svc.AddInbound(singboxAuditInbound(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = procMgr.Stop(singboxProcName(inbound.Id)) })
	clients, err := svc.GetClients(inbound)
	if err != nil || len(clients) != 1 {
		t.Fatalf("sing-box users: %d, %v", len(clients), err)
	}
	email := clients[0].Email
	if _, _, err := svc.updateSingboxAccount(inbound, email, func(account *model.Account) {
		account.TotalGB = 1024
	}); err != nil {
		t.Fatal(err)
	}
	if !procMgr.IsRunning(singboxProcName(inbound.Id)) {
		t.Fatal("listener did not start before the account quota was consumed")
	}
	if err := svc.UpdateClientTrafficByEmail(email, 1024, 0); err != nil {
		t.Fatalf("manual traffic update: %v", err)
	}
	if procMgr.IsRunning(singboxProcName(inbound.Id)) {
		t.Fatal("manual traffic update left an over-quota sing-box user connected")
	}
	if err := svc.UpdateClientTrafficByEmail(email, -1, 0); err == nil {
		t.Fatal("negative manual traffic counter was accepted")
	}
	var traffic xray.ClientTraffic
	if err := database.GetDB().Where("email = ?", email).First(&traffic).Error; err != nil || traffic.Up != 1024 {
		t.Fatalf("rejected negative counter changed usage: %+v, %v", traffic, err)
	}
	if err := svc.ResetAllClientTraffics(inbound.Id); err != nil {
		t.Fatalf("bulk traffic reset: %v", err)
	}
	if !procMgr.IsRunning(singboxProcName(inbound.Id)) {
		t.Fatal("bulk traffic reset did not restore the eligible sing-box user immediately")
	}

	// Reseller charge write-back reaches every account membership but native
	// sing-box users carry credentials only. The traffic row changes and matching
	// sing-box listeners must refresh after the transaction commits.
	xrayInbound := &model.Inbound{
		UserId: 1, Tag: "inbound-34472", Port: 34472, Protocol: model.VLESS,
		Enable:   false,
		Settings: fmt.Sprintf(`{"clients":[{"id":%q,"email":%q,"enable":true,"totalGB":1024}]}`, clients[0].ID, email),
	}
	if err := database.GetDB().Create(xrayInbound).Error; err != nil {
		t.Fatal(err)
	}
	if err := (&AccountService{}).SyncInboundAccounts(database.GetDB(), xrayInbound.Id); err != nil {
		t.Fatalf("sync shared Xray membership: %v", err)
	}
	if err := database.GetDB().Model(&xray.ClientTraffic{}).Where("email = ?", email).
		Updates(map[string]any{"up": 1024, "total": 1024, "enable": true}).Error; err != nil {
		t.Fatal(err)
	}
	if err := singboxService.RefreshAccount(email); err != nil {
		t.Fatal(err)
	}
	if procMgr.IsRunning(singboxProcName(inbound.Id)) {
		t.Fatal("quota-exhausted shared account remained connected before reseller top-up")
	}
	ticket := BulkTicket{Active: true, Charges: []BulkCharge{{Email: email, SetTotal: true, NewTotal: 2048}}}
	if err := (&ResellerService{}).ApplyBulkCharges(ticket); err != nil {
		t.Fatalf("apply reseller quota: %v", err)
	}
	if err := (&AccountService{}).SyncInboundAccounts(database.GetDB(), xrayInbound.Id); err != nil {
		t.Fatalf("sync topped-up account policy: %v", err)
	}
	if !procMgr.IsRunning(singboxProcName(inbound.Id)) {
		t.Fatal("reseller top-up did not refresh the shared sing-box membership")
	}
	if err := database.GetDB().Where("email = ?", email).First(&traffic).Error; err != nil || traffic.Total != 2048 {
		t.Fatalf("reseller quota did not reach client_traffics: %+v, %v", traffic, err)
	}
}

func TestSingboxAuditMigrationImportsNativeUsers(t *testing.T) {
	singboxAuditSetup(t)
	inbound := singboxAuditInbound(t)
	inbound.Enable = false
	if err := database.GetDB().Create(inbound).Error; err != nil {
		t.Fatal(err)
	}
	service := &AccountService{}
	needed, total, err := service.migrationNeeded(database.GetDB())
	if err != nil || !needed || total != 1 {
		t.Fatalf("migration preflight: needed=%v entries=%d err=%v", needed, total, err)
	}
	var report AccountsMigrationReport
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		return service.runAccountsPass(tx, &report, total)
	}); err != nil {
		t.Fatalf("migration pass: %v", err)
	}
	clients, err := (&InboundService{}).GetClients(inbound)
	if err != nil || len(clients) != 1 {
		t.Fatalf("native user readback: %d, %v", len(clients), err)
	}
	account, err := service.GetAccountByEmail(clients[0].Email)
	if err != nil || account == nil || account.UUID != clients[0].ID {
		t.Fatalf("managed account credential: %+v, %v", account, err)
	}
	memberships, err := service.GetMemberships(account.Id)
	if err != nil || len(memberships) != 1 || memberships[0].InboundId != inbound.Id {
		t.Fatalf("managed membership: %+v, %v", memberships, err)
	}
	var traffic xray.ClientTraffic
	if err := database.GetDB().Where("email = ?", account.Email).First(&traffic).Error; err != nil || traffic.InboundId != inbound.Id {
		t.Fatalf("managed traffic row: %+v, %v", traffic, err)
	}
}

func TestSingboxAuditCrossCorePolicyUpdateRequestsXrayRestart(t *testing.T) {
	singboxAuditSetup(t)
	svc := &InboundService{}
	singboxInbound, _, err := svc.AddInbound(singboxAuditInbound(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = procMgr.Stop(singboxProcName(singboxInbound.Id)) })
	clients, err := svc.GetClients(singboxInbound)
	if err != nil || len(clients) != 1 {
		t.Fatalf("sing-box users: %d, %v", len(clients), err)
	}

	xrayInbound := singboxAuditInbound(t)
	xrayInbound.Core, xrayInbound.Enable = "", false
	xrayInbound.Settings = fmt.Sprintf(`{"clients":[{"email":%q,"id":%q,"enable":true}]}`, clients[0].Email, clients[0].ID)
	if err := database.GetDB().Create(xrayInbound).Error; err != nil {
		t.Fatal(err)
	}
	if err := (&AccountService{}).SyncInboundAccounts(database.GetDB(), xrayInbound.Id); err != nil {
		t.Fatal(err)
	}
	needRestart, err := svc.ResetClientTrafficLimitByEmail(clients[0].Email, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !needRestart {
		t.Fatal("cross-core account policy update did not request Xray reconciliation")
	}
}

func TestSingboxAuditAccountPolicyAndQuotaRuntime(t *testing.T) {
	singboxAuditSetup(t)
	svc := &InboundService{}
	inbound, _, err := svc.AddInbound(singboxAuditInbound(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = procMgr.Stop(singboxProcName(inbound.Id)) })
	clients, err := svc.GetClients(inbound)
	if err != nil || len(clients) != 1 {
		t.Fatalf("initial users: %d, %v", len(clients), err)
	}
	email := clients[0].Email

	if _, err := svc.ResetClientTrafficLimitByEmail(email, 1); err != nil {
		t.Fatalf("set account quota: %v", err)
	}
	account, err := (&AccountService{}).GetAccountByEmail(email)
	if err != nil || account == nil || account.TotalGB != 1<<30 {
		t.Fatalf("account quota was not stored: %+v, %v", account, err)
	}
	var traffic xray.ClientTraffic
	if err := database.GetDB().Where("email = ?", email).First(&traffic).Error; err != nil || traffic.Total != 1<<30 || !traffic.Enable {
		t.Fatalf("traffic quota was not synchronized: %+v, %v", traffic, err)
	}

	var root map[string]any
	if err := json.Unmarshal([]byte(inbound.Settings), &root); err != nil {
		t.Fatal(err)
	}
	users := root["users"].([]any)
	user := users[0].(map[string]any)
	if user["uuid"] != clients[0].ID || user["totalGB"] != nil || user["expiryTime"] != nil {
		t.Fatalf("account policy leaked into native user: %#v", user)
	}

	if err := database.GetDB().Model(&xray.ClientTraffic{}).Where("email = ?", email).
		Update("up", int64(1<<30)).Error; err != nil {
		t.Fatal(err)
	}
	if err := singboxService.RefreshAccount(email); err != nil {
		t.Fatal(err)
	}
	if procMgr.IsRunning(singboxProcName(inbound.Id)) {
		t.Fatal("quota-depleted user kept the listener running")
	}
	if _, err := svc.ResetClientTraffic(inbound.Id, email); err != nil {
		t.Fatalf("reset traffic: %v", err)
	}
	if !procMgr.IsRunning(singboxProcName(inbound.Id)) {
		t.Fatal("traffic reset did not restore the listener")
	}

	if changed, _, err := svc.SetClientEnableByEmail(email, false); err != nil || !changed {
		t.Fatalf("disable account: changed=%v err=%v", changed, err)
	}
	if procMgr.IsRunning(singboxProcName(inbound.Id)) {
		t.Fatal("disabled account kept the listener running")
	}
	if changed, _, err := svc.SetClientEnableByEmail(email, true); err != nil || !changed {
		t.Fatalf("enable account: changed=%v err=%v", changed, err)
	}
	if !procMgr.IsRunning(singboxProcName(inbound.Id)) {
		t.Fatal("enabling the account did not restore the listener")
	}

	expiredAt := time.Now().Add(-time.Minute).UnixMilli()
	if _, err := svc.ResetClientExpiryTimeByEmail(email, expiredAt); err != nil {
		t.Fatalf("set expiry: %v", err)
	}
	if err := database.GetDB().Where("email = ?", email).First(&traffic).Error; err != nil || traffic.Enable {
		t.Fatalf("expired account remained enabled: %+v, %v", traffic, err)
	}
	if procMgr.IsRunning(singboxProcName(inbound.Id)) {
		t.Fatal("expired account kept the listener running")
	}
	if _, err := svc.ResetClientIpLimitByEmail(email, 1); err != nil {
		t.Fatalf("supported sing-box IP limit update was refused: %v", err)
	}
}

func TestSingboxAuditBundledRuntimeStatsAndQuota(t *testing.T) {
	exact := os.Getenv("VPN_AIO_TEST_SINGBOX")
	if exact == "" {
		t.Skip("requires the bundled sing-box build with with_v2ray_api")
	}
	singboxAuditSetup(t)
	svc := &InboundService{}
	inbound, _, err := svc.AddInbound(singboxAuditInbound(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = procMgr.Stop(singboxProcName(inbound.Id)) })
	clients, err := svc.GetClients(inbound)
	if err != nil || len(clients) != 1 {
		t.Fatalf("initial users: %d, %v", len(clients), err)
	}
	email := clients[0].Email
	if _, handled, err := svc.updateSingboxAccount(inbound, email, func(account *model.Account) {
		account.TotalGB = 1
	}); err != nil || !handled {
		t.Fatalf("set low quota: handled=%v err=%v", handled, err)
	}

	target, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	targetPort := target.Addr().(*net.TCPAddr).Port
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "runtime-ok")
	})}
	go func() { _ = server.Serve(target) }()
	t.Cleanup(func() { _ = server.Close() })

	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	socksPort := probe.Addr().(*net.TCPAddr).Port
	_ = probe.Close()
	clientConfig := map[string]any{
		"log": map[string]any{"level": "error"},
		"inbounds": []any{map[string]any{
			"type": "socks", "tag": "test-socks", "listen": "127.0.0.1", "listen_port": socksPort,
		}},
		"outbounds": []any{map[string]any{
			"type": "vless", "tag": "test-vless", "server": "127.0.0.1",
			"server_port": inbound.Port, "uuid": clients[0].ID,
		}},
		"route": map[string]any{"final": "test-vless"},
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
		t.Fatalf("client config rejected: %v: %s", err, out)
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
	socksAddr := fmt.Sprintf("127.0.0.1:%d", socksPort)
	deadline := time.Now().Add(5 * time.Second)
	ready := false
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", socksAddr, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			ready = true
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if !ready {
		t.Fatal("sing-box client SOCKS listener did not start")
	}

	conn, err := net.DialTimeout("tcp", socksAddr, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	if _, err := conn.Write([]byte{5, 1, 0}); err != nil {
		t.Fatal(err)
	}
	response := make([]byte, 2)
	if _, err := io.ReadFull(conn, response); err != nil || response[0] != 5 || response[1] != 0 {
		t.Fatalf("SOCKS greeting: %v %v", response, err)
	}
	request := []byte{5, 1, 0, 1, 127, 0, 0, 1, byte(targetPort >> 8), byte(targetPort)}
	if _, err := conn.Write(request); err != nil {
		t.Fatal(err)
	}
	response = make([]byte, 10)
	if _, err := io.ReadFull(conn, response); err != nil || response[1] != 0 {
		t.Fatalf("SOCKS connect: %v %v", response, err)
	}
	if _, err := io.WriteString(conn, "GET / HTTP/1.1\r\nHost: fixture\r\nConnection: close\r\n\r\n"); err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(conn)
	if err != nil || !strings.Contains(string(body), "runtime-ok") {
		t.Fatalf("VLESS request failed: %v: %s", err, body)
	}

	inboundRows, userRows, err := singboxService.GetTraffic()
	if err != nil {
		t.Fatalf("read v2ray-compatible stats: %v", err)
	}
	var measured int64
	for _, row := range userRows {
		if sameEmail(row.Email, email) {
			measured = row.Up + row.Down
		}
	}
	if measured == 0 {
		t.Fatalf("stats API returned no traffic for %q: %+v", email, userRows)
	}
	if err, _, _, _, _ := svc.AddTraffic(inboundRows, userRows); err != nil {
		t.Fatalf("commit collected traffic: %v", err)
	}
	if err := singboxService.RefreshAccount(email); err != nil {
		t.Fatalf("apply depleted quota: %v", err)
	}
	var traffic xray.ClientTraffic
	if err := database.GetDB().Where("email = ?", email).First(&traffic).Error; err != nil || traffic.Up+traffic.Down < traffic.Total || traffic.Enable {
		t.Fatalf("real VLESS traffic was not billed/depleted: %+v, %v", traffic, err)
	}
	if procMgr.IsRunning(singboxProcName(inbound.Id)) {
		t.Fatal("quota-depleted VLESS user kept the listener running")
	}

	if _, handled, err := svc.updateSingboxAccount(inbound, email, func(account *model.Account) {
		account.TotalGB = 0
		account.ExpiryTime = time.Now().Add(-time.Minute).UnixMilli()
	}); err != nil || !handled {
		t.Fatalf("apply expired account: handled=%v err=%v", handled, err)
	}
	if err := database.GetDB().Where("email = ?", email).First(&traffic).Error; err != nil || traffic.Enable {
		t.Fatalf("expired account remained enabled: %+v, %v", traffic, err)
	}
	if procMgr.IsRunning(singboxProcName(inbound.Id)) {
		t.Fatal("expired VLESS user kept the listener running")
	}
}

func TestSingboxAuditRejectsEngineSwitches(t *testing.T) {
	singboxAuditSetup(t)
	for _, core := range []string{"", "singbox"} {
		inbound := singboxAuditInbound(t)
		inbound.Core, inbound.Enable = core, false
		if err := database.GetDB().Create(inbound).Error; err != nil {
			t.Fatal(err)
		}
		changed := *inbound
		if core == "" {
			changed.Core = "singbox"
		} else {
			changed.Core = ""
		}
		if _, _, err := (&InboundService{}).UpdateInbound(&changed); err == nil {
			t.Fatal("engine switch acknowledged")
		}
		stored, err := (&InboundService{}).GetInbound(inbound.Id)
		if err != nil || stored.Core != core {
			t.Fatal("refused switch changed engine")
		}
	}
}

func TestSingboxAuditUninstallOwnershipAndAccounts(t *testing.T) {
	singboxAuditSetup(t)
	if err := (&SettingService{}).SetProvisionedProtocols([]string{"singbox"}); err != nil {
		t.Fatal(err)
	}
	raw := singboxAuditInbound(t)
	raw.Enable = false
	if err := database.GetDB().Create(raw).Error; err != nil {
		t.Fatal(err)
	}
	xrayInbound := singboxAuditInbound(t)
	xrayInbound.Core, xrayInbound.Enable, xrayInbound.Settings = "", false, `{"clients":[]}`
	if err := database.GetDB().Create(xrayInbound).Error; err != nil {
		t.Fatal(err)
	}
	cs := &CoreService{}
	if got := cs.inboundCountsByCore()["singbox"]; got != 1 {
		t.Fatalf("sing-box count=%d", got)
	}
	if err := cs.CanUninstallCores([]string{"singbox"}, InboundsBlock); err == nil || !strings.Contains(err.Error(), "still has 1 inbound") {
		t.Fatalf("missing disposition guard: %v", err)
	}
	if err := cs.CanUninstallCores([]string{"singbox"}, InboundsKeep); err != nil {
		t.Fatal(err)
	}
	if err := (&AccountService{}).ValidateMembershipSet([]*model.Inbound{raw}); err != nil {
		t.Fatalf("sing-box inbound rejected as a managed account target: %v", err)
	}
	assignable, err := (&AccountService{}).AssignableInboundsFor(&model.User{IsSuperAdmin: true})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, inbound := range assignable {
		if inbound.InboundId == raw.Id {
			found = true
		}
	}
	if !found {
		t.Fatal("sing-box inbound missing from managed account targets")
	}
	if _, err := cs.deleteCoreInbounds([]string{"singbox"}); err != nil {
		t.Fatal(err)
	}
	if _, err := (&InboundService{}).GetInbound(xrayInbound.Id); err != nil {
		t.Fatal("sing-box deletion removed Xray inbound")
	}
}
