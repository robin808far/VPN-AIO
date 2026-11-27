package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v2/config"
	"github.com/mhsanaei/3x-ui/v2/database"
	"github.com/mhsanaei/3x-ui/v2/database/model"
)

func TestMergeBaseConfigRejectsManagedInbounds(t *testing.T) {
	for _, raw := range []string{
		`{"inbounds":[]}`,
		`{"inbounds":null}`,
		`{"inbounds":[{"type":"direct"}]}`,
	} {
		if _, err := mergeBaseConfig(raw); err == nil || !strings.Contains(err.Error(), `must not contain "inbounds"`) {
			t.Errorf("mergeBaseConfig(%s) error = %v, want explicit inbounds refusal", raw, err)
		}
	}
	if _, err := mergeBaseConfig(DefaultSingboxBaseConfig); err != nil {
		t.Fatalf("valid base config rejected: %v", err)
	}
	merged, err := (&SingboxService{}).mergeIntoBase(json.RawMessage(`{"type":"vless","tag":"managed"}`), `{"inbounds":[{"type":"direct"}],"outbounds":[{"type":"direct"}]}`)
	if err != nil {
		t.Fatalf("legacy stored base prevented current listener rendering: %v", err)
	}
	var rendered map[string]json.RawMessage
	if err := json.Unmarshal(merged, &rendered); err != nil {
		t.Fatal(err)
	}
	var inbounds []map[string]any
	if err := json.Unmarshal(rendered["inbounds"], &inbounds); err != nil || len(inbounds) != 1 || inbounds[0]["tag"] != "managed" {
		t.Fatalf("stored base replaced or appended listeners: %s err=%v", rendered["inbounds"], err)
	}
}

func TestApiTokenRejectsUnknownScopesAndNormalizesKnownScopes(t *testing.T) {
	newNodeTestDB(t)
	svc := &ApiTokenService{}
	for _, scopes := range []string{"node:sync", "node-sync,unknown", "node-sync,,node-admin"} {
		if _, _, err := svc.Create("invalid-"+scopes, scopes); !errors.Is(err, ErrInvalidTokenScope) {
			t.Errorf("Create scopes %q error = %v, want ErrInvalidTokenScope", scopes, err)
		}
	}
	var count int64
	if err := database.GetDB().Model(&model.ApiToken{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("invalid scope requests persisted %d tokens", count)
	}

	_, token, err := svc.Create("normalized", " node-sync, node-admin,node-sync ")
	if err != nil {
		t.Fatal(err)
	}
	if token.Scopes != ScopeNodeSync+","+ScopeNodeAdmin {
		t.Fatalf("stored scopes = %q, want normalized supported scopes", token.Scopes)
	}
	_, token, err = svc.Create("default-scope", "   ")
	if err != nil || token.Scopes != ScopeNodeSync {
		t.Fatalf("blank scope default = %+v, err %v", token, err)
	}
}

func TestNodeAddRequiresApiToken(t *testing.T) {
	newNodeTestDB(t)
	if _, err := (&NodeService{}).Add(&model.Node{Name: "missing-token", Address: "127.0.0.1", Port: 1}); !errors.Is(err, ErrNodeTokenRequired) {
		t.Fatalf("Add without token error = %v, want ErrNodeTokenRequired", err)
	}
	var count int64
	if err := database.GetDB().Model(&model.Node{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("missing-token node was persisted: %d rows", count)
	}
}

func TestRestartImportedCoreServicesAttemptsBothCores(t *testing.T) {
	var xrayCalled, singboxCalled bool
	xrayErr := errors.New("xray unavailable")
	got := restartImportedCoreServices(func() error {
		xrayCalled = true
		return xrayErr
	}, func() error {
		singboxCalled = true
		return errors.New("sing-box unavailable")
	})
	if !xrayCalled || !singboxCalled {
		t.Fatalf("restart calls: xray=%t sing-box=%t", xrayCalled, singboxCalled)
	}
	if !errors.Is(got, xrayErr) || !strings.Contains(got.Error(), "sing-box unavailable") {
		t.Fatalf("joined restart error = %v", got)
	}
}

func TestImportDBBackfillsAndStartsSingboxUsers(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VPNAIO_DB_FOLDER", root)
	previousRoot := singboxConfigRoot
	singboxConfigRoot = filepath.Join(root, "singbox")
	t.Cleanup(func() {
		procMgr.StopByPrefix("singbox-server-")
		singboxConfigRoot = previousRoot
		_ = database.CloseDB()
	})

	binDir := t.TempDir()
	bin := filepath.Join(binDir, "sing-box")
	script := "#!/bin/sh\ncase \"$1\" in\ncheck) if grep -q review_invalid \"$3\"; then exit 1; fi;;\nrun) exec sleep 600;;\nesac\n"
	if err := os.WriteFile(bin, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	legacyPath := filepath.Join(root, "legacy.db")
	if err := database.InitDB(legacyPath); err != nil {
		t.Fatal(err)
	}
	legacy := &model.Inbound{
		Core: "singbox", Protocol: model.VLESS, Enable: true,
		Listen: "127.0.0.1", Port: port, Tag: fmt.Sprintf("inbound-%d", port),
		TrafficReset: "never",
		Settings:     fmt.Sprintf(`{"type":"vless","listen":"127.0.0.1","listen_port":%d,"users":[{"name":"restore@example.test","uuid":"11111111-1111-4111-8111-111111111111"}]}`, port),
	}
	if err := database.GetDB().Create(legacy).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Checkpoint(); err != nil {
		t.Fatal(err)
	}
	if err := database.CloseDB(); err != nil {
		t.Fatal(err)
	}

	if err := database.InitDB(config.GetDBPath()); err != nil {
		t.Fatal(err)
	}
	backup, err := os.Open(legacyPath)
	if err != nil {
		t.Fatal(err)
	}
	importErr := (&ServerService{}).ImportDB(backup)
	backup.Close()
	if importErr != nil && !strings.Contains(importErr.Error(), "restart Xray") {
		t.Fatalf("database import failed before managed-core activation: %v", importErr)
	}

	account, err := (&AccountService{}).GetAccountByEmail("restore@example.test")
	if err != nil || account == nil {
		t.Fatalf("legacy sing-box user was not backfilled to an account: %+v err=%v", account, err)
	}
	var restored model.Inbound
	if err := database.GetDB().Where("port = ?", port).First(&restored).Error; err != nil {
		t.Fatal(err)
	}
	if !procMgr.IsRunning(singboxProcName(restored.Id)) {
		t.Fatal("sing-box listener was not restarted from the restored database")
	}
}
