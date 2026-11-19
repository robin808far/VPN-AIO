package sub

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/mhsanaei/3x-ui/v2/database/model"
)

func TestMasqueSubscriptionCarriesClientCredentialsAndStripsServerKeys(t *testing.T) {
	inbound := &model.Inbound{
		Protocol: model.MASQUE, Listen: "203.0.113.9", Port: 8443,
		StreamSettings: `{"network":"masque","security":"tls","masqueSettings":{"path":"/tunnel","warp":{"privateKey":"server-only"}},"tlsSettings":{"alpn":["h2"],"serverName":"vpn.example","certificates":[{"key":["PRIVATE KEY"]}]}}`,
	}
	svc := NewSubJsonService("", "", "", "", &SubService{remarkModel: "-ieo"})
	configs := svc.getConfig(inbound, model.Client{Email: "alice", Password: "alice-pass"}, "vpn.example")
	if len(configs) != 1 {
		t.Fatalf("configs = %d, want 1", len(configs))
	}
	var config struct {
		Outbounds []struct {
			Protocol string         `json:"protocol"`
			Settings map[string]any `json:"settings"`
			Stream   map[string]any `json:"streamSettings"`
		} `json:"outbounds"`
	}
	if err := json.Unmarshal(configs[0], &config); err != nil {
		t.Fatal(err)
	}
	got := config.Outbounds[0]
	if got.Protocol != "masque" || got.Settings["address"] != "vpn.example" || got.Settings["port"] != float64(8443) {
		t.Fatalf("wrong MASQUE destination: %+v", got)
	}
	want := map[string]any{"path": "/tunnel", "user": "alice", "pass": "alice-pass"}
	if !reflect.DeepEqual(got.Stream["masqueSettings"], want) {
		t.Fatalf("credentials = %#v, want %#v", got.Stream["masqueSettings"], want)
	}
	tls := got.Stream["tlsSettings"].(map[string]any)
	if tls["certificates"] != nil || tls["serverName"] != "vpn.example" {
		t.Fatalf("server TLS keys leaked or SNI lost: %#v", tls)
	}
	if raw := (&SubService{}).getLink(inbound, "alice"); raw != "" {
		t.Fatalf("unsupported raw MASQUE link was emitted: %q", raw)
	}
}
