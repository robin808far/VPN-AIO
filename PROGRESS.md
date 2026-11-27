# Project Progress & State

Last updated: 2026-11-28 (v7.0.0 published and verified; status docs re-saved)

Project root: `C:\Users\mef\Documents\OpenCode Projects\VPN-AIO`

## Current Status

- **`v7.0.0` is published and verified.** `main` = `origin/main` =
  `6f2ef77deee873f6f2551c14c0fd17e103ef0ce1` = peeled tag `v7.0.0`
  (checked with `git ls-remote`). `config/version` = `7.0.0`.
- Working tree: only `PROGRESS.md` and `HANDOFF.md` are modified (this
  state-save). No source change is in flight.
- Release CI **46069263983** and Ubuntu validation **46069258533** both
  succeeded for `6f2ef77d`. Public assets were downloaded on `sweden`:
  `sha256sum -c SHA256SUMS` OK for amd64 (409,505,056 B,
  `84da4e5f...`) and arm64 (403,888,912 B, `d0035344...`); `VPN-AIO-amd64 -v`
  = `7.0.0`; unauthenticated latest-release API = `v7.0.0`.
- Live host (`sweden`, port 12301) was last observed running **6.6.0**
  (user-deployed, verified 2026-11-27). It was **not** re-checked today and
  was not touched by the 7.0.0 work. Upgrading it to 7.0.0 is the user's
  action and needs explicit authorization; the assistant only verifies.

## DEFECT found 2026-11-28 (fresh observation)

- User deployed 7.0.0 on the live host; panel `-v` = 7.0.0 but
  `/opt/VPN-AIO/bin/sing-box version` = **1.12.14** (46,985,351 B = old core).
  `grep -a` on the released `VPN-AIO-amd64` finds only `1.12.14`: **the v7.0.0
  release embeds the old sing-box.** Cause: release workflow `actions/cache`
  `restore-keys: vpn-aio-build-${{ runner.os }}-` restored the old
  `backend/bin`, and root `build.sh` only checked artifact presence, so the
  daemon bundle was not rebuilt.
- Fix (local, uncommitted): root `build.sh` now marks the bundle stale when
  `backend/bin/$ARCH/sing-box` lacks `vpn-aio-limits` or the pinned
  `SINGBOX_VERSION` string. Not yet run/tested. Needs a fix release (7.0.1)
  after user authorization.
- Live host also shows `systemctl is-active vpn-aio` inactive (unit is
  `VPN-AIO.service`, active), stray sing-box PID 531113 (`server-4`, since
  Nov 25) still running; runtime dirs still named `/etc/vpn-ui-singbox`,
  socket `vpn-ui.sock` (user asked to rename all `vpn-ui` -> `vpn-aio`; not
  started).

## Stray process cleanup (2026-11-28, done)

- PID 531113 (`server-4`) / 510411 were NOT the live panel: they belonged to
  container `vpn-aio-audit-610-primary` (image `vpn-aio-release-21bc50d`, panel
  6.1.0, binary sha `60ec1bc3...`). Compared before removal: commit `21bc50d`
  (3.0.0 docs, old history line) has no product change missing from `main`
  (only removed donate code / old 3.0.0 notes, removed deliberately in 4.1.0);
  container-only files were test scripts in `/checks` (2 not in repo:
  `audit-fixture.py`, `audit-features.py`; 2 differ: `live/uitest.js`,
  `container-matrix.py`). Backed up to
  `/home/ubuntu/audit-610-checks-backup-20261128.tgz` (307,433 B) on sweden.
  Container stopped and removed; both PIDs gone; `VPN-AIO.service` active;
  only live sing-box `server-5` (PID 1041180) remains.

## Open User Requests

- (Order set by user 2026-11-28) 1) full `vpn-ui` -> `vpn-aio` rename with
  automatic migration of runtime paths, in 7.0.1; 2) release 7.0.1 (with
  sing-box bundle fix, verify embedded core = 1.14.5); 3) after user/assistant
  updates live host to 7.0.1, run sing-box tests there; 4) remaining section-3
  options.

- (New 2026-11-28) Sync HANDOFF.md + commit/push status docs; rename every
  `vpn-ui` name to `vpn-aio` with migration; run sing-box tests on live host
  if needed; implement remaining section-3 options. Not started.

