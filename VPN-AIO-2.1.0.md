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

## Verified

- GitHub Actions: complete Go tests, GUI model round trips, Xray native protocol
  smoke tests, XDRIVE core configuration checks and SQLite build.
- Ubuntu 24.04 test host: AmneziaWG 3.1 DKMS build/load, AWG 3.1 netlink
  capability check, real handshake, bidirectional traffic and peer removal.

## Remaining before public release

- Full cross-distribution and complete protocol matrix.
- Complete quota/device-limit tests with AWG 3.1 enabled.
- Fresh 2.0.0 to 2.1.0 upgrade and rollback exercise using the preserved database
  backup; this is the next test step.
- Destructive uninstall and final release asset build.

Internal daemon paths retain upstream compatibility names. Public installation,
service, application and primary UI branding use VPN-AIO. Original GPL-3.0 and
third-party notices remain applicable.
