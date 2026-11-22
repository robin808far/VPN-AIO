package service

import (
	"path/filepath"
	"testing"

	"github.com/mhsanaei/3x-ui/v2/database"
)

func TestHappSettingsSurviveSaveAndDatabaseReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "happ.db")
	if err := database.InitDB(path); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.CloseDB() })
	service := &SettingService{}
	settings, err := service.GetAllSetting()
	if err != nil {
		t.Fatal(err)
	}
	if settings.SubHappAutoDetect {
		t.Fatal("upgrading a database without Happ settings must not enable app overrides")
	}
	settings.SubHappAutoDetect = true
	settings.SubHappSubInfoText = "پیام مشترک"
	settings.SubHappTunMode = "gvisor"
	settings.SubHappFallbackUrl = "https://backup.example/sub"
	settings.SubHappAutoConnect = true
	settings.SubHappAutoConnectType = "lastused"
	if err := service.UpdateAllSetting(settings); err != nil {
		t.Fatal(err)
	}
	if err := database.CloseDB(); err != nil {
		t.Fatal(err)
	}
	if err := database.InitDB(path); err != nil {
		t.Fatal(err)
	}
	reloaded, err := service.GetAllSetting()
	if err != nil {
		t.Fatal(err)
	}
	if !reloaded.SubHappAutoDetect || !reloaded.SubHappAutoConnect ||
		reloaded.SubHappSubInfoText != "پیام مشترک" || reloaded.SubHappTunMode != "gvisor" ||
		reloaded.SubHappFallbackUrl != "https://backup.example/sub" || reloaded.SubHappAutoConnectType != "lastused" {
		t.Fatal("saved Happ configuration changed after reopening the database")
	}
}