1. **(Open) Full sing-box feature runtime test after the release.** The user
   approved this order: release 7.0.0 first, then test every implemented
   sing-box feature — config generation, real communication, stats/quota,
   limits, chaining — enough to prove correctness. This has **not** been run
   against the released 7.0.0 binaries yet (see Verification).
2. **(Open) Remaining sing-box 1.12–1.14 options not in the panel yet:** Snell
   inbound (check `users` support first, because stats/quota depend on it),
   listen keep-alive fields, cloudflared inbound, sing-box OpenVPN server,
   Hysteria2 Realm. All were approved for implementation.
3. **(Open, user decision) Live-host upgrade** to 7.0.0 and the stray old
   panel PID 510411 plus its sing-box PID 531113 (`server-4`) seen on the live
   host on 2026-11-27. Not touched.
4. **(Open, optional) Remaining qualification:** full ARM64 runtime,
   VM/systemd/kernel, physical mobile. Not performed.

## Work State

### Completed in 7.0.0 (commits `v6.6.0..6f2ef77d`)

- `072812df` UI: sing-box card "Edit config" opens the base-config editor;
  "Set Cert from Panel" in the sing-box TLS form (`singboxSetDefaultCert`);
  VPN Services chips are a two-line grid (name row, state row).
- `677dcabe` build: bundled sing-box upgraded 1.12.14 -> **1.14.5**;
  `build/backend/singbox-limits.patch` re-ported (only the
  `RoutedPacketConnection` hunk needed manual work; `limiter.go` /
  `limiter_test.go` unchanged); `build/backend/build.sh` pins
  `SINGBOX_VERSION="1.14.5"` and builder image `golang:1.25.5-alpine`
  (1.14.5 needs Go 1.25.5); the release workflow cache key now also hashes
  `build/backend/build.sh` and the patch so CI cannot reuse the cached
  1.12.14 daemon.
- `75e3acb2` removed/deprecated sing-box features migrated: the base-config
  route form emits action rules (`reject`, `hijack-dns`, `route`) instead of
  removed `block` / `dns-out` outbounds; `geosite` (removed in 1.12) becomes a
  remote `rule_set` from `SagerNet/sing-geosite` `.srs` with explicit
  `http_clients` / `default_http_client`; `http_clients` added to
  `baseConfigBlocks`; legacy saved rules still parse.
- `55c8da78` UI for new server options (taken from v1.14.5 docs): TLS
  min/max version, ALPN, `curve_preferences`, `handshake_timeout`, kTLS TX/RX
  (default off, written only when set), mTLS `client_authentication` + client
  CA paths; Hysteria2 obfs none/salamander/gecko (+ min/max packet size),
  `bbr_profile`, masquerade URL; ShadowTLS v3 `strict_mode` + `wildcard_sni`;
  DNS `strategy` / `timeout` / `optimistic`.
- `b7937e6c` docs for the option audit/UI. `23b55b5e`, `e4bc79f0` are docs
  commits recording the v6.6.0 release and PROGRESS consolidation.
