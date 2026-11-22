package sub

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func happHeaderTestContext(userAgent string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/sub/id", nil)
	ctx.Request.Header.Set("User-Agent", userAgent)
	return ctx, recorder
}

func TestApplyHappHeadersWritesConfiguredValues(t *testing.T) {
	ctx, recorder := happHeaderTestContext("Happ/2.0 (Android)")
	applyHappHeaders(ctx, HappConfig{
		AutoDetect: true, ProviderID: "provider", NewURL: "https://new.example/sub",
		FallbackURL: "https://backup.example/sub", SubInfoColor: "green", SubInfoText: "Hello",
		SubInfoButtonText: "Help", SubInfoButtonLink: "https://example.com/help",
		SubExpire: true, SubExpireButtonLink: "https://example.com/renew", NotificationExpire: true,
		NoLimit: true, HideSettings: true, AlwaysHWID: true, TunMode: "gvisor", TunType: "singbox",
		ExcludeRoutes: "10.0.0.0/8", ExcludeAPNs: true, ColorProfile: "resetcolors", PingType: "proxy-head",
		AutoConnect: true, AutoConnectType: "lowestdelay", PerAppMode: "bypass", PerAppList: "org.example.app",
		LocalProxyAuth: "auto",
	})

	want := map[string]string{
		"ProviderID": "provider", "New-Url": "https://new.example/sub", "Fallback-Url": "https://backup.example/sub",
		"Sub-Info-Color": "green", "Sub-Info-Text": "Hello", "Sub-Info-Button-Text": "Help",
		"Sub-Info-Button-Link": "https://example.com/help", "Sub-Expire": "1", "Sub-Expire-Button-Link": "https://example.com/renew",
		"Notification-Subs-Expire": "1", "No-Limit-Enabled": "1", "Hide-Settings": "1",
		"Subscription-Always-Hwid-Enable": "1", "Tun-Mode": "gvisor", "Tun-Type": "singbox",
		"Exclude-Routes": "10.0.0.0/8", "Exclude-Apns-Enable": "true", "Color-Profile": "resetcolors",
		"Ping-Type": "proxy-head", "Subscription-Autoconnect": "1", "Subscription-Autoconnect-Type": "lowestdelay",
		"Per-App-Proxy-Mode": "bypass", "Per-App-Proxy-List": "org.example.app", "Socks-Auth-Mode": "auto", "Http-Auth-Mode": "auto",
	}
	for key, value := range want {
		if got := recorder.Header().Get(key); got != value {
			t.Errorf("%s = %q, want %q", key, got, value)
		}
	}
}

func TestApplyHappHeadersRequiresOptInAndHappUserAgent(t *testing.T) {
	for _, tc := range []struct {
		name      string
		userAgent string
		config    HappConfig
	}{
		{name: "other client", userAgent: "v2rayNG/1.9", config: HappConfig{AutoDetect: true, ProviderID: "provider"}},
		{name: "disabled opt-in", userAgent: "Happ/2.0", config: HappConfig{ProviderID: "provider"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, recorder := happHeaderTestContext(tc.userAgent)
			applyHappHeaders(ctx, tc.config)
			if got := recorder.Header().Get("ProviderID"); got != "" {
				t.Fatalf("ProviderID = %q, expected no Happ headers", got)
			}
			if got := recorder.Header().Get("Routing-Enable"); got != "" {
				t.Fatalf("applyHappHeaders unexpectedly set legacy routing header: %q", got)
			}
		})
	}
}

func TestApplyHappHeadersClearsManagedBooleanValues(t *testing.T) {
	ctx, recorder := happHeaderTestContext("Happ/2.0")
	applyHappHeaders(ctx, HappConfig{AutoDetect: true})
	for key, want := range map[string]string{
		"Sub-Info-Text": "0", "Sub-Expire": "0", "Notification-Subs-Expire": "0",
		"No-Limit-Enabled": "0", "Hide-Settings": "0", "Subscription-Always-Hwid-Enable": "0",
		"Subscription-Autoconnect": "0", "Exclude-Apns-Enable": "false", "Per-App-Proxy-Mode": "off",
	} {
		if got := recorder.Header().Get(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestApplyHappHeadersSanitizesNewlines(t *testing.T) {
	ctx, recorder := happHeaderTestContext("Happ/2.0")
	applyHappHeaders(ctx, HappConfig{AutoDetect: true, NewURL: "https://example.com/sub\r\nX-Injected: yes"})
	if got := recorder.Header().Get("X-Injected"); got != "" {
		t.Fatalf("injected header value = %q", got)
	}
	if got := recorder.Header().Get("New-Url"); got != "https://example.com/subX-Injected: yes" {
		t.Fatalf("New-Url = %q, want sanitized value", got)
	}
}

func TestApplyHappHeadersNormalizesEnumsAcceptedBySettings(t *testing.T) {
	ctx, recorder := happHeaderTestContext("happ/2.0")
	applyHappHeaders(ctx, HappConfig{
		AutoDetect: true, TunMode: " GVISOR ", TunType: " SINGBOX ", PingType: " PROXY-HEAD ",
	})
	for key, want := range map[string]string{"Tun-Mode": "gvisor", "Tun-Type": "singbox", "Ping-Type": "proxy-head"} {
		if got := recorder.Header().Get(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestApplyHappHeadersDoesNotSendAppListWhenPerAppProxyIsOff(t *testing.T) {
	ctx, recorder := happHeaderTestContext("Happ/2.0")
	applyHappHeaders(ctx, HappConfig{AutoDetect: true, PerAppMode: "off", PerAppList: "org.example.app"})
	if got := recorder.Header().Get("Per-App-Proxy-List"); got != "" {
		t.Fatalf("inactive per-app list sent to Happ: %q", got)
	}
}
