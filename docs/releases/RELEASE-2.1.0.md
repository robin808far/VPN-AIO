# VPN-AIO 2.1.0

Release for **Ubuntu 24.04 amd64**, with the existing Vue interface and
the pinned Xray 26.11.9 core.

## Changes

- XDRIVE transport settings for local storage, Google Drive and S3-compatible storage.
- MASQUE inbound/outbound configuration, TLS h2/h3, account lifecycle, traffic
  accounting and JSON subscriptions. MASQUE has no standardized raw share URI;
  use the supported JSON configuration output.
- FinalMask TCP/UDP settings, QUIC tuning and migration of legacy saved mask shapes.
- AmneziaWG 3.1 controls in modern/legacy forms, validation and kernel configuration,
  including S3/S4, I1-I5, header protection, padding/timing ranges, random trailers
  and cookie control. Linux `.conf` exports use the upstream client's `on`/`off`
  boolean syntax.
- TLS/REALITY/socket and routing-rule editors retain advanced settings when
  another field is edited. XHTTP session fields use the pinned core's
  `sessionIDPlacement`/`sessionIDKey` names; older panel names migrate on save.
- Full uninstall also removes panel-owned WireGuard, AmneziaWG and GRE kernel
  interfaces, while `--cores keep` retains them.
- Installer accepts `VPN_AIO_VERSION=2.0.0` (or another published version) for
  version-specific installation. Published 2.0.0 tags/assets remain unchanged.
- Resolve CodeQL findings in Clash list allocation and the routing-modal test.

## Validation and limits

- Live Ubuntu upgrade/rollback, authenticated API checks, an existing-account
  HTTPS tunnel, AWG kernel handshake/traffic/peer removal and full systemd uninstall.
- Pinned-core configuration and live native-protocol tests, model round trips,
  Go tests and isolated startup/uninstall checks in CI.
- Userland startup/native-core/config/uninstall checks passed on 13 distribution
  container images with an earlier full candidate. These containers share the
  Ubuntu host kernel and are not proof of VPN/DKMS support on other distro kernels.
- The cross-distro VM/kernel/tunnel matrix was **not run**, by operator decision.
  Ubuntu 24.04 amd64 is the real-host validation target. AWG desktop/mobile clients
  were not tested; Linux upstream tools and the matching 3.1 kernel were tested.
- XDRIVE cloud settings were parser-validated; this record does not claim live
  Google Drive or S3 transfers. See `validation/VPN-AIO-2.1.0.md` for the exact validation history.

## Installation and rollback

Use the installer from the matching release tag after publication:

```sh
curl -fsSL https://raw.githubusercontent.com/robin808far/VPN-AIO/v2.1.0/deploy.sh -o deploy.sh
sudo env VPN_AIO_VERSION=2.1.0 bash deploy.sh
```

Asset: `VPN-AIO-amd64`. Download `SHA256SUMS` alongside it and run
`sha256sum -c SHA256SUMS` before installation.

Full asset size: **354,332,320 bytes**. SHA-256:

```text
8777fe57cc50a2735af195a4ba5f98e88e1efb94b1c5c498b607b105606c6fdc  VPN-AIO-amd64
```

The binary is built from clean source commit
`bd467e04f867f88426c60ba03354a05c8fa2cc7e`, with the full cached pinned core,
six geodata files and daemon bundle. `BUILD-MANIFEST.json` and
`embedded-inputs.SHA256SUMS` record the source and embedded-input provenance.

Back up the database and configuration before upgrading. A version selector
chooses a binary; it does **not** downgrade a newer SQLite schema. To roll back,
restore the binary together with its matching pre-upgrade database/configuration.
Manage with `sudo VPN-AIO`; service `VPN-AIO`; install path `/opt/VPN-AIO`.

Based on Sir-MmD/vpn-ui and 3x-ui. GPL-3.0 and original third-party notices remain.
