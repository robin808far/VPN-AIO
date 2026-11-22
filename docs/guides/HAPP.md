# Happ subscription management

Version 3.1.0 adds a **Happ application settings** section under **Panel Settings → Subscription**, using the existing Vue panel.

## Enable and apply

1. Enable the subscription service and configure its public URL as usual.
2. Open the Happ section and enable **Enable Happ settings**.
3. Configure Routing, Appearance, Network, and Provider/failover as needed.
4. Save settings, then restart the panel using its existing restart action.
5. Refresh the subscription in Happ. Routing changes may also require reconnecting.

New application-management headers are only sent when the request User-Agent identifies Happ and the Happ settings switch is enabled. Existing `Routing` and `Routing-Enable` headers remain compatible with earlier VPN-AIO releases, independently of that switch. Browser HTML views and individual VPN config downloads are not app-management responses.

## Controls

- **Routing:** existing routing toggle/deeplink, presets for Iran, China, global proxy, LAN bypass, or off; ad blocking is optional. Presets modify the unsaved draft until Save.
- **Appearance:** information banner, button, banner color, expiration notice/reminder, and iOS theme JSON or `resetcolors`.
- **Network:** desktop TUN mode/core, excluded routes/APNs, ping, auto-connect strategy, Android per-app proxy list, and local SOCKS/HTTP authentication mode.
- **Provider/failover:** Provider ID, replacement/fallback subscription URLs, and non-disableable HWID reporting.

TUN mode accepts `system` or `gvisor`; leaving a selection at **Do not override** omits the corresponding header. TUN core selection is a desktop feature; color profiles are an iOS feature. Availability depends on the Happ version and platform.

Enabled Happ management sends explicit off values for managed boolean controls, including hiding settings and auto-connect. An empty banner sends `Sub-Info-Text: 0`. Disabling the master Happ switch stops sending app overrides; it does not necessarily clear values Happ has already cached.

Replacement and fallback fields are literal HTTP(S) URLs, shared by all subscribers. No account-specific substitution is performed. Do not put one customer's subscription ID into a panel-wide URL intended for everyone.

## Provider ID and privacy

Happ requires Provider ID for its advanced app-management features. Obtain it from your own Happ provider account; VPN-AIO does not register one or call the provider service.

According to Happ's documentation, the app itself can report Provider ID, subscription-domain hash, device HWID, and operating-system information to Happ's service. Enabling **Require HWID** is a client instruction, not server-side device-limit enforcement in VPN-AIO.

**No Limit** increases Happ's Xray memory allowance; it does not remove VPN-AIO account quotas. **Hide server settings** restricts Happ's UI, not the server's authorization or subscription access.

## Verification limits

Header/preset tests, full Linux subscription/web/service tests, and the actual panel HTTP contract check have passed. Windows Happ 4.8.3 also imported/refreshed a separate test subscription and displayed its quota, expiry and changing information banner. Global overrides were filtered out of the device test, so TUN/auto-connect and other platform-specific behaviors are not claimed as tested. See the [validation record](../releases/validation/VPN-AIO-3.1.0.md).

Sources: [app management](https://www.happ.su/main/dev-docs/app-management), [Provider ID](https://www.happ.su/main/dev-docs/provider-id), [routing](https://www.happ.su/main/dev-docs/routing).
