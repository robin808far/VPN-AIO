# VPN-AIO 3.0.0 — release provenance and validation

## Release gates

The operator authorized publication through GitHub PR #3. Publication requires
successful current-head CI, CodeQL language analyses and an explicit check for
open PR code-scanning alerts. A successful analysis job alone is not enough.
The public v3.0.0 release and its manifest identify the published artifact.

Published v2.0.0/v2.1.0 release/tag/asset metadata was snapshotted before this work.
Publication must preserve their tag SHAs, asset IDs, digests, sizes and timestamps;
only the computed Latest selection changes.

## Feature validation

Implementation commit: `dd89f19` (`feature/hosts-v3.0.0`). The full implementation
and pre-release validation are recorded in `hosts-plan.md` and `HOSTS.md`.
The development binary is not the release asset.

The full Go suite, pinned-core Host JSON acceptance, three Node scripts, browser
create/edit/toggle/picker/domain/IPv6 checks and 30/30 Ubuntu-container lifecycle
checks passed during implementation. The browser layout check used a 390px
viewport; raster capture was unavailable because the desktop window was hidden.

## Release asset

- Source commit: `21bc50db071409390197be1435043f4e640b0b26`, clean detached worktree.
- Xray pin: `e0a162b8eb10634e297afecedfc569308c490014`.
- Telemt pin: `1e6549df23136a676ef8ec36ef0f1d9af2a1083d`.
- Toolchain: Go `1.27.4`, Linux/amd64, SQLite with `CGO_ENABLED=1`.
- Command: `bash build.sh --quick --skip-bundle --no-color`.
- Asset: `VPN-AIO-amd64`, **354,573,744 bytes**.
- SHA-256: `4261971d3bcde8d95254fa4b4b2d59328619cd2e2fe8af0775809e60355c0603`.

The complete cached Xray, six geodata files and native-daemon assets were checked
against every entry in the validated v2.1.0 embedded-input inventory. Both
submodule checkouts matched their recorded pins, `go mod verify` succeeded, and
the build left tracked source clean. `--quick` reused those verified inputs;
it did not select a reduced bundle or omit geodata. No upstream tip was adopted.

Build directory: `/home/ubuntu/vpn-aio-release-21bc50d` on the authorized test host.
The release includes `SHA256SUMS`, `BUILD-MANIFEST.json` and
`embedded-inputs.SHA256SUMS` alongside the binary.

## Exact-asset validation

- Complete Linux `go test -count=1 -timeout=10m ./...`: passed without test-name
  exclusions, including actual pinned-core acceptance of generated Host JSON.
- The private network/mount test namespace avoided the running host's port 12301;
  the existing URL-validation test used the implementation phase's public-DNS
  fixture for `example.com`, and geodata came from the complete bundled cache.
- Ubuntu container lifecycle: **30/30**, process exit **0**, using the exact
  release asset. Authenticated fresh startup, native protocol smoke, XDRIVE,
  FinalMask and transport configuration checks and isolated uninstall passed.
- Node scripts ran successfully during implementation and in both GitHub Ubuntu
  CI runs for this source. Node is not installed on the release-build host; no
  server-side Node execution is claimed.
- Live existing-install upgrade/rollback: **8/8**. The exact candidate booted with
  Xray/AWG, migrated the existing DB to the Host schema, and passed a
  certificate-verified HTTPS request through an existing profile with both Xray
  traffic counters increasing. The original application/configuration trees,
  binary, SQLite and unit were restored; hashes, DB integrity and the original
  enabled-inbound count were verified, and Xray/AWG recovered.

Protected recovery snapshot:
`/var/lib/vpn-aio-tests/release-3.0.0-21bc50d/`. Build/input/module/full-Go/container
logs are under the build directory's `build/out/`; the root-only snapshot holds
the upgrade results, core states, traffic result and restoration hashes.

## Build policy

Build from clean committed source. Keep the Xray submodule at its recorded pin;
do not move to upstream tips as part of publication. Complete cached Xray,
geodata and native-daemon assets may be reused only after verifying their input
hashes. Record the source commit, toolchain, build command, file size and SHA-256.

Any change affecting compiled source or embedded files requires a new asset and
its checks. Subsequent documentation-only commits may refer to the exact tested
source commit, as in the v2.1.0 release record.

## Preservation and coverage

The test server was returned to its original experimental 2.1.0 binary
(`e0865aa6d1884b5b95a484e6974d1ef35171d05fa5328a0fbcb4b256439e3695`),
with four original enabled inbounds and a healthy SQLite database. No real users
are present on the operator-designated disposable `sweden` server.

The cross-distribution VM/kernel/DKMS matrix remains explicitly waived. Container
userland checks do not establish cross-kernel, VPN-daemon or systemd-boot coverage.
Do not count skipped work as a passed test, or claim an unperformed CDN test.
