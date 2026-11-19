package service

import (
	"encoding/json"
	"testing"

	"github.com/mhsanaei/3x-ui/v2/database"
	"github.com/mhsanaei/3x-ui/v2/database/model"
)

func TestGetXrayConfigProjectsMasquePasswordToPass(t *testing.T) {
	newInboundDB(t)
	inbound := &model.Inbound{
		UserId: 1, Tag: "masque-wire", Protocol: model.MASQUE, Enable: true, Port: 8443,
		Settings:       `{"clients":[{"email":"user@masque","password":"secret","enable":true}],"address":["10.14.0.1/24"]}`,
		StreamSettings: `{"network":"masque","security":"tls","tlsSettings":{"alpn":["h3"],"certificates":[{"certificateFile":"/tmp/cert","keyFile":"/tmp/key"}]}}`,
	}
	if err := database.GetDB().Create(inbound).Error; err != nil {
		t.Fatal(err)
	}
	config, err := (&XrayService{}).GetXrayConfig()
	if err != nil {
		t.Fatalf("GetXrayConfig: %v", err)
	}
	for _, got := range config.InboundConfigs {
		if got.Tag != inbound.Tag {
			continue
		}
		var settings struct {
			Clients []map[string]any `json:"clients"`
		}
		if err := json.Unmarshal(got.Settings, &settings); err != nil {
			t.Fatal(err)
		}
		if len(settings.Clients) != 1 || settings.Clients[0]["email"] != "user@masque" || settings.Clients[0]["pass"] != "secret" {
			t.Fatalf("emitted MASQUE clients = %#v", settings.Clients)
		}
		if _, exists := settings.Clients[0]["password"]; exists {
			t.Fatalf("panel password key reached Xray: %#v", settings.Clients[0])
		}
		return
	}
	t.Fatalf("generated config omitted %s", inbound.Tag)
}
