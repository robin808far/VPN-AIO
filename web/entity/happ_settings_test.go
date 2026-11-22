package entity

import (
	"strings"
	"testing"
)

func TestHappSettingValidation(t *testing.T) {
	valid := AllSetting{WebPort: 2083, SubPort: 2097}
	if err := valid.CheckValid(); err != nil {
		t.Fatalf("valid default Happ settings rejected: %v", err)
	}

	for _, tc := range []struct {
		name    string
		mutate  func(*AllSetting)
		wantErr string
	}{
		{name: "banner limit counts unicode characters", mutate: func(s *AllSetting) { s.SubHappSubInfoText = strings.Repeat("界", 201) }, wantErr: "Happ subscription info text must be at most 200 characters"},
		{name: "button label limit", mutate: func(s *AllSetting) { s.SubHappSubInfoButtonText = strings.Repeat("x", 26) }, wantErr: "Happ subscription info button text must be at most 25 characters"},
		{name: "unsupported TUN mode", mutate: func(s *AllSetting) { s.SubHappTunMode = "default" }, wantErr: "Happ TUN mode must be system or gvisor: default"},
		{name: "unsupported per-app mode", mutate: func(s *AllSetting) { s.SubHappPerAppMode = "allow-all" }, wantErr: "Happ per-app proxy mode must be off, on, or bypass: allow-all"},
		{name: "replacement must be an HTTP URL", mutate: func(s *AllSetting) { s.SubHappNewUrl = "file:///etc/passwd" }, wantErr: "Happ replacement URL must be an absolute HTTP(S) URL without credentials"},
		{name: "fallback must not contain credentials", mutate: func(s *AllSetting) { s.SubHappFallbackUrl = "https://user:password@backup.example/sub" }, wantErr: "Happ fallback URL must be an absolute HTTP(S) URL without credentials"},
		{name: "theme must be a JSON object", mutate: func(s *AllSetting) { s.SubHappColorProfile = "[1,2]" }, wantErr: "Happ color profile must be a JSON object or resetcolors"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			settings := valid
			tc.mutate(&settings)
			err := settings.CheckValid()
			if err == nil || err.Error() != tc.wantErr {
				t.Fatalf("CheckValid() error = %v, want %q", err, tc.wantErr)
			}
		})
	}
}
