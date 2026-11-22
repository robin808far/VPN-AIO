package sub

import "testing"

func TestApplyCommonHeadersPreservesRoutingAndAddsHappMetadata(t *testing.T) {
	ctx, recorder := happHeaderTestContext("Happ/2.0")
	controller := &SUBController{happConfig: HappConfig{AutoDetect: true, ProviderID: "provider"}}
	controller.ApplyCommonHeaders(ctx, "upload=0", "12", "VPN", "", "", "", true, "happ://routing/off")

	if got := recorder.Header().Get("ProviderID"); got != "provider" {
		t.Fatalf("ProviderID = %q, want provider", got)
	}
	if got := recorder.Header().Get("Routing-Enable"); got != "true" {
		t.Fatalf("Routing-Enable = %q, want true", got)
	}
	if got := recorder.Header().Get("Routing"); got != "happ://routing/off" {
		t.Fatalf("Routing = %q, want legacy routing value", got)
	}
}

func TestApplyCommonHeadersDoesNotChangeNonHappRoutingBehavior(t *testing.T) {
	ctx, recorder := happHeaderTestContext("v2rayNG/1.9")
	controller := &SUBController{happConfig: HappConfig{AutoDetect: true, ProviderID: "provider"}}
	controller.ApplyCommonHeaders(ctx, "", "", "", "", "", "", false, "")

	if got := recorder.Header().Get("ProviderID"); got != "" {
		t.Fatalf("ProviderID = %q for non-Happ client", got)
	}
	if got := recorder.Header().Get("Routing-Enable"); got != "false" {
		t.Fatalf("Routing-Enable = %q, want legacy false value", got)
	}
}
