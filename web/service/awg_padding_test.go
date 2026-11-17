package service

import (
	"encoding/json"
	"testing"

	"github.com/Jipok/wgctrl-go/wgtypes"
)

func TestAwgPaddingReachesKernelConfig(t *testing.T) {
	var settings awgSettings
	if err := json.Unmarshal([]byte(`{"s3":12,"s4":24}`), &settings); err != nil {
		t.Fatal(err)
	}
	b := settings.resolveObfs()
	var cfg wgtypes.Config
	b.apply(&cfg)
	if cfg.S3 == nil || *cfg.S3 != 12 || cfg.S4 == nil || *cfg.S4 != 24 {
		t.Fatalf("padding lost before netlink: %+v", cfg)
	}
	changed := b
	changed.S4++
	if b.sig() == changed.sig() {
		t.Fatal("editing transport padding must invalidate applied configuration")
	}
	if settings.mtu() != wgDefaultMTU-24 {
		t.Fatal("automatic MTU must account for transport padding")
	}
	settings.Mtu = 1300
	if settings.mtu() != 1300 {
		t.Fatal("explicit MTU must be preserved")
	}
}

func TestAwgOutboundPaddingAndLegacyDefaults(t *testing.T) {
	var settings awgOutSettings
	if err := json.Unmarshal([]byte(`{"s3":16,"s4":32}`), &settings); err != nil {
		t.Fatal(err)
	}
	b := settings.obfs()
	if b.S3 != 16 || b.S4 != 32 || settings.mtu() != wgDefaultMTU-32 {
		t.Fatalf("outbound lost padding: %+v", b)
	}
	legacy := (&awgSettings{}).resolveObfs()
	if legacy.S3 != 0 || legacy.S4 != 0 {
		t.Fatal("legacy configurations must not acquire new padding")
	}
}
