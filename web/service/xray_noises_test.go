package service

import (
	"encoding/json"
	"testing"

	"github.com/mhsanaei/3x-ui/v2/util/json_util"
	"github.com/mhsanaei/3x-ui/v2/xray"
)

func TestStripEmptyFreedomNoises(t *testing.T) {
	in := `[{"tag":"direct","protocol":"freedom","settings":{"domainStrategy":"AsIs","noises":[]}},` +
		`{"tag":"keep","protocol":"freedom","settings":{"noises":[{"type":"rand","packet":"10-20"}]}},` +
		`{"tag":"other","protocol":"blackhole","settings":{"noises":[]}}]`
	cfg := &xray.Config{OutboundConfigs: json_util.RawMessage(in)}
	if err := stripEmptyFreedomNoises(cfg); err != nil {
		t.Fatal(err)
	}
	var obs []map[string]any
	if err := json.Unmarshal(cfg.OutboundConfigs, &obs); err != nil {
		t.Fatal(err)
	}
	if _, ok := obs[0]["settings"].(map[string]any)["noises"]; ok {
		t.Fatal("empty freedom noises not removed")
	}
	if obs[0]["settings"].(map[string]any)["domainStrategy"] != "AsIs" {
		t.Fatal("other freedom settings lost")
	}
	if n, _ := obs[1]["settings"].(map[string]any)["noises"].([]any); len(n) != 1 {
		t.Fatal("non-empty noises must be kept")
	}
	if _, ok := obs[2]["settings"].(map[string]any)["noises"]; !ok {
		t.Fatal("non-freedom outbound must not be touched")
	}
}