- `73607345` UI fix: the protocol picker's **"Hysteria" is protocol
  `hysteria`** (Xray's name for Hysteria2), while sing-box stores it as
  `hysteria2`. `coreSelectable` now accepts `hysteria`/`hysteria2`;
  `onCoreChange` renames `_protocol`; `singboxSeed` maps `hysteria` ->
  `hysteria2`. Before this fix the Core selector never appeared for Hysteria,
  so a sing-box Hysteria2 inbound could not be created from the UI.
- `6f2ef77d` release commit: `config/version` 7.0.0.

### Verification actually performed for 7.0.0

- Source checks: `go build ./...` OK, `go vet` clean, and all
  `./web/... ./sub/...` tests passed except `TestCustomGeoValidateURL`, which
  fails only inside the isolated netns because `example.com` has no DNS there
  and passes outside it.
- Frontend: `node test_unit/export_test/singbox.test.js` PASS (model, modal,
  payload, defaults, page-listener cleanup; re-run after the Hysteria fix).
- sing-box 1.14.5: a sample config using every new field passes
  `sing-box check`; a `rule_set` config passes check and the `.srs` URL
  returns HTTP 200.
- Docker-built 1.14.5 daemon on sweden: 62,021,780 B, SHA-256
  `dfe79582aa1005dc20cb54ed2839e059f95b085d142a2ff0fa9e06fe66970bf7`,
  reports `1.14.5` with `with_v2ray_api`. `go test -run TestSingboxAudit
  ./web/service` against it: **26 PASS, 1 SKIP**
  (`TestSingboxAuditStartupFailureRestoresDatabaseAndProcess`, skipped by
  design with a real binary), **0 FAIL**.
- Browser check in disposable container `vpn-aio-ui-check-20261128` (port
  22348, SSH tunnel from Windows, user `uitest`), with the 1.14.5 binary
  copied to `/usr/local/bin/bin/sing-box`:
  - Core page: 15 cards, order XRAY, SING-BOX, L2TP, IPSEC, PPTP, ...;
    sing-box card shows "Version 1.14.5", "Locked", no "Update from source";
    "Edit config" opens "Edit Base Config — Sing-Box" with Graphic Form /
    JSON / Log level / DNS strategy / timeout / optimistic / routing rules.
  - Dashboard at 1296 px: `bo-t-sbox` tile directly under Xray with state,
    `v1.14.5`, uptime, Logs/Stop/Restart; Manage tile below it; VPN Services
    chips confirmed two-line (`state.top >= name.bottom`).
  - A **Hysteria2 inbound was created through the UI on the sing-box core**
    with gecko obfs, `bbr_profile=aggressive`, masquerade URL, TLS 1.3, ALPN
    h3, `handshake_timeout=15s`, `curve_preferences`, and panel cert/key
    paths. Save returned "Inbound has been successfully created."; the
    generated `/etc/vpn-ui-singbox/server-1/config.json` contained every
    field; the process ran.
  - Real traffic: a sing-box CLI `tools fetch` through that inbound returned
    `FETCH_RC=0`; a wrong obfs password timed out (`BAD_OBFS_RC=1`), a valid
    negative control.
  - Container and its temp files were removed afterwards.

### Not tested / unknown (do not claim as done)

- **Embedded sing-box inside the released 7.0.0 binaries was not extracted
  or executed.** Only the panel `-v` was checked. (In the 6.5.0 release the
  embedded cores were extracted and version-probed; that was not repeated.)
- **The full sing-box runtime suite was not run against the released 7.0.0
  build.** `TestSingboxAudit` ran against the docker-built 1.14.5 core in the
  source tree, not against the published binary.
- ARM64 runtime, 13-distro container matrix, installer contract, VM/
  systemd/kernel, and physical mobile were not run for 7.0.0.
- Snell, listen keep-alive, cloudflared, OpenVPN server, Hysteria2 Realm are
  not implemented.
- `/home/ubuntu/sb-1145-port` on sweden: removal was planned after the
  release but is **not confirmed**; treat as possibly still present.
- Pre-existing console 404 `/ui/ui/panel/nodes/list` (doubled base path) was
  observed earlier, unrelated, not investigated.

## Next Steps

1. Run the full sing-box runtime/feature suite against the released 7.0.0
   artifacts (config generation, real protocol traffic, stats/quota,
   speed/IP/stream limits, ShadowTLS chaining), in private network
   namespaces, without touching port 12301.
2. Implement the remaining section-3 options (Snell after checking `users`
   support, listen keep-alive, cloudflared, sing-box OpenVPN server,
   Hysteria2 Realm) with UI fields and docs.
3. Clean up `/home/ubuntu/sb-1145-port` on sweden after confirming it is no
   longer needed.
4. User decides on the live-host 7.0.0 upgrade and the stray old panel/
   sing-box processes; assistant verifies only after the user deploys.

## Decisions & Constraints

- Version **7.0.0** because sing-box jumped two major versions and the
  route-rule shape changed (block/dns-out/geosite removal).
- sing-box core updates stay **locked** in the panel (no "Update from
  source"): upstream archives lack the `vpn-aio-limits` patch, so the core
  only changes with a panel release.
- Limits model is unchanged: account-owned policy in
  `accounts`/`client_traffics`; native `users` carry credentials only; speed/
  IP enforcement through the shared `speedlimits.json` sidecar consumed by the
  patched sing-box via `XRAY_SPEEDLIMIT_FILE`; physical-device counting is
  rejected; `StreamLimit` is the honest concurrent-stream cap.
- ShadowTLS chaining stays a detour graph (`SingboxDetourInboundID` plus raw
  `detour`) resolved into process groups, one root owner per group.
- Every new option the operator must fill in is exposed in the UI; values are
  read from sing-box docs first.
- Live host changes are the user's action; the assistant does not deploy.
- Network tests only in private namespaces; the live listener on 12301 must
  stay untouched.
- kTLS TX/RX default off because the docs warn of performance loss; both keys
  are written only when set, so untouched JSON stays byte-identical.

## Changed Files & References (7.0.0)

- `config/version`: `7.0.0`.
- `build/backend/build.sh`: sing-box 1.14.5, builder `golang:1.25.5-alpine`.
- `build/backend/singbox-limits.patch`: re-ported for 1.14.5.
- `.github/workflows/vpn-aio-release.yml`: cache key includes `build.sh` and
  the patch.
- `web/service/singbox_base.go`: `http_clients` in `baseConfigBlocks`,
  action-based route rules, remote geosite `rule_set`.
- `web/html/core.html`: base-config form (route rules, DNS
  strategy/timeout/optimistic), core cards, edit-config button.
- `web/html/form/protocol/singbox.html`: TLS/Hysteria2/ShadowTLS fields,
  "Set Cert from Panel".
- `web/html/modals/inbound_modal.html`: `singboxSetObfs`,
  `singboxSetDefaultCert`, `singboxGet/Set`, `coreSelectable`, `onCoreChange`,
  `singboxSeed` (incl. the Hysteria -> hysteria2 mapping).
- `web/html/index.html` + `web/assets/css/components.css`: dashboard
  `bo-t-sbox` tile, bento grid areas, `.bo-svc-chip` two-row grid.
- `web/service/core.go`, `web/service/procmgr.go`, `web/controller/core.go`:
  core order, `extra.uptime`/`OldestStartWithPrefix`, locked `/singbox/update`.
- `web/service/singbox_accounts.go`, `database/model/model.go`: Hysteria2
  handling and `IsHysteria`.
- `test_unit/export_test/singbox.test.js`: frontend regressions.
- `HANDOFF.md`: kept consistent with this status.

## Failed Attempts & Lessons

- The temp UI container was built with `--skip-bundle` and a cached bundle
  without sing-box, so the card read "NOT INSTALLED". Fixed by copying the
  built 1.14.5 binary to `/usr/local/bin/bin/sing-box` in the container.
- The Hysteria picker bug (above) only surfaced in the browser check: the
  form compared `block.type` to `ib.protocol`, so the rename to `hysteria2`
  is required in `onCoreChange`, not just in the seed.
- A Windows copy built on sweden needs
  `find . -name "*.sh" -exec sed -i "s/\r$//" {} +` first; CRLF breaks
  `set -o pipefail` in `build.sh`.
- CI would have reused the cached 1.12.14 sing-box bundle; the cache key now
  hashes `build.sh` + patch.
- Full-suite runs inside `sudo -n unshare -n` need `PATH=/usr/local/go/bin`,
  `ip link set lo up`, `GODEBUG=netdns=go`, `GOCACHE`, `GOMODCACHE`, and
  `VPN_AIO_TEST_SINGBOX`; otherwise `go: not found` or network fixtures fail.
- `TestCustomGeoValidateURL` fails only in the isolated netns (no DNS for
  `example.com`); it passes outside. Not a product defect.
- Multi-line remote scripts: prefer a literal PowerShell here-string piped to
  `ssh ... 'python3 -'`; ad-hoc quoting traps failed before.

## Historical (pre-7.0.0; retained summary)

- **v6.6.0** (tag `1fb52746`, 2026-11-27): sing-box "Update from source"
  removed and `/singbox/update` locked; `GetCoresStatus` puts sing-box after
  Xray; dashboard `bo-t-sbox` tile; Persian labels for `[pages.core]`; release
  CI 46055354122 + validation 46055349392 succeeded; assets verified. Live
  host was user-deployed to 6.6.0 and verified (`-v` 6.6.0, service active,
  no `noises` warning).
- **v6.5.1 / v6.5.0** (2026-11-27): installer `/dev/tty` defect fixed in
  `deploy.sh` (`deploy_have_tty` opens `/dev/tty`); unattended custom login via
  `PANEL_USER/PANEL_PASS/PANEL_PORT/PANEL_PATH`; `stripEmptyFreedomNoises` in
  `web/service/xray.go`; ShadowTLS picker exposed; both releases verified with
  `sha256sum -c`, `-v`, and embedded `vpn-aio-limits` cores.
- **WSL2 systemd/kernel install test** (2026-11-27): Ubuntu 24.04 with
  `systemd=true`, Microsoft WSL2 kernel (not a stock VM kernel). Install
  succeeded via pseudo-tty; `systemctl restart`, `kill -9` auto-restart, and
  distro restart survived; panel HTTP 200 on localhost. Test distro and
  `.wslconfig` removed. Not a full VM/kernel qualification.
- **Panel-managed ShadowTLS chaining** (v6.5.0): detour graph with
  `SingboxDetourInboundID` (`json:"-"`), `migrateDetourInboundIDs` backfill,
  group-aware build/render/restart/traffic/apply, cycle/self/missing/non-
  sing-box rejection, `InboundId = 0` for chained user stats. Runtime test
  `TestSingboxAuditBundledShadowTLSV3ManagedMultiInboundRuntime` passed in a
  private netns. UI detour selector + `Protocols.SHADOWTLS` picker added.
- **Managed-account remediation** (v6.5.0): native sing-box `users` mapped to
  `Account` / `AccountInbound` / `ClientTraffic`; account edits project back;
  quota/expiry enforcement; `Bug.md` BUG-06..BUG-10 fixed (DB import core
  restart/account backfill, base-config inbound validation, node token/name/
  IPv6, API-token scopes, partial settings merge); BUG-01/02 not reproduced.
- **Earlier verification** on candidate `6bdbeb42...`: 13-distro container
  matrix 13/13 (30/30 each), installer contract 8/8, live UI harness 60/60,
  API/process detour checks 18/18. These belong to that 6.1.0-era candidate,
  not to 7.0.0.
- **Release asset evidence retained from earlier versions** (all downloaded on
  sweden, `sha256sum -c` OK, `-v` checked):
  - v6.5.0: amd64 409,479,064 B `9a26f831...`; arm64 403,822,560 B
    `9224c545...`; SHA256SUMS `db30cbdb...`. Embedded sing-box 1.12.14 cores:
    amd64 46,985,351 B `6ea33f44...`, arm64 45,219,975 B `00c6e6bb...`, both
    with the `vpn-aio-limits` marker.
  - v6.5.1: amd64 409,479,888 B `052cecd0...`; arm64 403,822,952 B
    `97126a13...`; SHA256SUMS `d04a9309...`.
  - v6.6.0: amd64 409,488,560 B `80ca3a74...`; arm64 403,823,312 B
    `7121e090...`; SHA256SUMS `223e6db4...`.
  - 6.1.0-era candidates: `a079aeba...` 271,609,888 B; `cb476c47...`
    271,661,024 B; `6bdbeb42...` 271,661,032 B.
- **Patched 1.12.14 sing-box** used for the earlier runtime work:
  `/tmp/vpn-aio-managed-accounts-singbox-limits-20261126/sing-box`, SHA-256
  `6ea33f44f4d1bec1c640538e637378032786be0b9282618f851779a405adfc35`
  (static, reports 1.12.14 with `with_v2ray_api`).
- **Project relocation** (2026-11-27): main checkout moved to
  `C:\Users\mef\Documents\OpenCode Projects\VPN-AIO`; old
  `Default Project\VPN-AIO` deleted; node-feature WIP preserved on
  `backup/local-node-wip`; review worktree
  `Default Project\VPN-AIO-review-6.1.0` still attached.
- **sweden housekeeping** (2026-11-27): disk 87% -> 63%; kept
  `/home/ubuntu/vpn-aio-managed-accounts-20261125` (build cache) and the
  running review/audit containers; live `/opt/VPN-AIO` untouched.
- Deeper history and the independent 6.1.0 audit are in `HANDOFF.md` and
  `docs/releases/6.1.0-audit-20261125.md`.

## Related Documents

- `HANDOFF.md`: delivery/handoff document; must stay consistent with this
  status. Updated 2026-11-28 with the v7.0.0 result.
- `docs/releases/6.1.0-audit-20261125.md`: independent pre-remediation audit.
