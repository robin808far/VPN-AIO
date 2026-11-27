# Project Progress & State

Last updated: 2026-11-27 (v6.6.0 tag pushed, release CI in progress at save time; v6.5.1 published and verified; installer no-tty defect fixed; sing-box core locked in UI; sweden disk cleaned to 63%; sing-box 1.14.5 upgrade not started — patch does not apply)

> Current truth supersedes earlier candidate/release-preparation notes below.
> `main` = `origin/main` = tag `v6.6.0` peeled = `1fb52746` (verified with
> `git ls-remote`). Earlier tags: `v6.5.1` peeled `4a752bc0`, `v6.5.0` peeled
> `a52abcd71639f74c6d3b3f57102ea435125ae4eb` (tag object `81be9cb9`).
> Only `PROGRESS.md`/`HANDOFF.md` are uncommitted at save time.
> Live host (sweden) runs v6.5.0 (installed by the user) and was NOT touched by
> this session except read-only checks and disk cleanup of temp/build files.

## Project relocation (2026-11-27)

- Main checkout moved to `C:\Users\mef\Documents\OpenCode Projects\VPN-AIO`
  (full copy incl. `.git`); old `Default Project\VPN-AIO` deleted after copy.
- Old uncommitted node-feature WIP saved as branch `backup/local-node-wip`
  (pushed to origin). Local `main` fast-forwarded to `f129f173` (= origin/main).
- Worktree `Default Project\VPN-AIO-review-6.1.0` repaired and still attached.
- Continue work in a new OpenCode session rooted at the new folder.

## Next step (open requests)

- Production: user updated the live host to v6.5.0 themselves (2026-11-27);
  assistant only verifies, does not redeploy.
- Optional remaining qualification: full ARM64 runtime, VM/systemd/kernel, and
  physical mobile tests (not performed).
- (Done 2026-11-27) 1) docs consistency commit `ba870c16`.
- (Done) 2) Installer `/dev/tty` defect fixed in `deploy.sh` (commit `3b640772`,
  pushed): `deploy_have_tty` opens /dev/tty instead of `-r`. Verified on sweden:
  `bash -n` OK; under `setsid` the new check reports no tty while old `-r` was
  true. Not yet in a release (public install.sh serves raw `main` deploy.sh?
  unverified); full unattended install rerun not performed.
- (Done) 3) Live host (sweden) verified: `/opt/VPN-AIO/VPN-AIO-amd64` 409479064
  bytes, SHA-256 `9a26f831...` (= v6.5.0 release), `-v` 6.5.0, service
  active/running since 2026-11-27 16:11:50 UTC, port 12301 listening.
- (Done) Full unattended install verified (2026-11-27, WSL2 Ubuntu 24.04,
  systemd, mirrored networking): public `main/install.sh` run under `setsid`
  with stdin `/dev/null`, no PANEL_TLS -> completed without hang: HTTP default,
  random credentials, `-v` 6.5.0, unit enabled/active, panel HTTP 200 on
  localhost, NRestarts=0. Exit code not captured cleanly (PowerShell escaping
  printed `EXIT=True`), but log reached "Deploy complete". Cleanup: distro
  unregistered, `.wslconfig` removed.
- `install.sh` always fetches `deploy.sh` from `main` (`BRANCH="main"`); the
  version argument only pins the binary. So deploy.sh fixes reach all installs
  immediately (also when pinning older tags) — main must stay installable.
- (Done) Small improvements (2026-11-27, pushed):
  a) `90ab5caa`: `deploy.sh` unattended custom login via env
     `PANEL_USER/PANEL_PASS/PANEL_PORT/PANEL_PATH` (any set -> custom mode, no
     prompt; checked after import-keep, before tty prompt) + new
     `test_unit/deploy-tty-check.py` (5/5 PASS on sweden; `bash -n` OK).
     Full install with PANEL_* env not run end-to-end.
  b) `01fb60fd`: `stripEmptyFreedomNoises` in `web/service/xray.go` removes
     empty `noises` from freedom outbounds at config build (non-empty kept);
     removed `"noises": []` from `web/service/config.json` and
     `sub/default.json`. `TestStripEmptyFreedomNoises` passed, `go vet
     ./web/service` clean, gofmt clean (after CRLF strip) on sweden.
     Takes effect on the live host only after a new release/deploy.
- Not released: deploy.sh changes are live via `main`; the Xray fix needs a
  new binary release (not done, needs user decision).
