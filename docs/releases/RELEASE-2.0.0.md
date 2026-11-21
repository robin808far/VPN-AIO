# VPN-AIO 2.0.0

Fresh-install release for **Ubuntu 24.04 amd64**.

- Rebranded panel, installer, service and release updater as VPN-AIO.
- Xray **26.11.9**, preserving native AnyTLS, TUIC and NaiveProxy.
- AmneziaWG kernel module **3.1.20261113-02**, control library **v1.3.1**.
- Full embedded VPN daemon and geo bundles.
- Real-core TCP/UDP, user-counter and authentication tests; actual AWG DKMS,
  handshake, traffic and peer-removal tests on Ubuntu; full Go test suite passed.

Install:

```sh
curl -fsSL https://raw.githubusercontent.com/robin808far/VPN-AIO/main/deploy.sh | sudo bash
```

Manage with `sudo VPN-AIO`; service `VPN-AIO`; installation `/opt/VPN-AIO`.
The installer generates login credentials. Asset: `VPN-AIO-amd64`; verify its
checksum using `sha256sum -c SHA256SUMS` in the download directory.

**Scope:** This 2.0.0 release uses the compatibility AWG profile. AWG 3.1 UI
controls are being developed for 2.1.0. Only Ubuntu 24.04 amd64 was validated.
See [validation record](validation/VPN-AIO-2.0.0.md) for exact test coverage. Existing
upstream daemon paths remain; do not install alongside vpn-ui on the same host.

Based on Sir-MmD/vpn-ui and 3x-ui; upstream licenses and credits are retained.
