package service

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/mhsanaei/3x-ui/v2/database/model"
)

func TestAwg31NestedSettingsMergeBeforeKernelParse(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	settings := `{"jc":4,"jmin":8,"jmax":80,"awg31":{"s3":12,"s4":24,"i1":"<r 32>",` +
		`"headerProtectionKey":"` + key + `","contentPaddingAddition":"8-24","rekeyAfterTime":"100-120",` +
		`"rekeyTimeout":"3-6","rejectAfterTime":"160-200","keepaliveTimeout":"8-16",` +
		`"maxHandshakeAttempts":"15-25","randomTrailers":true,"disableCookies":true}}`
	in := &model.Inbound{Settings: settings}
	got, err := (&AwgService{}).parseSettings(in)
	if err != nil {
		t.Fatal(err)
	}
	obfs := got.resolveObfs()
	if obfs.S3 != 12 || obfs.S4 != 24 || obfs.I1 != "<r 32>" || !*obfs.RandomTrailers || !*obfs.DisableCookies {
		t.Fatalf("nested 3.1 fields not merged: %+v", obfs)
	}
	wire := map[string]any{"s3": obfs.S3, "s4": obfs.S4, "i1": obfs.I1, "headerProtectionKey": obfs.HeaderProtectionKey,
		"contentPaddingAddition": obfs.ContentPaddingAddition, "randomTrailers": *obfs.RandomTrailers, "disableCookies": *obfs.DisableCookies}
	data, _ := json.Marshal(wire)
	if len(data) == 0 {
		t.Fatal("failed to serialize effective AWG options")
	}
}

func TestAwg31SettingsRejectInvalidKeyAndRanges(t *testing.T) {
	for _, settings := range []string{
		`{"awg31":{"headerProtectionKey":"bad"}}`,
		`{"awg31":{"rekeyAfterTime":"100-nope"}}`,
		`{"awg31":{"s4":1281}}`,
	} {
		if _, err := (&AwgService{}).parseSettings(&model.Inbound{Settings: settings}); err == nil {
			t.Errorf("accepted invalid settings %s", settings)
		}
	}
}

func TestAwg31NestedClientSettingsMerge(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	d := &awgOutDriver{}
	st, err := d.settings(VpnOutboundConfig{Settings: json.RawMessage(`{"awg31":{"s3":12,"s4":20,"headerProtectionKey":"` + key + `","rekeyTimeout":"3-6","randomTrailers":true,"disableCookies":true}}`)})
	if err != nil {
		t.Fatal(err)
	}
	b := st.obfs()
	if b.S3 != 12 || b.S4 != 20 || b.HeaderProtectionKey != key || b.RekeyTimeout != "3-6" || b.RandomTrailers == nil || !*b.RandomTrailers {
		t.Fatalf("nested client AWG settings lost: %+v", b)
	}
}