- Release 6.5.1 (user-authorized 2026-11-27): `config/version` 6.5.1, commit
  `4a752bc0` pushed to main; annotated tag `v6.5.1` (object `e545c8bb`)
  pushed. Release CI 46036573081 + Ubuntu validation 46036567653 both
  succeeded for `4a752bc0`. Public assets downloaded on sweden: amd64
  409479888 B `052cecd0...`, arm64 403822952 B `97126a13...`, SHA256SUMS
  `d04a9309...`; `sha256sum -c` OK for both; amd64 `-v` = 6.5.1; both
  contain `vpn-aio-limits` marker. Temp dir removed. arm64 `-v` not executed.
  Live host not updated by assistant (user deploys).
- (Done, pushed `00f79955`, not released) UI items from user 2026-11-27:
  4) sing-box "Update from source" button removed from Core settings; lock tag
     + `singboxCoreLocked` i18n (13 locales, English text); `/singbox/update`
     route now returns a locked error. (Upstream archives lack vpn-aio-limits,
     so the old update always failed its own guard; patch was never at risk.)
  3) `GetCoresStatus` order: singbox right after xray.
  2) Dashboard: new `bo-t-sbox` tile (state, version, uptime; Log/Stop/Restart
     via `/panel/core/{logs,stop,restart}/singbox`) under Xray; manage tile
     moved one row down (half of the left column; grid areas updated for 760px
     and 1180px). Uptime from new `ProcManager.OldestStartWithPrefix` +
     `started` field, exposed as `extra.uptime` (seconds).
  Checks on sweden: `go build ./...` OK, `go vet ./web/...` clean,
  `go test ./web/controller` + `-run Core|Status|Singbox ./web/service` OK.
  NOT done: browser visual check of the dashboard/Core page.
- Pending: release 6.6.0 for these UI items (user approved order).
- 6.6.0 (user-authorized 2026-11-27): Persian labels added to
  `[pages.core]` in fa_IR (version/stop/restart/logs/state*) + Persian
  `singboxCoreLocked`; dashboard `coreStateText` now uses i18n. TOML parsed
  OK. Commit/tag `v6.6.0` -> `1fb52746`, pushed. Release CI 46055354122 +
  Ubuntu validation 46055349392 started; asset verification pending.
  Earlier temp UI check (dashboard, 1000px): sbox tile under Xray, Manage
  under it — OK; Core settings page and >=1180px not visually checked by
  assistant (user will check on live host after deploying 6.6.0).
  Pre-existing console 404 `/ui/ui/panel/nodes/list` (doubled base path)
  noted, not investigated.
- sweden disk cleanup (user-requested 2026-11-27): 87% -> 63% (22G free).
  Removed: go build cache (4.7G), old release/verification/build dirs
  (vpn-aio-build-70af, vpn-aio-release-{21bc50d,bd467e0,1785614,272ab016},
  public-{2.1.0,3.0.0}-verification, amd64-pre-ui, hosts-v3-lifecycle-context,
  ci-lifecycle-context), temp sing-box build dirs and go.tar.gz in /tmp,
  apt cache, journal >7d, exited container vpn-aio-hosts-v3-ui-20261120, UI
  check container/workdir/tunnel. Kept: running review/audit containers,
  their images, vpn-aio-managed-accounts-20261125 (build cache for core),
  patched sing-box `/tmp/vpn-aio-managed-accounts-singbox-limits-20261126`,
  live /opt/VPN-AIO (service active).
- Then item 1: sing-box 1.14.5. `singbox-limits.patch` fails on 1.14.5
  (`experimental/v2rayapi/stats.go:98`); needs rewrite + full runtime tests.
- Next: 4) new feature — awaiting user specification.

## systemd/kernel install test on WSL2 (2026-11-27, fresh observation)

- User chose WSL2 + Ubuntu 24.04 on the Windows desktop (no KVM on sweden).
  Installed distro `Ubuntu-24.04`, enabled `systemd=true` in `/etc/wsl.conf`,
  and created `%USERPROFILE%\.wslconfig` (`networkingMode=mirrored`,
  `autoProxy=true`). Kernel `6.18.58.1-microsoft-standard-WSL2` (Microsoft
  kernel, not a stock Ubuntu kernel; not a full VM qualification).
- Windows proxy `127.0.0.1:10808` is unreachable from WSL; installs ran with
  proxy variables unset (direct GitHub returned 200).
- Public `install.sh v6.5.0` with stdin `/dev/null`: download OK, then hung
  on `read < /dev/tty` (credential prompt). With `setsid` (no ctty): failed at
  `deploy.sh` line 458 `/dev/tty: No such device or address`.
  **Installer defect (unfixed):** `[[ -r /dev/tty ]]` is true without a usable
  controlling terminal, so non-interactive installs (cloud-init/CI) hang or
  fail, contrary to the script's own comments. Not changed in source.
