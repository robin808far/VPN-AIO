package service

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v2/database"
	"github.com/mhsanaei/3x-ui/v2/database/model"
)

func newNodeTestDB(t *testing.T) {
	t.Helper()
	if err := database.InitDB(filepath.Join(t.TempDir(), "test.db")); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
}

// A created token verifies by plaintext, never leaks the plaintext through the
// model, and carries exactly the scopes it was minted with.
func TestApiTokenCreateAndVerify(t *testing.T) {
	newNodeTestDB(t)
	s := &ApiTokenService{}
	plaintext, token, err := s.Create("master-a", ScopeNodeSync+","+ScopeNodeAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(plaintext, "vpnaio_") {
		t.Fatalf("unexpected plaintext format: %q", plaintext)
	}
	if token.TokenHash == plaintext || token.TokenHash == "" {
		t.Fatal("hash must be stored, not the plaintext")
	}
	found, err := s.Verify(plaintext)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if found.Id != token.Id || found.Name != "master-a" {
		t.Fatalf("verify returned the wrong token: %+v", found)
	}
	if !s.HasScope(found, ScopeNodeSync) || !s.HasScope(found, ScopeNodeAdmin) {
		t.Error("token must carry both minted scopes")
	}
	if _, err := s.Verify("bogus"); err == nil {
		t.Error("verify must reject an unknown token")
	}
	if err := s.Revoke(token.Id); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, err := s.Verify(plaintext); err == nil {
		t.Error("verify must reject a revoked token")
	}
}

func TestNodeURLAndNormalize(t *testing.T) {
	s := &NodeService{}
	n := &model.Node{Address: "vpn.example.com", Port: 8443, BasePath: "/panel/", Scheme: ""}
	s.normalize(n)
	if got := s.url(n, "status"); got != "http://vpn.example.com:8443/panel/nodeapi/status" {
		t.Fatalf("unexpected url: %s", got)
	}
	empty := &model.Node{Address: "10.0.0.1"}
	s.normalize(empty)
	if empty.Scheme != "http" || empty.BasePath != "/" || empty.SyncMode != "all" {
		t.Fatalf("defaults not applied: %+v", empty)
	}
}

// Add probes the node right away, so an unreachable node is stored as "down"
// instead of sitting on the optimistic default.
func TestNodeAddProbesUnreachable(t *testing.T) {
	newNodeTestDB(t)
	s := &NodeService{}
	node, err := s.Add(&model.Node{Name: "child-1", Address: "127.0.0.1", Port: 1})
	if err != nil {
		t.Fatal(err)
	}
	if node.Status != "down" {
		t.Fatalf("expected status down for an unreachable node, got %q", node.Status)
	}
	if err := s.Del(node.Id); err != nil {
		t.Fatalf("del: %v", err)
	}
	if _, err := s.Get(node.Id); err != ErrNodeNotFound {
		t.Fatalf("expected ErrNodeNotFound after delete, got %v", err)
	}
}
