# VPN-AIO 2.1.0 validation status

Target: fresh installation and upgrade from 2.0.0 on Ubuntu 24.04 amd64.

## Added in this branch

- AmneziaWG 3.1 controls are exposed in the modern and legacy inbound forms.
- AWG 3.1 fields are stored under `awg31` and merged safely before kernel/netlink
  configuration: S3/S4, I1-I5, header protection, padding/timing ranges, random
  trailers and cookie control.
- AWG 3.1 fields are validated, exported to client configurations and included in
  Clash-style subscription output.
- XDRIVE local, Google Drive and S3-compatible settings are exposed for VLESS
  inbound/outbound forms and validated against the pinned Xray core.
- Specific release selection is supported by `VPN_AIO_VERSION=2.0.0` or another
  published semantic version; a newer database must not be rolled back blindly.
- MASQUE inbound and outbound support covers TLS h2/h3, Basic authentication, address
  pools, account lifecycle, accounting and JSON subscriptions. Raw share links are not
  emitted because MASQUE has no standardized URI format.
- FinalMask forms now cover the pinned core's TCP/UDP mask variants and migrate legacy
  fragment, mKCP and xDNS shapes without silently discarding settings.

## Verified

- GitHub Actions: complete Go tests, GUI model round trips, Xray native protocol
  smoke tests, XDRIVE core configuration checks and SQLite build.
- Ubuntu 24.04 test host: AmneziaWG 3.1 DKMS build/load, AWG 3.1 netlink
  capability check, real handshake, bidirectional traffic and peer removal.

## Remaining before public release

- Complete full cross-distribution and protocol-matrix validation for the final release.
- Audit AWG export formats against actual client support and test editing/clearing
  advanced parameters, beyond the successful kernel handshake tests.

- Complete quota/device-limit tests with AWG 3.1 enabled.
- Authenticated GUI/API and existing-user traffic checks after upgrade.
- Destructive uninstall and final release asset build.

Internal daemon paths retain upstream compatibility names. Public installation,
service, application and primary UI branding use VPN-AIO. Original GPL-3.0 and
third-party notices remain applicable.

## Upgrade and rollback exercise (2026-11-18)

The full bundled experimental binary built successfully and was installed on
the Ubuntu test host. Version, service, control socket and HTTPS login page
were verified. The old 2.0.0 release binary was downloaded and its SHA256
checked against the published checksum.

The pre-upgrade SQLite backup was recovered with its WAL in a separate directory
and passed `PRAGMA integrity_check`. With the service stopped, the current 2.1.0
application tree and database were snapshotted. Restoring the recovered 2.0.0
database and 2.0.0 binary passed service/control-socket/HTTPS-page checks, with
the same username, port, base path and certificate settings. Finally, the exact
pre-test 2.1.0 application/database snapshot was restored and passed those checks.

Final host state: experimental 2.1.0 running. Protected recovery snapshot:
`/var/lib/vpn-aio-tests/rollback-20261118-034311`.
These checks do not establish authenticated login, all VPN traffic paths or
completion of the requested 2.1.0 feature scope. No public 2.1.0 release yet.