- Succeeded via pseudo-tty (`script`) + `PANEL_TLS=http`, default random
  credentials: exit 0, `VPN-AIO is running`. Verified: `-v` = 6.5.0, binary
  SHA-256 `9a26f831...` (= release), unit enabled/active, `Restart=on-failure`;
  `systemctl restart` OK; `kill -9` -> auto-restarted (NRestarts=1, new PID);
  survived `wsl --terminate` restart (active after boot); no failed units;
  journal only Xray deprecation warning about Freedom `noises`.
- Panel HTTP 200 via `localhost`/`[::1]`; `127.0.0.1` times out because
  mirrored mode routes 127.0.0.1 to the Windows host (table 127) — WSL
  artifact, not a product defect.
- Not tested: real VM/stock kernel, AWG/kernel modules, protocol traffic,
  ARM64 runtime, mobile. User accepted the test results as sufficient.
- Cleanup done (user-approved): `wsl --unregister Ubuntu-24.04` (no distros
  remain), `.wslconfig` deleted, temp download dir removed.
- Status docs committed and pushed to `main` (docs only; no source change since
  v6.5.0, so no 6.6.0 release per user instruction). Next: user's next stage.

## Release verification (2026-11-27, complete)

- [Release CI 45991545057](https://github.com/robin808far/VPN-AIO/actions/runs/45991545057)
  and [Ubuntu validation 45991539428](https://github.com/robin808far/VPN-AIO/actions/runs/45991539428)
  both completed successfully for `a52abcd7`.
- [v6.5.0](https://github.com/robin808far/VPN-AIO/releases/tag/v6.5.0) is public,
  not draft/prerelease, published `2026-11-27T11:35:52Z`; unauthenticated latest
  API returns this tag. Assets: two architecture binaries plus `SHA256SUMS`.
- Downloaded all three public assets to a new temporary directory on `sweden`.
  Both complete binary hashes match `SHA256SUMS` and GitHub asset digests:

| Asset | Bytes | SHA-256 | Executed `-v` |
| --- | ---: | --- | --- |
| `VPN-AIO-amd64` | 409479064 | `9a26f83110b3b7654c309b75ae8277ce41121bde723e350280ffb8945db88cc7` | `6.5.0` natively on x86_64 Linux |
| `VPN-AIO-arm64` | 403822560 | `9224c5456a9d0a17c495bcf5053e2ebcdb9d02a45e22efa17884acc74d975e78` | `6.5.0` under QEMU user emulation |

- `SHA256SUMS`: 160 bytes, SHA-256
  `db30cbdbe23a49b7790bcde8b3a61e5024430afcb635be57e23b49f8ec09b2b3`.
- CI reused cached daemon bundles, so verified the actual architecture-specific
  embedded sing-box bytes using ELF symbols and Go embed metadata (without
  starting a panel). Both cores contain `vpn-aio-limits`, execute `version` as
  `1.12.14`, and report `with_v2ray_api` plus revision
  `f56d9ab94569d46a48bfc2bcd97eea566ad365c8`.
  amd64 core: 46985351 bytes, SHA-256
  `6ea33f44f4d1bec1c640538e637378032786be0b9282618f851779a405adfc35`
  (identical to the previously runtime-tested core).
  arm64 core: 45219975 bytes, SHA-256
  `00c6e6bb55333de24cafb5b5eaff718a1fc35651f097d8857f28cd18b6ee7396`.
- QEMU and its ARM64 userspace libraries were downloaded/extracted inside the
  disposable verification directory only; no host packages/binfmt were installed.
  Removed that entire created directory and the empty directory from an initial
  failed command after verification; no test service/container was started.
- Resolved installer line-ending uncertainty: Git index has LF for `install.sh`
  and `deploy.sh`; both public raw `main` URLs return HTTP 200 with LF and no CR
  (81 and 587 LF respectively). Local Windows working copies are CRLF. No source
  line-ending changes or installer execution were needed.
- This continuation changed only the two status documents. The existing
  candidate's matrix/UI/runtime results below apply to that candidate; they
  were not rerun against the larger public CI binaries. Full ARM64 runtime,
  VM/systemd/kernel and physical mobile qualification remain unperformed.

## Latest session (2026-11-27, fresh observations)

1. Commit `359ee7f4` "fix: expose ShadowTLS in inbound protocol picker"
   (inbound.js, singbox.test.js, PROGRESS.md, HANDOFF.md); local Node
   regression passed before commit. Committer identity is the auto-configured
   `mef <mef@safagroup.safait>` (same as 122e8866).
2. Pushed `HEAD:main`; `origin/main` = `359ee7f4` (verified by fetch).
3. Remaining checks on candidate `6bdbeb42...` (sha verified on sweden):
   - Installer contract `test_unit/install-sh-check.py`: 8/8 PASS on an LF copy
     in `/tmp/inst-check`. Running it in the sweden workspace failed only
     because that copy of `install.sh` has CRLF endings (`set: pipefail`);
     repo/curl-served file line endings were not changed.
   - Live UI harness `test_unit/live/uitest.js`: 60/60 PASS against disposable
     container `vpn-aio-ui-harness-20261127` (image
     `vpn-aio-lifecycle-test:latest`, published 127.0.0.1:22346), local headless
     Chrome over SSH tunnel.
   - Detour runtime check (ad-hoc disposable script, API-driven, not committed):
     18/18 PASS — select detour merges two processes into one, clear splits back
     to two, re-select merges, deleting target clears the front's detour and the
     front keeps running, self-detour rejected, cleanup leaves no stray process.
     This is API/process level, not a click-through of the selector.
   - Container, tunnel, headless Chrome, and temp files removed. Live listener
     12301 still present and untouched.
4. Release: `config/version` changed 6.1.0 -> 6.5.0, committed as `a52abcd7`
   and pushed to main; annotated tag `v6.5.0` pushed (user-authorized). Release workflow `.github/workflows/vpn-aio-release.yml` runs on
   `v*` tag push and builds amd64+arm64 in CI (installs gcc-aarch64 and qemu
   binfmt there), so the local ARM64 toolchain gap does not block CI.
   Done: bump commit, tag push, successful release/validation CI, and public
   asset verification recorded above. VM/systemd/kernel and physical mobile
   checks were NOT run.
Path: `C:\Users\mef\Documents\Default Project\VPN-AIO-review-6.1.0`

## Goal

Completed sing-box managed-account remediation and user-authorized v6.5.0
publication in
`VPN-AIO-review-6.1.0`: raw sing-box `users` map to `Account`,
`AccountInbound`, and `ClientTraffic`; account edits project back into native
`users`; quota/expiry enforcement works. Speed and IP limits are now enforced
for sing-box through the shared `speedlimits.json` sidecar (consumed by a
patched bundled sing-box via `XRAY_SPEEDLIMIT_FILE`); physical-device counting
is not supported and is rejected. A separate `StreamLimit` field caps concurrent
routed streams/UDP associations. Then confirm and fix defects in the external
report `C:\Users\mef\Desktop\Bug.md` against this checkout, changing only
issues reproduced in current code. Commits `122e8866` (remediation), `359ee7f4`
(picker fix), and `a52abcd7` (6.5.0 version) are pushed and released as v6.5.0.
No production/live-database deployment was made. Earlier audit-host provisioning,
performed separately, is documented in `HANDOFF.md`.

Implemented requirement (2026-11-27): panel-managed ShadowTLS multi-inbound
chaining replaces the earlier one-process-per-inbound design with a detour graph (`SingboxDetourInboundID`
plus raw `detour`), resolved into connected process groups, with one root owner
per group for process/config/stats API port. Cycles, self-detours, missing
targets, and non-sing-box targets must be rejected. Deleted targets clear
detour references and old standalone processes are stopped before group startup.
Chained process user stats must use `InboundId = 0` to avoid misattributing
account traffic, while per-member inbound rows remain emitted.

## Current State

- Panel-managed ShadowTLS multi-inbound chaining is committed and published in
  v6.5.0. A chained ShadowTLS front plus its Shadowsocks detour target
  now runs in one shared sing-box process/config, with one root owner, and the
  runtime test `TestSingboxAuditBundledShadowTLSV3ManagedMultiInboundRuntime`
  passed in a private network namespace with a real patched sing-box binary.
- `SingboxDetourInboundID` was added to `database/model/model.go` and hidden
  from JSON (`json:"-"`) so it stays an internal panel relation, not a
  user-edited setting.
- `web/service/singbox_chain.go` resolves detour edges into process groups,
  validates cycles/self-detours/missing/non-sing-box targets, and provides
  `singboxDetourTarget`/`singboxDetourMember` helpers that survive tag renames.
- `web/service/singbox.go` was refactored from per-inbound processing to
  process groups for build/render/generate/restart/refresh/traffic/apply.
- `web/service/singbox_inbound.go` compares previous/new groups and restarts
  affected groups on rollback paths; also persists an already-expired/depleted
  listener as disabled.
- `web/service/inbound.go` deletion stops affected groups, clears detour
  references, and restarts remaining groups.
- `web/service/singbox_base.go` and `web/service/singbox_clients.go` are
  group-aware for rendering/validation.
- UI support was added: `web/html/form/protocol/singbox.html` has a detour
  target selector, and `web/html/modals/inbound_modal.html` has
  `singboxDetourTarget`/`singboxDetourTargets`/`singboxSetDetourTarget`.
- The browser smoke exposed and the local tree fixed a separate UI omission:
  `Protocols.SHADOWTLS` and the `ShadowTLS` picker label are now present in
  `web/assets/js/model/inbound.js`, so the Add inbound dropdown can create a
  ShadowTLS inbound instead of requiring an API-created draft.
- `web/service/singbox_chain_test.go` gained pure unit tests plus a stale-tag
  recovery assertion; `singbox_protocol_runtime_test.go` gained the managed
  multi-inbound runtime test.
- `migrateDetourInboundIDs` backfills stable target IDs transactionally from
  valid legacy raw tags without changing the user's settings JSON. It runs
  before sing-box startup and during process restart; the migration is
  idempotent and does not persist partial updates for an invalid graph.
- `gofmt` and Linux-targeted `go test -c ./web/service` passed for the changed
  package. The focused group/render/chain tests passed in a private network
  namespace on `sweden`.
- Full Linux `CGO_ENABLED=1 go test -count=1 -timeout=15m ./...`, `go vet ./...`,
  and `go build ./...` passed in a private mount/network namespace with loopback,
  a namespace-local `example.com` DNS fixture, `GODEBUG=netdns=go`, and the lean
  geo asset path. The opt-in managed ShadowTLS multi-inbound runtime test ran
  against patched sing-box and passed.
- Final refreshed isolated amd64 candidate (after syncing the UI templates):
  `/home/ubuntu/vpn-aio-managed-accounts-20261125/build/out/VPN-AIO-amd64`,
  271,661,024 bytes, SHA-256
  `cb476c47d61ea93011011c11e8cb421e786a296173a7bae1791295d0a1834914`.
  The `vpn-aio-limits` marker is embedded and `-v` reports 6.1.0. Built with
  `bash ./build.sh --quick --skip-bundle --geo-lean`, reusing cached core and
  daemon bundle. This is a verification candidate, not a release artifact.
- Rebuilt isolated candidate after the picker fix:
  `/home/ubuntu/vpn-aio-managed-accounts-20261125/build/out/VPN-AIO-amd64`,
  271,661,032 bytes, SHA-256
  `6bdbeb42dcdb7f55199235a83428d46466baa3bd2a77fa7428a3bded741cbae2`;
  `-v` reports 6.1.0. It reused the cached patched core and was copied only
  into the disposable UI container for smoke testing.
- Fresh observation (2026-11-27, this session): the 13-distro disposable
  container userland lifecycle matrix passed 13/13 for the rebuilt picker-fix
  candidate (`test_unit/container-matrix.py --binary build/out/VPN-AIO-amd64
  --expected-version 6.1.0`, run as root in
  `/home/ubuntu/vpn-aio-managed-accounts-20261125` on sweden). Reported
  results path:
  `/home/ubuntu/vpn-aio-managed-accounts-20261125/test_unit/results/containers-20261127-035332`.
  The matrix script reports "container userland only; no VM/systemd/kernel
  qualification". After the run, that workspace/results path was not found on
  sweden during a later check; the pass result is recorded from the observed
  command output, not a re-read of `results.json`.
- The earlier candidate hash `a079aeba5b2db399814fe5512cc83be8c4ee5dd066f4022104c791aa69601401`
  (271,609,888 bytes) is superseded by the rebuilt candidate above.
- Previously recorded state: four pre-existing review/audit containers and
  port 12301 owned by PID 724485. This continuation used private namespaces
  and did not access those containers or the listener.
- Commit `122e8866b5e294254144b16a8d5ba88e681294cf` and the picker/version
  follow-ups are included in public v6.5.0. No production deployment was made.

## Work State

### Completed (chaining stage)
- Added `SingboxDetourInboundID` to `database/model/model.go`.
- Added `web/service/singbox_chain.go` with group resolution, cycle validation,
  helper functions, and unit tests in `web/service/singbox_chain_test.go`.
- Refactored `web/service/singbox.go` build/render/generate/restart/refresh/
  traffic/apply logic from per-inbound processing to process groups.
- Updated `web/service/singbox_inbound.go` to compare previous/new groups and
  restart affected groups on rollback paths.
- Updated `web/service/inbound.go` deletion to stop affected groups, clear
  detour references, and restart remaining groups.
- Updated `web/service/singbox_base.go` and `web/service/singbox_clients.go`
  for group-aware rendering and validation.
- Added ShadowTLS detour target selection/editing in the inbound form/UI.
- Synced stale raw `detour` tag to the persisted stable id; a target's
  generated tag can change without breaking the chain.
- Ran `gofmt` and Linux-targeted `go test -c ./web/service` successfully.
- Ran the focused chaining/runtime tests in a private network namespace on
  `sweden`; the managed multi-inbound runtime test passed.
- Backfill/normalization of existing raw `detour` tags into
  `SingboxDetourInboundID` is now persisted by an idempotent startup/restart
  migration; focused regression confirms it preserves raw settings and is a
  no-op after the first successful pass.

### Completed private-candidate browser verification
- Browser smoke was completed in disposable container
  `vpn-aio-ui-final-20261127` through SSH tunnel `127.0.0.1:22345`, then the
  container and tunnel endpoint were removed. Login, inbound list, and the
  Add/Edit inbound modal rendered. The patched sing-box 1.12.14 (SHA-256
  `6ea33f44f4d1bec1c640538e637378032786be0b9282618f851779a405adfc35`) was
  installed and provisioned in that container DB only. The Shadowsocks target
  on port 23457 was created in the UI. Before the picker fix, the ShadowTLS
  front on port 23458 was created through the API; its UI detour selector then
  listed only the managed sing-box target, selecting it wrote raw
  `detour: "inbound-127.0.0.1:23457"`, and saving returned HTTP 200.
  The resulting runtime had one active `server-2` sing-box process containing
  both inbounds and one stats API group; the old standalone `server-3` process
  was not running. The saved group config contained both tags and the detour.
  After adding `Protocols.SHADOWTLS` and its label, the rebuilt candidate's Add
  dropdown visibly listed `ShadowTLS`; a new UI-created ShadowTLS inbound on
  port 23459 was accepted with HTTP 200 and persisted with version 3, a users
  array, and the entered handshake server. The UI-created no-detour inbound
  ran as a separate `server-4` process, confirming group separation.
- The full end-to-end chained traffic path is covered by the passing private
  Go runtime test, not by this browser run. Browser clear-and-resave after the
  final rebuild was not repeated. No production/live listener or database was
  touched; port 12301 remains untouched.
- The picker fix is committed/pushed as `359ee7f4` and released in v6.5.0.
  The later full harness passed 60/60 and API/process detour checks passed
  18/18 on candidate `6bdbeb42...` (see Latest session).
- Preserve pre-existing local documentation changes; this continuation adds
  release-verification evidence without a new commit/push.

### Remaining coverage and constraints
- CI built/published ARM64 successfully, and the public binary/core version
  probes passed under temporary user emulation. Full ARM64 runtime and physical
  mobile/systemd/kernel qualification remain unperformed.
- The live listener on port 12301 must remain untouched; network tests require
  private namespaces.

## Next Move
1. Requested continuation through release CI/assets verification is complete;
   no pending release workflow or source fix was identified in this stage.
2. If requested, separately scope full ARM64 runtime, VM/systemd/kernel and
   physical mobile qualification; existing container tests do not cover those.
3. Production deployment requires separate user authorization. This release
   verification is not authorization to replace the live panel or its database.

## Relevant Files (chaining stage)
- `database/model/model.go`: new `SingboxDetourInboundID` field.
- `web/service/singbox_chain.go`: detour graph and process-group logic.
- `web/service/singbox_chain_test.go`: pure unit tests for chaining validation.
- `web/service/singbox.go`: group-aware config generation and runtime.
- `web/service/singbox_inbound.go`: save/rollback group handling.
- `web/service/inbound.go`: deletion cleanup and group restart.
- `web/service/singbox_base.go`: group-aware base config validation.
- `web/service/singbox_clients.go`: group-aware account projection rendering.
- `web/html/form/protocol/singbox.html`: ShadowTLS UI, detour control.
- `web/html/modals/inbound_modal.html`: inbound form, detour control.
- `web/assets/js/model/inbound.js`: ShadowTLS protocol-picker constant/label.
- `test_unit/export_test/singbox.test.js`: picker regression assertion.
- `web/service/singbox_protocol_runtime_test.go`: managed multi-inbound
  runtime test.

## Important Context
- Worktree is detached at `a52abcd71639f74c6d3b3f57102ea435125ae4eb`, the
  release/main commit. Only status documents are dirty. Unrelated changes in
  the original sibling `VPN-AIO` checkout must be preserved.
- The earlier exact candidate
  `a079aeba5b2db399814fe5512cc83be8c4ee5dd066f4022104c791aa69601401` is
  superseded by the final candidate hash recorded above.
- The previously recorded state had four review/audit containers and port
  12301 owned by PID 724485. This continuation did not access either.
- v6.5.0 is published and verified; production deployment has not been made.

## Decisions (chaining stage)
- Sing-box policy remains account-owned in `accounts`/`client_traffics`;
  native `users` carry credentials only.
- Speed/IP enforcement uses shared `speedlimits.json` sidecar consumed by
  patched sing-box via `XRAY_SPEEDLIMIT_FILE`.
- Physical-device counting is unsupported and rejected; `StreamLimit` is the
  honest concurrent-stream cap.
- ShadowTLS chaining is modeled as a detour graph: `SingboxDetourInboundID`
  plus raw `detour`, resolved into connected process groups.
- Each group has one root owner for process/config/stats API port; detour
  edges are resolved to current generated tags.
- Cycles, self-detours, missing targets, and non-sing-box targets are
  rejected.
- Deleted targets clear detour references and old standalone processes are
  stopped before group startup.
- Chained process user stats use `InboundId = 0` to avoid misattributing
  account traffic, while per-member inbound rows remain emitted.
- The persisted stable id is authoritative within a process group and
  survives target tag renames; an explicitly changed raw tag still wins when
  it resolves.

## Verification (chaining stage)
- `gofmt` passed for changed Go files.
- Linux-targeted `go test -c ./web/service` passed.
- Focused chaining/runtime tests passed in a private network namespace on
  `sweden` against patched sing-box.
- `node test_unit/export_test/singbox.test.js` passed locally.
- Fresh observation (2026-11-27): picker-fix candidate `6bdbeb42...` passed the
  13-distro disposable container userland lifecycle matrix (13/13) on sweden
  (`test_unit/container-matrix.py --binary build/out/VPN-AIO-amd64
  --expected-version 6.1.0`, run as root). Container userland only; not
  VM/systemd/kernel qualification. Reported results path was not found during a
  later sweden check; recorded from observed command output.
- The frontend regression exercises detour-selector behavior: it filters the
  current inbound and Xray-only targets, writes the selected raw `detour`, and
  removes it when cleared. This is VM-based Node coverage, not a browser smoke.
- Full Linux `go test -count=1 -timeout=15m ./...`, `go vet ./...`, and
  `go build ./...` passed after the migration and chaining changes. Network
  tests ran in a private mount/network namespace with loopback and namespace-
  local DNS; no host listener was involved.
- The full suite included `TestSingboxMigrationBackfillsStableDetourID` and
  `TestSingboxAuditBundledShadowTLSV3ManagedMultiInboundRuntime`; both passed.
- Refreshed candidate version/hash/size/marker are recorded in Current State.
- `git diff --check` passed; Git emitted existing LF/CRLF working-copy notices.

## Failed Attempts / Lessons (chaining stage)
- Release verification: an initial PowerShell/SSH trap quoting attempt failed
  before downloading; removed its empty temporary directory. Prefer a literal
  PowerShell here-string piped to `ssh ... 'python3 -'` for multi-line scripts.
  Public downloads need an executable bit on Linux before `-v`; after applying
  it only to the temporary files, both version probes passed.
- The first managed multi-inbound runtime test failed because the loopback
  interface was down in the private namespace; bringing `lo` up fixed it.
- The first `SingboxDetourInboundID` was exposed as a JSON field; it was
  hidden (`json:"-"`) because it is an internal relation, not a user setting.
- The first detour-target resolution preferred the persisted id over the raw
  tag, which prevented editing the target; raw tag now wins when it resolves,
  with the persisted id as fallback for renamed targets.
- The empty-member process-stop condition was broadened so a chained process
  also stops when any member has no enabled managed users.
- The first full-suite launch exited before tests because `sudo` reset `PATH`
  inside the private namespace (`go: not found`). Re-exporting
  `/usr/local/go/bin` inside the namespace allowed the complete suite, vet, and
  build to pass. The initial isolated service-suite attempt also lacked the
  namespace DNS/geodata fixtures and observed a network-sensitive SSH UDP
  accounting failure; the corrected full run with fixtures passed.
- The first chaining candidate build used an isolated source copy whose two UI
  templates predated the detour selector. Hash comparison exposed the mismatch;
  both templates and the frontend regression file were synced and hash-matched,
  then the candidate was rebuilt. The intermediate hash
  `650197fc232d06d4734268f1d1484c0bd4d6b5e48000dc81e6b33c25ada5506f` is
  superseded by the final hash above.
- The first frontend behavior assertion ran before the test reopened the edit
  modal, so the current inbound id was unset and the self row was not filtered.
  Moving the assertion after modal setup fixed the test; the corrected Node
  regression passed without a product-code change.

## Historical state (earlier remediation; retained summary)

- (Historical) The worktree was then detached and uncommitted at
  `60951c62e3d5624cb1694868ac1387ae6d1d2144`. Superseded: on 2026-11-27 HEAD is
  `a52abcd7` (= v6.5.0) and only PROGRESS.md/HANDOFF.md are modified.
  The published/installed 6.1.0 binary was not replaced by this local work.
- Earlier managed-account work maps native sing-box `users` into
  `accounts`/`account_inbounds`/`client_traffics`, keeps native users
  credentials-only, projects account edits across memberships, and supports
  quota/expiry refresh and account traffic statistics through the V2Ray API.
  Speed/IP policy is enforced by the shared sidecar in patched sing-box;
  `UserLimitOverride > 0` is rejected because physical-device counting is not
  supported, and `StreamLimit` caps concurrent routed streams/UDP associations.
- The patched sing-box 1.12.14 artifact used for isolated runtime tests is
  `/tmp/vpn-aio-managed-accounts-singbox-limits-20261126/sing-box`, SHA-256
  `6ea33f44f4d1bec1c640538e637378032786be0b9282618f851779a405adfc35`.
  Earlier runtime tests covered stats/quota, rate/IP/stream limits, TUIC,
  AnyTLS, Hysteria2, Shadowsocks, VMess, Trojan, and ShadowTLS v3 auth.
- The local `Bug.md` triage fixed reproduced BUG-06..BUG-10 (DB import core
  restart/account backfill, base-config inbound validation, node token/name/
  IPv6 handling, API-token scope validation, and partial settings merge).
  BUG-03..05 were already addressed in the checkout. BUG-01/02 were not
  reproduced in raw sing-box settings and were left unchanged.
- Before the chaining continuation, the isolated candidate
  `a079aeba5b2db399814fe5512cc83be8c4ee5dd066f4022104c791aa69601401`
  passed the corrected 13-distro lifecycle matrix (13/13, 30/30 each), the
  installer contract (8/8), and the corrected live UI harness (60/60). Those
  results apply to that earlier candidate.
- Intermediate candidate `cb476c47...` (refreshed UI templates) is superseded
  by the picker-fix candidate `6bdbeb42...`.
- Fresh observation (2026-11-27): picker-fix candidate `6bdbeb42...` passed the
  13-distro disposable container userland lifecycle matrix (13/13) on sweden
  via `test_unit/container-matrix.py`; this is container userland only, not
  VM/systemd/kernel qualification. The results path reported by the run was
  not found during a later sweden check, so the pass is recorded from observed
  command output. Targeted disposable Playwright smoke for the ShadowTLS picker
  also passed on `6bdbeb42...`. Full browser harness (60/60), installer
  contract, ARM64 packaging, VM/systemd/kernel, and physical mobile checks have
  not been rerun/reported for `6bdbeb42...` in this session.
- The earlier candidate's browser checks resolved the Clients URL and
  `StreamLimit` visibility. Two font HTTP 500s and an Ant Design notification
  TypeError were observed once, then not reproduced by the corrected 60/60
  harness; no product fixes were made for those observations.
- Previously recorded test-harness failures included Windows-native Go builds
  failing on Linux-only symbols; missing `example.com` DNS fixture; wrong lean
  geo asset path; a network-sensitive SSH accounting failure; and test setup
  mistakes. The current corrected full Linux run is the authoritative latest
  result. Run network tests only in private namespaces; keep port 12301
  untouched.
- (Historical, superseded) At that stage no commit/push/release had been made.
  Current truth: commits are pushed and released as v6.5.0 (see top); still no
  production deployment.

## Important Files (chaining stage)

- `database/model/model.go`: `SingboxDetourInboundID` field.
- `web/service/singbox_chain.go`: detour graph and process-group logic.
- `web/service/singbox_chain_test.go`: pure unit tests.
- `web/service/singbox.go`: group-aware config generation and runtime.
- `web/service/singbox_inbound.go`: save/rollback group handling.
- `web/service/inbound.go`: deletion cleanup and group restart.
- `web/service/singbox_base.go`: group-aware base config validation.
- `web/service/singbox_clients.go`: group-aware account projection rendering.
- `web/html/form/protocol/singbox.html`: detour target selector.
- `web/html/modals/inbound_modal.html`: detour helpers.
- `web/service/singbox_protocol_runtime_test.go`: managed multi-inbound
  runtime test.

## Related sibling file

- `HANDOFF.md` exists in this checkout and must be kept consistent with this
  status as the diff review proceeds.
