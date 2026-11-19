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

- GitHub Actions run `44600039700` passed for commit `82505b3`, including the
  complete Go suite, GUI model tests, pinned-core build, native-protocol smoke
  tests, XDRIVE checks and FinalMask configuration matrix.
- Ubuntu `sweden`: the full-asset 2.1.0 candidate is 354,323,144 bytes
  (338 MiB), SHA-256
  `912ecc7c14f76fbf9e3528cafe201902eb5fed76e7bb58d6347e955816843c95`.
  It was built with the complete pinned Xray/geodata and daemon bundles.
- Pinned Xray 26.11.9: native MASQUE h2/h3, AnyTLS, TUIC, Naive and all FinalMask
  TCP/UDP configuration variants parsed or passed live traffic/authentication checks.
- MASQUE live account add, removal, re-enable, traffic accounting, generated Xray
  `pass` mapping and JSON subscription output passed targeted Linux tests.
- Controlled upgrade test on `sweden`: the full 2.1.0 binary passed the authenticated
  live API suite (34/34 checks); test objects were removed, then the exact pre-test
  binary, systemd unit and SQLite database were restored. The old binary SHA-256 and
  database bytes/integrity were verified; the restored service, Xray 26.11.9 and AWG
  3.1 report healthy. Recovery snapshot: `/var/lib/vpn-aio-tests/upgrade-20261119-200458`.
- Ubuntu 24.04 test host: AmneziaWG 3.1 DKMS build/load, AWG 3.1 netlink
  capability check, real handshake, bidirectional traffic and peer removal. Tests
  now assert every 3.1 field in `.conf`, structured export and Clash output.
- Linux client compatibility: upstream `amneziawg-tools` v3.1.20260812 imported
  a profile rendered from the live panel and round-tripped its available settings
  through the AWG 3.1 kernel. A separate full-option profile (including all new
  3.1 fields) completed a real handshake and bidirectional tunnel ping. The client
  accepts boolean values as `on`/`off`, not `true`/`false`; the `.conf` exporter and
  regression test now use the accepted syntax. Desktop/mobile AWG clients were not
  tested.
- Controlled full-candidate upgrade on `sweden`: the candidate above started the
  panel and Xray 26.11.9; a certificate-verified HTTPS request traversed an existing
  enabled VLESS/XHTTP profile and increased both Xray traffic counters. The live
  panel also rendered an existing AWG profile that imported and round-tripped with
  upstream AWG tools/kernel.
- Rollback restored and verified the exact pre-test binary, systemd unit, SQLite
  database and WAL/SHM snapshot. The service is active again with the original
  binary (SHA-256 `e0865aa6d1884b5b95a484e6974d1ef35171d05fa5328a0fbcb4b256439e3695`),
  SQLite integrity is `ok`, four enabled inbounds remain, and no test inbounds remain.
  Recovery snapshot: `/var/lib/vpn-aio-tests/upgrade-fullprofile-awgfix-20261120`.

## Remaining before public release

- Complete cross-distribution protocol-matrix validation and destructive uninstall.
- Publish the final release asset/checksum after the remaining host/client checks.

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
