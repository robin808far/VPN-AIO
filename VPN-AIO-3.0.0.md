# VPN-AIO 3.0.0 — release provenance and validation

## Release workflow

- [x] Implement managed Hosts and run local regression/browser validation.
- [x] Snapshot published v2.0.0/v2.1.0 release and tag metadata before release work.
- [ ] Open the release PR and pass GitHub CI, CodeQL analysis and alert checks.
- [ ] Build a complete release binary from a clean, identified source commit.
- [ ] Verify cached embedded inputs against the previously validated inventory.
- [ ] Run the complete Linux Go suite and exact-asset isolated lifecycle checks.
- [ ] Exercise upgrade/rollback with recovery on the disposable test server.
- [ ] Publish the reviewed asset, manifest and checksums as v3.0.0 / Latest.
- [ ] Verify public download, version, checksum, installer and old-release integrity.

Publication is authorized by the operator. It is gated on the checks above; this
record does not claim a completed release until the public verification succeeds.

## Feature validation

Implementation commit: `dd89f19` (`feature/hosts-v3.0.0`). The full implementation
and pre-release validation are recorded in `hosts-plan.md` and `HOSTS.md`.
The development binary is not the release asset.

The full Go suite, pinned-core Host JSON acceptance, three Node scripts, browser
create/edit/toggle/picker/domain/IPv6 checks and 30/30 Ubuntu-container lifecycle
checks passed during implementation. The browser layout check used a 390px
viewport; raster capture was unavailable because the desktop window was hidden.

## Build policy

Build from clean committed source. Keep the Xray submodule at its recorded pin;
do not move to upstream tips as part of publication. Complete cached Xray,
geodata and native-daemon assets may be reused only after verifying their input
hashes. Record the source commit, toolchain, build command, file size and SHA-256.

Any change affecting compiled source or embedded files requires a new asset and
its checks. Subsequent documentation-only commits may refer to the exact tested
source commit, as in the v2.1.0 release record.

## Preservation and coverage

Preserve published v2.0.0/v2.1.0 tags and all asset identities/digests. Back up the
test server's binary, database/WAL and service state before any live upgrade;
restore and verify them after the exercise. No real users are present on the
operator-designated disposable `sweden` server.

The cross-distribution VM/kernel/DKMS matrix remains explicitly waived. Container
userland checks do not establish cross-kernel, VPN-daemon or systemd-boot coverage.
Do not count skipped work as a passed test, or claim an unperformed CDN test.
