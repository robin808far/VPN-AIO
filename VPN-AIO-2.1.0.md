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
- XHTTP, TLS/REALITY and socket forms preserve unedited core fields during saves;
  DNS/routing template JSON remains raw and round-trippable. The routing-rule modal
  now retains fields it does not expose when editing a known field.

## Verified

- GitHub Actions run `44680089511` passed for commit `1785614`, including the
  complete Go suite, GUI model tests, pinned-core build, native-protocol smoke
  tests, XDRIVE checks, FinalMask configuration matrix and isolated Ubuntu
  startup/authentication/native-core/uninstall lifecycle check.
- Clean source build from commit `178561444feab699ba5c9a40c7fed545b82a04ae`:
  the full-asset 2.1.0 candidate is 354,332,600 bytes
  (338 MiB), SHA-256
  `5c5c654a043d22bda91345c91e48cf5f37f144ca59f783ec4a0e090d6375f64b`.
  Built in `/home/ubuntu/vpn-aio-release-1785614/` with Go 1.27.4, initialized pinned
  submodules and the complete cached Xray/geodata/daemon bundles. No tracked source
  changes were present in the build checkout. `BUILD-MANIFEST.json` and
  `embedded-inputs.SHA256SUMS` record the source and embedded-input provenance.
- On that exact clean checkout, `go test -timeout 10m ./... -count=1` passed on
  `sweden` without a test-name exclusion filter. The panel was temporarily stopped
  to free the fixed SSH integration-test port and restored afterward. An initial
  run in an offline network namespace was inconclusive (a DNS-dependent test and
  an SSH accounting timing assertion failed); the complete host-network run passed.
- The exact release asset passed the Ubuntu container lifecycle check: 30/30
  checks, including authenticated startup, embedded native protocols, XDRIVE,
  FinalMask, transport settings, uninstall and preservation of operator files.
  Results: `/home/ubuntu/vpn-aio-release-1785614/build/out/lifecycle-final/`.
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
- Frontend transport/settings audit: round-trip checks cover inbound/outbound
  XHTTP, TLS/REALITY, sockopt, sniffing and unknown future fields; legacy REALITY
  `dest` is normalized to `target`, and routing-rule edits preserve selectors such
  as `localIP`, `process`, `webhook` and unknown keys. The pinned Xray 26.11.9 parser
  accepted inbound and outbound XHTTP+TLS and XHTTP+REALITY configurations containing
  sockopt; the inbound profiles also include DNS, routing and sniffing. A JSON
  regression test confirms DNS/routing template sections remain raw across config
  validation/representation.
- Pinned-core option names: canonical `v6only`/`maxTimeDiff` values survive edits,
  and XHTTP sessions serialize as `sessionIDPlacement`/`sessionIDKey`, including
  copied share links; legacy panel spellings are read and normalized on save.
- Controlled full-candidate upgrade on `sweden`: an earlier full candidate started the
  panel and Xray 26.11.9; a certificate-verified HTTPS request traversed an existing
  enabled VLESS/XHTTP profile and increased both Xray traffic counters. The live
  panel also rendered an existing AWG profile that imported and round-tripped with
  upstream AWG tools/kernel.
- Rollback restored and verified the exact pre-test binary, systemd unit, SQLite
  database and WAL/SHM snapshot. The service is active again with the original
  binary (SHA-256 `e0865aa6d1884b5b95a484e6974d1ef35171d05fa5328a0fbcb4b256439e3695`),
  SQLite integrity is `ok`, four enabled inbounds remain, and no test inbounds remain.
  Recovery snapshot: `/var/lib/vpn-aio-tests/upgrade-fullprofile-awgfix-20261120`.
- Disposable-container lifecycle smoke with the earlier full candidate: authenticated
  startup and Xray native/XDRIVE/FinalMask checks passed; uninstall removed the
  binary, SQLite DB, extracted bundles, firewall table, policy rule and route, while
  preserving unrelated sentinel files. Container has no systemd boot, so this does
  not close the VM/systemd uninstall gate.
- Distribution **userland** matrix: 13/13 passed using full-asset candidate
  `912ecc7c…16843c95`, built before the uninstall-only kernel-interface cleanup and
  before the final transport round-trip audit (neither change alters tunnel handling):
  Ubuntu 24.04/26.04, Debian 12/13, Fedora 43/44, AlmaLinux 9/10, Rocky Linux 9/10,
  CentOS Stream 9/10 and Arch. Every image passed authenticated startup, the native
  Xray protocol smoke suite, XDRIVE/FinalMask configuration validation and isolated
  uninstall. Runtime tests ran offline in separate Docker network/filesystem/PID
  namespaces; all images shared the Ubuntu host kernel. The tests verified binary,
  DB, bin/cert/backups/socket/menu/unit-file removal, private nft/policy-route
  cleanup and preservation of unrelated sentinel files. This does not establish
  VPN daemon/tunnel, distro-kernel/DKMS or systemd service-stop compatibility.
  Logs, image IDs and machine-readable checks are archived on the host under
  `/home/ubuntu/vpn-aio-validation-containers-20261120/`.
- Added a regression test that verifies full uninstall attempts cleanup of every
  daemon-less kernel core interface and records failures without skipping later
  cleanups.
- Destructive `sweden` cycle using the pre-audit uninstall-fix candidate
  (`ab6c062e8f25c6acae877124f0a5d7dbc80fb96e8be2c8ae735cbcd2b20918eb`)
  passed 39/39 host
  assertions, including systemd stop/disable, removal of `awg1`, release of TCP/UDP
  listeners, DKMS cleanup and an existing-account HTTPS request. This first exposed
  and then verified the fix for an AWG interface surviving uninstall. The exact
  original binary, service unit and SQLite bytes were restored; SQLite is healthy,
  four enabled inbounds remain and Xray/AWG are running. Recovery snapshot:
  `/var/lib/vpn-aio-tests/host-uninstall-final-20261120/`.
- The later transport-audit candidate (`5fa99d1eaf7491d261a0bf753c3ae15ee2692a8abb6c4b565ecdc3a09ff2488d`)
  also passed the 39-check host upgrade/HTTPS/uninstall/restore cycle. Recovery
  snapshot: `/var/lib/vpn-aio-tests/host-uninstall-release-20261120/`.
- The clean-source candidate listed above passed 7/7 upgrade/rollback checks:
  exact asset installed; Xray and AWG running; certificate-verified HTTPS through
  an existing VLESS/XHTTP profile with both counters increased; original binary,
  database and service-unit bytes restored; service recovered; SQLite integrity
  `ok`; four original enabled inbounds retained. Recovery snapshot:
  `/var/lib/vpn-aio-tests/release-1785614/`.

## Accepted validation limit

- Per operator decision, the cross-distribution VPN tunnel/kernel/DKMS VM matrix is
  skipped; release claims remain limited accordingly. The Docker userland matrix
  shares one host kernel and live tunnel checks were limited to Ubuntu 24.04. Do not
  interpret the 13-container result as proof that VPN tunnels or DKMS work across all
  listed distro kernels.

## Release preparation

- Public release is not yet published. `RELEASE-2.1.0.md` describes the changes,
  installation/rollback requirements, checksum and exact scope of validation.
- The release asset is built from commit `1785614`; subsequent release notes and
  validation-record changes do not alter the built source. Keep all published
  2.0.0 tags and assets unchanged.

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
