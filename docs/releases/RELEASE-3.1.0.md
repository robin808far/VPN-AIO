# VPN-AIO 3.1.0

## Happ subscription controls

VPN-AIO 3.1.0 adds native Happ client management to Panel Settings → Subscription.
Operators can configure routing, informational and expiration notices, Provider ID,
replacement and fallback subscription URLs, TUN behavior, route exclusions, ping,
auto-connect, per-app proxy behavior, and local proxy authentication.
See the [Happ guide](../guides/HAPP.md) for client compatibility and Provider ID/privacy notes.

Happ-specific application headers are opt-in and are returned only when the request
User-Agent identifies Happ. Existing routing headers remain unchanged for all
subscription clients. Settings are persisted in the panel key/value store and are
loaded when the subscription server starts; restart the panel after changing them.

## Compatibility and validation

This release continues to target Ubuntu 24.04 amd64. This note does not claim a
fresh validation of every distribution in the inherited upstream support matrix.
See [the 3.1.0 validation record](validation/VPN-AIO-3.1.0.md) for checks performed.

The committed source passed the complete Linux Go suite and Ubuntu CI; the exact
asset passed 53 HTTP contract checks and 30 isolated lifecycle checks. Windows
Happ import/refresh of subscription metadata was checked separately with a safe
test profile. Global client overrides and Android/iOS were not exercised.
