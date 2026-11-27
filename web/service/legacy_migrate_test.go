package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mhsanaei/3x-ui/v2/config"
)

func TestMigrateLegacyIn(t *testing.T) {
	root := t.TempDir()
	old := filepath.Join(root, "etc", config.LegacyToken+"-singbox", "server-5")
	if err := os.MkdirAll(old, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(old, "config.json"), []byte("{}"), 0o600)
	// both exist: legacy must be kept, new untouched
	os.MkdirAll(filepath.Join(root, "etc", config.LegacyToken+"-sstp"), 0o755)
	os.MkdirAll(filepath.Join(root, "etc", "vpn-aio-sstp"), 0o755)
	os.MkdirAll(filepath.Join(root, "opt"), 0o755)
	os.WriteFile(filepath.Join(root, "opt", config.LegacyToken+".log"), []byte("x"), 0o644)

	res := migrateLegacyIn(root, []string{"/etc"}, []string{"/opt/" + config.LegacyToken + ".log"})
	if len(res.moved) != 2 || len(res.warnings) != 1 {
		t.Fatalf("moved=%v warnings=%v", res.moved, res.warnings)
	}
	if _, err := os.Stat(filepath.Join(root, "etc", "vpn-aio-singbox", "server-5", "config.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "opt", "vpn-aio.log")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "etc", config.LegacyToken+"-sstp")); err != nil {
		t.Fatal("legacy dir must be kept when target exists")
	}
	// idempotent
	res = migrateLegacyIn(root, []string{"/etc"}, nil)
	if len(res.moved) != 0 {
		t.Fatalf("second run moved %v", res.moved)
	}
}

func TestCopyTree(t *testing.T) {
	src, dst := t.TempDir(), filepath.Join(t.TempDir(), "d")
	os.MkdirAll(filepath.Join(src, "a"), 0o755)
	os.WriteFile(filepath.Join(src, "a", "f"), []byte("hi"), 0o640)
	if err := copyTree(src, dst); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dst, "a", "f"))
	if err != nil || string(b) != "hi" {
		t.Fatal(err, string(b))
	}
}
