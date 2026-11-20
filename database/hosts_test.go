package database

import (
	"path/filepath"
	"testing"

	"github.com/mhsanaei/3x-ui/v2/database/model"
	"github.com/mhsanaei/3x-ui/v2/util/crypto"
)

func TestHostsLegacyMigrationRunsOnceAndKeepsOriginalSettings(t *testing.T) {
	file := filepath.Join(t.TempDir(), "hosts.db")
	if err := InitDB(file); err != nil {
		t.Fatal(err)
	}
	// Recreate the pre-Hosts state while keeping the normal application schema.
	db.Exec("DELETE FROM history_of_seeders WHERE seeder_name = ?", "HostsFromExternalProxyV3")
	stream := `{"network":"ws","externalProxy":[{"dest":"cdn.example","port":8443,"forceTls":"tls","remark":"edge","sni":"sni.example"}],"future":{"keep":true}}`
	ib := &model.Inbound{UserId: 1, Protocol: model.VLESS, Port: 41941, Tag: "hosts-migrate", StreamSettings: stream, Settings: `{"clients":[]}`}
	if err := db.Create(ib).Error; err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := CloseDB(); err != nil {
			t.Fatal(err)
		}
		if err := InitDB(file); err != nil {
			t.Fatal(err)
		}
		var rows []struct {
			Address, Sni, Security string
			Port                   int
		}
		if err := db.Table("hosts").Where("inbound_id = ?", ib.Id).Find(&rows).Error; err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 || rows[0].Address != "cdn.example" || rows[0].Sni != "sni.example" || rows[0].Security != "tls" || rows[0].Port != 8443 {
			t.Fatalf("migration %d: %#v", i, rows)
		}
		var got model.Inbound
		if err := db.First(&got, ib.Id).Error; err != nil {
			t.Fatal(err)
		}
		if got.StreamSettings != stream {
			t.Fatalf("migration changed legacy stream: %s", got.StreamSettings)
		}
	}
	t.Cleanup(func() { _ = CloseDB() })
}

func TestHostsMigrationDoesNotRehashFreshPanelPasswordsOnRestart(t *testing.T) {
	file := filepath.Join(t.TempDir(), "hosts-password.db")
	if err := InitDB(file); err != nil {
		t.Fatal(err)
	}
	var before model.User
	if err := db.First(&before).Error; err != nil {
		t.Fatal(err)
	}
	if !crypto.CheckPasswordHash(before.Password, defaultPassword) {
		t.Fatal("fresh password is invalid")
	}
	if err := CloseDB(); err != nil {
		t.Fatal(err)
	}
	if err := InitDB(file); err != nil {
		t.Fatal(err)
	}
	var after model.User
	if err := db.First(&after).Error; err != nil {
		t.Fatal(err)
	}
	if before.Password != after.Password || !crypto.CheckPasswordHash(after.Password, defaultPassword) {
		t.Fatal("Hosts seeder ordering rehashed an existing bcrypt password")
	}
	t.Cleanup(func() { _ = CloseDB() })
}
