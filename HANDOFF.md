# HANDOFF

Last updated: 2026-11-28 (tunnel repository research; no implementation)

## Latest task — research seven tunnel repositories (2026-11-28)

- User requested deep research only, to retain features and implementation
  knowledge before giving integration requirements. No engine selection,
  implementation, installation, deployment, commit/push or release in this task.
- Persistent report:
  [بررسی هفت پروژهٔ تانل](docs/plans/tunnel-repositories-review-20261128.md).
  It pins all seven main SHAs and records source maps, topology/transports,
  config, management, metrics/limits, packaging, licensing and VPN-AIO seams.
- Important findings: Kariz's present license forbids management from another
  panel; TunTunnel lacks a license, and Overal has only an MIT badge without a
  grant found. WatTunnel is MIT; BackPack/C-Tunnel/Arange have AGPL/notices.
  C-Tunnel has 28 active and 23 legacy methods; some protocol names are shaped
  framing. Arange's frp is not official FRP. Overal uses official FRP and has an
  optional fake speed/ping display, off on a fresh install. Details are in report.
- Actual local observation: HEAD `7d688912ecc807d13f1023a0e272d5636bc1c486`,
  version `7.0.1`, initially clean. Prior release/live-host paragraphs below are
  historical checkpoints; this task did not verify releases or access the host.
- Verification is source/documentation research and read-only GitHub metadata;
  TunTunnel's Python/embedded benchmark and Overal's embedded panel were
  AST-parsed without execution; C-Tunnel's method catalog was counted. Pinned
  source/local references and whitespace passed checks; `git diff --check`
  passed with environment LF/CRLF warnings only. No runtime, throughput, kernel or ARM64
  claim. Product source and dependencies remain untouched.
- Changed documents: report, `docs/README.md`, `PROGRESS.md`, `HANDOFF.md`.
  Source/architecture research and documentation are complete. Next: await the
  user's desired design; practical runtime qualification remains unperformed.

## Prior checkpoint — v7.0.0 released and verified (2026-11-28)

- Remote `main`, local HEAD and peeled tag `v7.0.0` all resolve to
  `6f2ef77deee873f6f2551c14c0fd17e103ef0ce1`; `config/version` is `7.0.0`.
  Only `PROGRESS.md` and `HANDOFF.md` are modified (state-save, not committed).
- v7.0.0 content: bundled sing-box upgraded to **1.14.5** (limits patch
  re-ported, builder `golang:1.25.5-alpine`, release CI cache key now hashes
  `build.sh` + patch); removed features migrated (`block`/`dns-out` -> action
  rules, `geosite` -> remote `rule_set`, explicit `http_clients`); UI for the
  1.12–1.14 TLS/Hysteria2/ShadowTLS/DNS options; sing-box base-config editor,
  "Set Cert from Panel", two-line VPN Services chips; dashboard/core UI; and
  the Hysteria picker -> sing-box `hysteria2` fix.
- Release CI **46069263983** and Ubuntu validation **46069258533** succeeded.
  Public assets downloaded on `sweden`: `sha256sum -c SHA256SUMS` OK for amd64
  (409,505,056 B, `84da4e5f...`) and arm64 (403,888,912 B, `d0035344...`);
  `VPN-AIO-amd64 -v` = `7.0.0`; latest-release API = `v7.0.0`.
- Verification: `go build ./...` OK, `go vet` clean, `./web/... ./sub/...`
  tests pass except `TestCustomGeoValidateURL` (isolated-netns DNS only);
  `node test_unit/export_test/singbox.test.js` PASS; the docker-built 1.14.5
  core ran `TestSingboxAudit` with 26 PASS / 1 SKIP (by design) / 0 FAIL.
  A disposable-container browser check confirmed core-card order, lock tag,
  base-config editor, dashboard tile and two-line chips, and created a
  Hysteria2 sing-box inbound through the UI; its generated config ran and
  carried real traffic (wrong obfs password timed out as a negative control).
  The container and temp files were removed.
- Not verified: the embedded sing-box inside the released 7.0.0 binaries was
  not extracted/executed; the full runtime suite was not run against the
  released build; ARM64 runtime, 13-distro matrix, installer contract,
  VM/systemd/kernel and physical mobile were not run. Snell, listen
  keep-alive, cloudflared, sing-box OpenVPN server and Hysteria2 Realm are
  not implemented yet.
- Live host: last observed running **6.6.0** (user-deployed, verified
  2026-11-27); not re-checked today and untouched by the 7.0.0 work. The
  upgrade is the user's action with explicit authorization; the assistant
  only verifies. Stray old panel PID 510411 and its sing-box PID 531113 were
  not touched.
- Next: run the full sing-box feature runtime suite against the released 7.0.0
  artifacts, then implement the remaining approved options. See
  [PROGRESS.md](PROGRESS.md) for the detailed state.

## Historical status — v6.5.0 release verification (2026-11-27; superseded by v7.0.0 above)

- Remote main, local detached HEAD and peeled `v6.5.0` tag all resolve to
  `a52abcd71639f74c6d3b3f57102ea435125ae4eb`. Source and picker fixes are
  committed/pushed. Earlier statements below about uncommitted fixes or no
  release are historical; current evidence is here and in [PROGRESS.md](PROGRESS.md).
- [Release CI 45991545057](https://github.com/robin808far/VPN-AIO/actions/runs/45991545057)
  and [Ubuntu validation 45991539428](https://github.com/robin808far/VPN-AIO/actions/runs/45991539428)
  completed successfully. [v6.5.0](https://github.com/robin808far/VPN-AIO/releases/tag/v6.5.0)
  is public Latest, not draft/prerelease, with amd64/arm64 binaries and SHA256SUMS.
- Downloaded both complete public binaries on `sweden`: checksums match the
  downloaded manifest and GitHub digests; both execute `-v` as `6.5.0` (amd64
  natively, ARM64 under temporary QEMU user emulation). Full artifact hashes and
  sizes are in PROGRESS.md.
- Examined actual architecture-specific embedded sing-box files because CI
  reused daemon caches. Both have the `vpn-aio-limits` marker and execute
  `version` as `1.12.14` with `with_v2ray_api`. amd64 core SHA-256 matches the
  previously runtime-tested `6ea33f44...`; ARM64 core SHA-256 is `00c6e6bb...`.
- Installer caveat resolved: Git index and public raw `main` copies of both
  `install.sh`/`deploy.sh` use LF; CRLF is confined to local working copies.
  No source changes or installer execution were required.
- Removed only this continuation's verification directories (including extracted
  QEMU/sysroot); no host packages installed, panel/container started, or live
  service/database/listener changed. The original sibling `VPN-AIO` checkout's
  unfinished Nodes changes were only inspected and remain untouched.
- This continuation changes PROGRESS.md and HANDOFF.md only, without a new
  commit/push. Private candidate matrix/UI/runtime results remain attributed to
  that candidate, not the public CI binaries. Full ARM64 runtime, VM/systemd/kernel
  and physical mobile qualification remain unperformed. Release verification is
  complete; production deployment still requires separate user authorization.

## systemd install test on WSL2 Ubuntu 24.04 (2026-11-27)

- Public v6.5.0 installed and runs under systemd (enable, restart, kill -9
  recovery, distro restart all OK; panel HTTP 200). Microsoft WSL2 kernel, so
  not a full VM/stock-kernel qualification.
- Unfixed installer defect: non-interactive installs hang/fail on `/dev/tty`
  prompts (`deploy.sh` ~458/505). Details in PROGRESS.md.
- Test distro, `.wslconfig` and temp files removed (user-approved). Docs pushed;
  no source change since v6.5.0, so no 6.6.0 release.

## Historical audit — independent 6.1.0 review on sweden (2026-11-25)

The current user authorized review and isolated testing on `ssh sweden`, plus handoff updates. The published 6.1.0 sing-box integration has important defects; the local managed-account remediation below has passed isolated Linux verification but is not deployed or released. Read [the full review](docs/releases/6.1.0-audit-20261125.md) before deploying customer sing-box inbounds.

- Reviewed release `v6.1.0` at `b9dad8618c0f793b1036850f851802ff4f439ce8`, plus main `60951c62e3d5624cb1694868ac1387ae6d1d2144`. The latter changes only the CI checkout; runtime source is identical. GitHub's successful release run is <https://github.com/robin808far/VPN-AIO/actions/runs/45560156102>.
- The live host already had the official 6.1.0 panel installed at 06:34 UTC before this review. Verified `/opt/VPN-AIO/VPN-AIO-amd64`: **403,042,152 bytes**, SHA-256 `60ec1bc3714747b48db87abc283f2e944cd4ce7c8fd38040e5d6f9fcf8d31919`. Kept that exact release binary and the active systemd service.
- Installed only the optional release-bundled **sing-box 1.12.14** at `/opt/VPN-AIO/bin/sing-box` (SHA-256 `b2ae95c753335acd51a18c5a605621c82ca7520f5d17fd3e2dbdea2f159cc834`). Added `singbox` to the existing `provisionedProtocols` manifest, preserving ten prior entries. The core is left **idle**, with no production sing-box inbound. Status incorrectly leaves its version blank because it invokes `--version` instead of `version`.
- Before those host changes, made root-only rollback snapshot `/var/backups/VPN-AIO/audit-6.1.0-20261125T072851Z/`: binary, consistent SQLite backup (`integrity_check=ok`), service unit and configuration archive. Directory 0700, files 0600. Previous backups retained.
- Live production data preserved: **4 enabled inbounds (3 VLESS, 1 AWG), 1 account, 1 panel user, 0 nodes, 0 API tokens**. All three existing VLESS profiles (XHTTP/TLS, WS/TLS, XHTTP/Reality) successfully carried trusted external HTTPS. Live panel/subscription HTTPS, DB integrity and a temporary local sing-box VLESS tunnel passed; **11/11 final host checks**.

### Findings in the published 6.1.0 binary (pre-remediation)

- **Security/accounting:** a reseller granted a sing-box inbound receives unowned raw `users` credentials; raw users do not enter Clients/account management; transferred traffic is not accounted and a 1 KiB quota allowed repeated 128 KiB transfers. Do not regard sing-box limits as enforced.
- **Sing-box GUI/storage/runtime:** the draft lives on the modal Vue instance but submit reads another object; GUI submits Xray settings. Model round trips discard `core` and raw user settings. Create/edit restart before DB commit and see old state. Invalid configs are stored and runtime errors are swallowed. Engine changes are acknowledged without persisting `Core`. Several GUI defaults fail the bundled core's check.
- **Uninstall:** sing-box counts are absent from `inboundCountsByCore`; engine-aware ownership is missing from deletion. A disposable fixture reported zero inbounds, accepted core uninstall without an inbound disposition, removed the binary and left the inbound enabled. This was NOT run on the live host.
- **UI:** Inbounds → Nodes → Inbounds produces `INBOUND_SECTIONS` redeclaration; the inbound modal remains detached and does not open. Repeated visits retain WebSocket callbacks (1→2→3). Persian mobile rail dialogs are unusable because an unconditional RTL `row-reverse` overrides the mobile column layout; pane measured 32 px wide at x=766.7 in a 388 px viewport.
- **Nodes:** form-encoded API-token creation fails (`name/scopes` lack form tags); JSON token creation works. Rename is not saved. Bare IPv6 URLs lack brackets. Authentication/scopes/revocation and real prefixed worker probes/reads passed. Remote inbound/client mutation and traffic sync remain unimplemented, as previously documented.
- **Other:** fresh sing-box base-config save fails until its directory exists; raw share-link generator selects the first user, loses WS/Reality fields, can panic for Hysteria2 without obfs, and mishandles password URI encoding. Manual release dispatch does not check out its requested tag. Full evidence and source locations are in the review.

### Verification and retained evidence

- Full existing Go suite: **2,308 passing test/subtest results, zero failures** with CGO/SQLite. Four opt-in skips were then independently executed and passed: AWG kernel attributes, AWG handshake/counters/peer removal in private mount/network namespaces, OpenSSH+sshpass forwarding in a disposable container, and live MASQUE add/remove/re-enable with the exact bundled Xray.
- `go vet ./...`, all six frontend Node scripts and the installer contract passed. Existing tests did not cover the new sing-box/soft-navigation failures.
- Exact public binary: Happ HTTP **53/53**; disposable lifecycle/native-core/transport/uninstall **30/30**; legacy accounts HTTP **34/34** with the pool assertion corrected in a staged test copy to accept a valid empty list. Its missing-core OpenVPN refusal is not proof of certificate validation.
- New feature HTTP **20/29**; extended pages/auth/Reality/engine checks **29/32**; isolated sing-box uninstall safeguards **0/3**. Eleven independent direct regression expectations also failed (twelve result events include their parent test). These are unresolved product findings, not successful checks.
- Fresh lean installation **7/7**: only the base geo pair was embedded; saving the first Iran rule fetched `geoip_IR.dat` (16,350,058 bytes); real Xray validation and restart succeeded.
- Browser exercised actual UI, payload generation and navigation, including Persian desktop and a phone-width iframe. `browser-findings.json`, screenshots and HTTP/Go logs retained. No real ARM machine or current real-device Happ/global-option test; no claim of end-user validation for every native protocol on every OS.
- Server scripts/results: `/home/ubuntu/vpn-aio-audit-6.1.0-20261125/codex-review/`. Local scripts/results: `C:\Users\mef\AppData\Local\Temp\opencode\vpn-aio-610-review\`. Production credentials/subscription tokens were not printed or added to the handoff/report.
- Review checkout: `C:\Users\mef\Documents\Default Project\VPN-AIO-review-6.1.0`. The original `VPN-AIO` checkout is behind main and contains uncommitted user work; it was preserved. At the time of the initial audit no product fixes, commits, pushes or releases had been made; the uncommitted local remediation candidate is recorded below. The original checkout remains untouched.
- Source/runtime verification is complete for the exercised paths: the managed-account candidate maps native users into accounts, memberships and traffic rows, enforces supported quota/expiry, and refuses unsupported limits/operations. The 2026-11-27 continuation below also implements panel-managed ShadowTLS multi-inbound chaining. This is not a claim that sing-box is problem-free overall, and the published/installed 6.1.0 binary was not updated by the local remediation. The latest isolated full suite, vet/build, chained runtime test, and amd64 verification candidate passed; the previously recorded scoped lifecycle/browser checks belong to the earlier candidate. Remaining platform/UI qualification and separate release approval are recorded below. Older notes are historical and superseded by the latest continuation.

### Latest continuation — panel-managed ShadowTLS chaining (2026-11-27)

- Added `SingboxDetourInboundID` as an internal inbound relation (`json:"-"`) and an idempotent `migrateDetourInboundIDs` backfill. The migration resolves existing raw detour tags, persists only the relation column transactionally, and leaves settings JSON unchanged. It runs before sing-box startup and during process restart.
- Refactored sing-box runtime/config operations to operate on connected process groups. Each group has one root process/config/stats API port; rendering uses current generated tags. Cycles, self-detours, missing targets, and non-sing-box targets are rejected. Deleting a target clears references, stops old standalone processes before group startup, and restarts remaining groups. A chained process reports user stats with `InboundId = 0` to avoid double attribution while retaining per-member inbound rows.
- Added the ShadowTLS detour selector to the inbound form and handling in the modal. Local `node test_unit/export_test/singbox.test.js` passed and exercises self/Xray-target filtering plus select/clear behavior. The disposable browser smoke used SSH tunnel `127.0.0.1:22345`: login, inbound list, and Add/Edit inbound modal rendered; the sing-box Shadowsocks target on port 23457 was created in the UI. The API-created ShadowTLS front on port 23458 displayed the selector, which listed only the managed sing-box target; selecting it wrote raw `detour: "inbound-127.0.0.1:23457"`, and saving returned HTTP 200. The resulting runtime had one active `server-2` process containing both inbounds and a shared stats API group; the old standalone `server-3` process was not running. The group config contained both tags and the detour.
- The smoke also found a separate picker omission: `Protocols` had no ShadowTLS entry, so Add could not create it. Added `Protocols.SHADOWTLS` and the `ShadowTLS` label to `web/assets/js/model/inbound.js`, with regression assertions in `test_unit/export_test/singbox.test.js`. Rebuilt candidate `6bdbeb42dcdb7f55199235a83428d46466baa3bd2a77fa7428a3bded741cbae2` (271,661,032 bytes, version 6.1.0); the Add dropdown then visibly listed `ShadowTLS`. A new UI-created ShadowTLS inbound on port 23459 was accepted with HTTP 200 and persisted with version 3, a users array, and the entered handshake server. Its no-detour configuration ran separately as `server-4`. The disposable container was then removed.
- Focused group/render/migration tests passed, including `TestSingboxMigrationBackfillsStableDetourID`. The opt-in real patched-core test `TestSingboxAuditBundledShadowTLSV3ManagedMultiInboundRuntime` passed in a private network namespace; it verified the generated two-inbound process config and an end-to-end proxied HTTP request through the managed ShadowTLS → Shadowsocks chain.
- Full Linux `CGO_ENABLED=1 go test -count=1 -timeout=15m ./...`, `go vet ./...`, and `go build ./...` passed in a private mount/network namespace with loopback, local `example.com` DNS, `GODEBUG=netdns=go`, and the lean geo asset path. The suite included the managed chained runtime test.
- Final refreshed isolated amd64 candidate: `/home/ubuntu/vpn-aio-managed-accounts-20261125/build/out/VPN-AIO-amd64`, **271,661,024 bytes**, SHA-256 `cb476c47d61ea93011011c11e8cb421e786a296173a7bae1791295d0a1834914`; `vpn-aio-limits` marker confirmed and `-v` reports 6.1.0. Built with `bash ./build.sh --quick --skip-bundle --geo-lean`, reusing cached Xray/geo and daemon bundle. This is a private verification candidate, not a release artifact. An intermediate build was superseded after hash comparison showed the isolated copy lacked the new UI templates; the templates and Node regression test were synced and the final candidate rebuilt.
- Historical pre-picker-fix attempt: the existing `audit-shadowtls-target` inbound was Xray, so its core selector was disabled, and the Add dropdown omitted ShadowTLS. A first API/JSON-editor attempt also exposed the managed-users-array validation and replacement issue. This is superseded by the completed browser qualification in lines 40–41 above.
- Fresh observation (2026-11-27, this session): picker-fix candidate 6bdbeb42... passed the 13-distro disposable container userland lifecycle matrix (13/13) on sweden (	est_unit/container-matrix.py --binary build/out/VPN-AIO-amd64 --expected-version 6.1.0, run as root). The matrix is container userland only; no VM/systemd/kernel qualification. The reported results path was not found during a later sweden check; pass recorded from observed command output. Installer contract/full browser harness reruns for 6bdbeb42..., ARM64 packaging, VM/systemd/kernel, and physical mobile qualification remain pending. The previously recorded four review/audit containers and live listener on port 12301 were not accessed. Commit 122e8866b5e294254144b16a8d5ba88e681294cf was explicitly pushed to origin/main; the picker-fix follow-up remains uncommitted locally. No release or production deployment was made.

### Earlier mitigation candidate (not deployed; superseded below) — 2026-11-25

- Built `VPN-AIO-candidate` on `sweden` from the uncommitted review checkout. Build, full Go suite, `go vet ./...`, installer contract, exact sing-box defaults, `softnav.test.js`, and `singbox.test.js` passed. The candidate's HTTP regression suite passed **19/19** checks covering token form binding, node rename, fresh base config, version display, explicit refusal of unsupported raw-user/quota operations, post-commit listener start/edit/disable/enable/delete, invalid-config rollback, core uninstall safeguards, and a real local VLESS transfer.
- Rechecked Persian mobile inbound form in Chromium against the candidate at a **388×842** iframe viewport. The modal is 372px wide, uses a column layout, the Identity section is 332px wide with three visible inputs, and the document has no horizontal overflow. Browser console had no errors. Capture: `C:\Users\mef\AppData\Local\Temp\opencode\vpn-aio-610-review\persian-mobile-fixed.png`.
- At this intermediate stage raw sing-box users remained unmanaged; that limitation is superseded by the managed-account remediation below. This candidate rejected operations whose limits or ownership it could not enforce. Engine switching was explicitly rejected.
- Disposable evidence and candidate remain under `/home/ubuntu/vpn-aio-fix-610-20261125/out/`. After validation, removed only the named `vpn-aio-fix-610-ui` container and its dedicated `vpn-aio-fix-610-net` network, and stopped the local SSH tunnel on `127.0.0.1:22062`. Production panel, database, and service were not changed.

### Managed-account remediation verification (not deployed) — updated 2026-11-26

- Local review worktree: `C:\Users\mef\Documents\Default Project\VPN-AIO-review-6.1.0`, dirty and detached at `60951c62e3d5624cb1694868ac1387ae6d1d2144`. No commit, push, release, production database or live service change was made during this remediation. Preserve unrelated worktree changes.
- Native sing-box `users` now migrate/project through `Account`, `AccountInbound` and `ClientTraffic`; account policy edits project to memberships while native users retain credentials. Per-user V2Ray-compatible stats use `v2ray.core.app.stats.command.StatsService`.
- Supported account/inbound quota and expiry are applied by traffic state and listener refresh. Speed/IP policy is enforced through the shared sidecar and patched bundled sing-box; physical-device counting remains rejected because sing-box cannot enforce it honestly. Bulk edits and depleted-user sweeps fail closed. Legacy quota, expiry, enable, Telegram ID and reset paths use the account-aware service; Xray memberships request reconciliation when runtime policy changes.
- Current Go source was copied to the isolated, non-Git test tree `/home/ubuntu/vpn-aio-managed-accounts-20261125` on `sweden`. After the controller, rename-compensation and reseller bulk-policy fixes, `CGO_ENABLED=1 go test -count=1 -timeout=10m ./...`, `go vet ./...` and `go build ./...` passed in private network namespaces, and the opt-in runtime test passed again. The full test used a local dnsmasq fixture for `example.com` and `GODEBUG=netdns=go`; no external DNS or host network namespace was used.
- The opt-in real VLESS → SOCKS → local HTTP → stats/quota runtime test passed in a private network namespace. Its temporary sing-box binary was built from tag `v1.12.14`, source revision `f56d9ab94569d46a48bfc2bcd97eea566ad365c8`, using Go 1.23.1 and the `with_v2ray_api` build tag, under `/tmp/vpn-aio-managed-accounts-singbox-build/sing-box`. This focused test binary reported `version unknown` because the build omitted version-stamping linker flags.
- Follow-up stamped build: the release-script sing-box command was rebuilt with resource caps from tag `v1.12.14`, verified source revision `f56d9ab94569d46a48bfc2bcd97eea566ad365c8`, all configured tags including `with_v2ray_api`, and the version linker flag. `/tmp/vpn-aio-managed-accounts-singbox-stamped-20261126/sing-box` is statically linked and stripped, reports version `1.12.14` and the expected tags/revision, and has SHA-256 `84b794b4bf2e2185c3657f88b08e672e52b2163624a1708011457ba4355782b0`. The real VLESS/stats/quota test passed again against this exact artifact in a private network namespace.
- Historical, superseded: at the initial managed-account verification stage, `backend/bin` and `corebundle/core` contained only `.gitkeep`, and the pinned submodules were not initialized. The capped amd64 bundle was later built successfully from cached pinned inputs; details are in the bug-report/runtime section below.
- Historical at the initial verification stage: runtime integration covered VLESS only, and packaged behavior/browser workflows had not yet been tested. Later protocol and release-preparation results are recorded below. The live host listener on port 12301 was left untouched; do not run tests in its network namespace.
- A full-suite rerun once failed in `TestSshServerTCPPath` with directional counters `up=0, down=18`; that test passed alone and the subsequent complete suite passed. An earlier full-suite attempt lacked the namespace DNS fixture and failed `TestCustomGeoValidateURL`; local dnsmasq plus `GODEBUG=netdns=go` resolved the fixture issue. The stale unsupported-policy test was corrected to treat inbound expiry and traffic reset as supported, with behavioral coverage.
- Resolved in this stage: backend save/runtime validation now requires integer `version: 3` for every ShadowTLS inbound, including an empty `users` list; the graphical form selects version 3 and blocks other values. This prevents later membership projection from producing a managed v1/v2 inbound that sing-box cannot enforce.
- Current second-pass fixes: deleting an account from one Xray inbound or deleting its home inbound now preserves the panel-wide quota/IP rows while another membership (including sing-box) remains, and re-homes the traffic row. Final membership removal cleans the rows and account mirror. Mixed Xray/sing-box bulk batches are atomic and fail closed. Manual traffic edits reject negative counters; manual/bulk traffic changes and reseller quota write-back refresh affected sing-box users. Account-mirror queries inside write transactions use reset GORM sessions so earlier query clauses cannot leak.
- New memberships are preflighted before client settings or reseller reservations are written: add-client checks all posted emails as a set on each requested inbound; update-client checks only memberships newly requested, excluding the existing set and anchor inbound. The membership transaction rechecks admission. If a later projection fails, add/update best-effort syncs the anchor account, reconciles saved/requested inbounds, returns an explicit partial-failure response, and retains any charge for the persisted client/quota change. Batch add continues attempting the remaining accounts and reports each projection failure.
- Client rename failures are explicit partial failures too. A reseller ledger rename failure stops before account projection. If account-row rename fails after the ledger moved, projection stops, saved inbounds are reconciled, and the ledger key is best-effort restored; any restoration error is included in the response. Regression tests cover account duplication prevention, ledger failure, and ledger restoration. This closes the last open review question (rename errors only logged after the primary write), so no code-review blocker remains in these paths.
- Reseller bulk write-back now updates `accounts.total_gb`/`expiry_time` and `client_traffics.total`/`expiry_time` atomically, in addition to Xray client settings. Native sing-box `users` remain credentials-only; the account policy update plus post-commit refresh is authoritative. A sing-box-only regression verifies a later `ProjectAccount` cannot restore stale policy.
- Direct-settings review found no other generic sing-box policy writer: Xray config generation skips sing-box, account projection/rename are core-aware, host writes only touch host-rule metadata, and VPN/peer/relay migrations and writers are protocol-scoped.
- Regression coverage now includes add/update admission-before-write, multi-account IKEv2 PSK batch refusal, post-write partial failures with retained reseller charges, rename failures/ownership compensation, and sing-box-only reseller quota/expiry write-back. At that stage the full `go test ./...`, `go vet ./...`, `go build ./...`, local Linux compile-only checks and opt-in real VLESS runtime test passed on the isolated source copy. `git diff --check` passes locally with existing LF/CRLF warnings.
- Historical at that stage: ShadowTLS v3 authentication and core-level Shadowsocks detour passed, while ARM64 packaging and packaged/browser workflows had not yet been tested. The panel still runs one inbound per process, so it does not manage a multi-inbound chain. The admission preflight and primary write are not one transaction; a concurrent capacity change may still become a truthful post-write partial failure. Latest smoke results follow below.

### Bug-report fixes (2026-11-26, not deployed)

Triaged `C:\Users\mef\Desktop\Bug.md` against this checkout; only reproduced issues were changed.

- Fixed BUG-06: database import/restore now backfills accounts and restarts **both** managed cores. `ServerService.ImportDB` runs `MigrationAccounts` + WireGuard reconcile after `MigrateDB`, then `RestartImportedServices` (attempts Xray and sing-box independently, `errors.Join` on failure). `ImportForeignDB` and the import controller use the same path, including a both-core retry when import fails after the DB swap. A regression imports an old-format DB containing a raw sing-box user, then asserts the account row and a running listener.
- Fixed BUG-07: saving a base config that contains `"inbounds"` now fails with an explicit message, and the proposed base is validated against every enabled inbound before storing. Legacy stored base configs with `inbounds` are still ignored during rendering so an upgrade cannot drop a running listener. `checkConfigText` also creates the sing-box directory, fixing the fresh-install save failure.
- Fixed BUG-08: node add/update binds a separate `nodeForm` that accepts `token` (and `basePath`/`syncMode`); an empty token is a 400 (`ErrNodeTokenRequired`). `model.Node.Token` remains `json:"-"`, so the secret is never returned. Update now persists the name, rejects an empty address, and bare IPv6 addresses are bracketed in the URL.
- Fixed BUG-09: API-token creation trims/splits/dedupes scopes and rejects anything outside `node-sync`/`node-admin` (`ErrInvalidTokenScope`); the controller returns 400, and the token form now has form tags. `HasScope` is nil-safe.
- Fixed BUG-10: settings updates load the stored settings and merge the submitted JSON/form body, so partial updates keep omitted fields and no longer fail `WebPort` validation.
- BUG-03/04/05 were already addressed in this checkout (sing-box traffic job; `version` subcommand for `sing-box`; `RefreshInbound` after committed inbound writes) and were not re-changed. BUG-01/02 are not exercised by the Xray-defaults code path for raw sing-box settings; the stamped-core TUIC and AnyTLS client/server tests now pass real loopback HTTP transfers.
- The report's unnumbered “Other” code claims are already covered in this checkout: fresh base save has `TestSingboxAuditFreshBaseSaveAndNullRejection`; sing-box share links have regressions for requested-user selection, WebSocket/REALITY, optional Hysteria2 obfs and credential escaping; manual release dispatch checks out `refs/tags/${inputs.tag}`. These were inspected and not re-changed.
- Final isolated Linux verification: after synchronizing the current 641-line runtime test file (SHA-256 `0e6c03d372ec4385cc1f94b17232101ce4397c8d2b73cbe4d4ce73314382476c`), full `CGO_ENABLED=1 go test -count=1 -timeout=10m ./...`, `CGO_ENABLED=1 go vet ./...`, and `CGO_ENABLED=1 go build ./...` all passed. The full suite ran in a private network namespace with stamped sing-box 1.12.14, namespace-local DNS for `example.com`, `GODEBUG=netdns=go`, and `XRAY_LOCATION_ASSET` set to the lean geo directory.
- Opt-in runtime coverage passed against the stamped core: VLESS stats/quota, TUIC/AnyTLS, Hysteria2/Shadowsocks/VMess/Trojan, ShadowTLS v3 listener startup, and ShadowTLS v3 authentication through a test-only Shadowsocks detour with a wrong-password negative control. The temporary config adds a second inbound only in the test; panel-managed multi-inbound chaining remains unverified. Frontend `singbox.test.js`, Linux compile-only service check and `gofmt` passed locally; `git diff --check` reports only the existing HANDOFF LF/CRLF warning.
- The capped amd64 full build on `sweden` completed successfully: `build/out/VPN-AIO-amd64` is 271,554,248 bytes, SHA-256 `407f2fd5067d5b55a7853ef0af1cef16467968844428ef3fa4882f485d586467`. Build output confirmed sing-box 1.12.14 with `with_v2ray_api` (revision `f56d9ab94569d46a48bfc2bcd97eea566ad365c8`), telemt 3.4.23, lean geo files, Xray, OpenVPN, strongSwan, ocserv and the other amd64 VPN bundles. The build used `GEO_LEAN=1` and Docker caps of 1 CPU/1.5 GiB memory/2.8 GiB memory+swap.

  An earlier fresh suite had a single fixture failure in `TestWgcXrayConfig`: the lean bundle keeps geo files in `corebundle/core/`, while the test's default lookup was beside the Xray binary. Setting `XRAY_LOCATION_ASSET=/home/ubuntu/vpn-aio-managed-accounts-20261125/corebundle/core` corrected it; the latest full suite passed including the new ShadowTLS test.

- Historical, superseded by the 2026-11-27 continuation above: panel-managed ShadowTLS multi-inbound chaining was unavailable in the one-process-per-inbound design, although core authentication/detour behavior passed in a temporary combined config. ARM64 packaging and NC/OPN decisions remain open. No commit, push, release, production database or live-service change was made as part of the local remediation; earlier audit-host provisioning is recorded above. Port 12301 was not touched by tests.

- Core remediation and source/runtime verification are complete for tested paths; the later browser continuation resolved the previously open UI issue. Preserve the dirty, uncommitted tree; no commit, push, release or production change is authorized. ARM64 packaging requires a runner with the cross-compiler and QEMU/binfmt setup; broader device/distro coverage and NC/OPN decisions remain follow-up items. The live port 12301 listener remains untouched.

### Historical status checkpoint — 2026-11-26 (release-preparation smoke; superseded by the final continuation below)

- Continuation added speed/IP/stream enforcement for sing-box through the shared `speedlimits.json` sidecar consumed by a patched bundled sing-box via `XRAY_SPEEDLIMIT_FILE`. Physical-device counting (`UserLimitOverride > 0`) is rejected; a separate `StreamLimit` caps concurrent routed streams/UDP associations.
- Added `StreamLimit` to `model.Account`, projection, migration, account list, sidecar generation, UI, and i18n.
- Patched sing-box 1.12.14 tree enforces sidecar speed/IP/stream limits in the v2rayapi stats tracker; patch exported to `build/backend/singbox-limits.patch`; `build/backend/build.sh` applies it and verifies the `vpn-aio-limits` marker.
- Patched sing-box was rebuilt in Docker on `sweden` from tag `v1.12.14` + patch. Final artifact `/tmp/vpn-aio-managed-accounts-singbox-limits-20261126/sing-box` is static, reports version 1.12.14, and has SHA-256 `6ea33f44f4d1bec1c640538e637378032786be0b9282618f851779a405adfc35`.
- Focused Linux verification passed on the isolated copy: policy publication, physical-device rejection, unsupported-policy validation, and the complete runtime sidecar test; `go vet ./web/service` and `go build ./...` also passed.
- The first runtime attempt exposed a real integration defect: returning a closed connection after admission rejection still allowed route processing to reach the outbound. The final patch returns explicit rejected TCP/UDP wrappers with limit errors. The final `TestSingboxAuditBundledRuntimeEnforcesSidecarSpeedIPAndStreams` passed in a private namespace: rate limiting, second-source-IP rejection, concurrent-stream rejection, and stream-slot release were all verified.
- Full Linux `CGO_ENABLED=1 go test -count=1 -timeout=10m ./...`, `CGO_ENABLED=1 go vet ./...`, and `CGO_ENABLED=1 go build ./...` passed after the final continuation edits. Network tests used a private namespace with loopback enabled, namespace-local dnsmasq for `example.com`, `GODEBUG=netdns=go`, and the lean geo asset path.
- The isolated amd64 candidate was rebuilt after copying the exact patched sing-box binary into `backend/bin/amd64/sing-box`. `build/out/VPN-AIO-amd64` is 271,605,816 bytes with SHA-256 `06a71ab045b94b17f31c03e95bf3ac9e7c373c747e875c32ad07a5d4309b5aec`; the embedded `vpn-aio-limits` marker was confirmed. This is not a release artifact.
- The final sidecar runtime test was rerun using the exact copied `backend/bin/amd64/sing-box` path from the isolated candidate and passed.
- Local `singbox.test.js` and `softnav.test.js` passed; `git diff --check` passed with existing LF/CRLF warnings only.
- Disposable package/lifecycle smoke passed **30/30** checks against the exact amd64 candidate SHA-256 `06a71ab045b94b17f31c03e95bf3ac9e7c373c747e875c32ad07a5d4309b5aec` in an Ubuntu 24.04.5 container. It covered fresh authenticated startup/API/SQLite, embedded Xray, native-core and XDRIVE/FinalMask/transport config checks, uninstall cleanup, preservation of unowned fixtures, private-container firewall/route cleanup, and port release. The test network was disconnected before execution; no host mounts were used. This is container-userland coverage, not systemd boot or kernel qualification.
- `test_unit/install-sh-check.py` passed all **8** contract checks from a temporary LF-normalized copy of the Windows-staged CRLF scripts. Curl was stubbed; no installer network action occurred.
- Browser smoke against the same candidate passed Persian RTL login, Dashboard → Inbounds → Nodes → Inbounds navigation, and the Add Inbound modal with no console errors. A loopback-only VLESS sing-box inbound and its default user were created in the disposable container. The user appeared in the managed Clients page and its account edit modal opened.
- The UI issue recorded by this smoke was resolved in the final 2026-11-27 continuation: `StreamLimit` rendered for a selected sing-box membership, policy wording directed edits to Clients, and the Clients link no longer doubled the panel path.
- A 384×838 CSS-pixel iframe preview of the Persian RTL Clients page had no horizontal overflow. This was a viewport simulation, not a physical mobile device/emulation, and the account modal was not tested at that size.
- Cleanup removed only the named test container, its private Docker network, the temporary test inbound with the container, the SSH tunnel and tunnel logs. The four earlier review/audit containers remained running. Post-cleanup, the live port 12301 listener still belonged to PID 724485.
- No product code or production/live-database/service, commit, push, or release action. The only database changes were confined to the disposable test container, which was removed. Live listener on port 12301 untouched.
- ARM64 packaging remains blocked by missing cross-compiler/QEMU. No systemd boot, kernel/distro matrix, or real-device mobile test was performed.

- User asked whether sing-box now has no remaining problems. Answer: no blanket guarantee. The local, uncommitted candidate passed the recorded source and runtime tests for exercised paths, but the published/installed 6.1.0 binary was not changed by this remediation. Known limits include panel-managed ShadowTLS multi-inbound chaining; speed and IP limits are now enforced via the shared sidecar, while physical-device counting remains unsupported (rejected).
- Historical: user considered browser/package follow-up and asked if it was necessary. The initial recommendation was to defer it; the user later requested the scoped smoke, which is now recorded above. A broad distro matrix is needed only for a broad distro-support claim; ARM64 build only if ARM64 is a release target.
- Earlier read-only preflight on `sweden` confirmed Docker CLI 29.1.3, an active Docker service, and 8.8 GiB free disk; it predates the final rebuilt candidate and is superseded for lifecycle/browser smoke by the actual results above.
- No product-code or production/live-database/service change, release, commit, or push was made. The temporary database/container were discarded. Port 12301 remains untouched; preserve the dirty worktree and unrelated changes.
- Historical next step after the UI fix: preserve the dirty candidate and decide separately whether broader qualification is required. The 13-distro container userland matrix is now complete; mobile/real-device, systemd/kernel, and ARM64 qualification remain separate follow-ups. Do not deploy or release without separate approval.

### Final continuation — amd64 rebuild and browser verification (2026-11-27)

- The local sing-box form/test files were synchronized to `/home/ubuntu/vpn-aio-managed-accounts-20261125`; local `singbox.test.js` and `softnav.test.js` both passed. The remote source copy has no Node runtime, so those frontend tests were not run remotely.
- Remote `CGO_ENABLED=1 go test -count=1 ./web -run '^TestTranslationKeyParity$'` passed. `bash ./build.sh --skip-submodules --geo-lean` completed with Go 1.27.4.
- Final exact candidate: `/home/ubuntu/vpn-aio-managed-accounts-20261125/build/out/VPN-AIO-amd64`, 271,609,888 bytes, SHA-256 `a079aeba5b2db399814fe5512cc83be8c4ee5dd066f4022104c791aa69601401`; embedded `vpn-aio-limits` marker confirmed. This remains an isolated, uncommitted verification candidate, not a release artifact.
- The candidate was copied into disposable `vpn-aio-ui-smoke-20261127`. Browser verification confirmed the sing-box Clients anchor resolved to `/panel/clients`, created a loopback VLESS sing-box inbound on port 34411, rendered the managed account `StreamLimit` field, accepted and posted `2`, and then deleted the temporary inbound. Final API lists contained zero inbounds and zero clients.
- Cleanup removed only the disposable UI container/network, SSH tunnel, and browser tab. The four pre-existing review/audit containers remained running. Port 12301 remained owned by PID 724485 and was untouched.
- The final browser console retained two HTTP 500 font loads for the referenced Vazirmatn/Nunito files and an Ant Design notification `TypeError` after the successful account save. These did not prevent the tested workflow or cleanup, but this run is not a zero-console-error result; no fix was made for those separate observations.

### Continuation — broader container matrix launch (2026-11-27)

- The first 13-distro `test_unit/container-matrix.py` attempt used the exact candidate SHA-256 `a079aeba5b2db399814fe5512cc83be8c4ee5dd066f4022104c791aa69601401` but failed at every image-pull stage because the harness invoked Docker without the required socket privileges. The result directory was `/home/ubuntu/vpn-aio-managed-accounts-20261125/test_unit/results/containers-20261126-213448`; no test container was started or left behind.
- The privileged rerun exposed a stale harness expectation: `container-lifecycle.py` defaults to `3.1.0`, while the exact candidate reports `6.1.0`. `test_unit/container-matrix.py` was corrected to accept `--expected-version` and default to the checkout's `config/version`, then pass `VPN_AIO_EXPECTED_VERSION` into each container. The patched script was synchronized to `sweden`; local compile-only validation and `git diff --check` passed.
- A one-distro validation passed 30/30 on `ubuntu:24.04`. The complete corrected rerun passed **13/13 distros**, each with **30/30** lifecycle checks: Ubuntu 24/26, Debian 12/13, Fedora 43/44, AlmaLinux 9/10, Rocky 9/10, CentOS Stream 9/10, and Arch. Results are at `/home/ubuntu/vpn-aio-managed-accounts-20261125/test_unit/results/containers-20261126-214514`.
- The result recorded the exact candidate SHA-256 `a079aeba5b2db399814fe5512cc83be8c4ee5dd066f4022104c791aa69601401` and size 271,609,888 bytes. Matrix containers were removed after each distro; the four pre-existing review/audit containers remained. Port 12301 remained owned by PID 724485 and untouched.
- The installer contract passed all 8 checks from temporary LF-normalized copies on `sweden`. A direct Windows invocation failed before the checks because WSL reported no installed Linux distribution; no installer/source behavior was changed.

### Continuation — exact candidate live UI harness rerun (2026-11-27)

- The exact final candidate `build/out/VPN-AIO-amd64` (271,609,888 bytes, SHA-256 `a079aeba5b2db399814fe5512cc83be8c4ee5dd066f4022104c791aa69601401`) was run in a fresh disposable container and reached only through a private SSH tunnel.
- The first live harness attempt produced four false failures because `test_unit/live/uitest.js` selected the first `.ant-btn` in an account row. The current row places Account Info before the overflow menu; the helper now targets `.ant-dropdown-trigger`.
- The corrected harness run passed **60/60** checks covering rendering, account add/edit/delete, multi-inbound membership changes, bulk operations, credentials, exports, protocol identity, Xray-native WireGuard assignment/configs, and Inbounds-page client isolation.
- The corrected run recorded no console errors. Direct requests to the two referenced font files returned HTTP 200 with `font/woff2` content. The earlier browser-session font HTTP 500s and Ant Design notification TypeError were not reproduced; no product-code fix was made for those historical observations.
- Cleanup removed the disposable panel container, SSH tunnel and local headless browser. The four pre-existing review/audit containers remained. Port 12301 remained owned by PID 724485 and untouched.
- This validates the live harness path only. Physical mobile, external mobile VPN clients, systemd/kernel, ARM64 packaging and panel-managed ShadowTLS multi-inbound chaining remain outside this result.

## Historical status before the independent review

- **Released VPN-AIO 6.1.0 (2026-11-25):** first lean release — both release assets are built with `GEO_LEAN=1` (~118MB smaller than 6.0.0: ~417MB amd64 / ~411MB arm64 instead of 535MB/529MB). Also carries the CI changelog fixes (`0d41f08` drop duplicate Full Changelog, `56874be` fetch tag before changelog, `75c65bb` tag-fetch failure now fails the step).
- **Lean release builds (2026-11-25):** the release workflow sets `GEO_LEAN=1` for both architecture builds

- **Released VPN-AIO 6.0.0 (2026-11-25):** tag `v6.0.0` pushed to `robin808far/VPN-AIO`; `vpn-aio-release.yml` builds both amd64 and arm64 bundles and publishes the release as **VPN-AIO 6.0.0** with an auto-generated changelog body (features/fixed/CI from commit messages). 6.0.0 highlights: soft navigation (shell persists, only the content area reloads), sing-box inbound core, Reality target scan, RTL/Persian inbound-form polish, protocol list on login, cached CI bundles.
- **Lean release builds (2026-11-25):** the release workflow sets `GEO_LEAN=1` for both architecture builds — only the base geo pair (geoip.dat + geosite.dat, ~28MB) is embedded; the four Iran/Russia country files (~118MB) are no longer in the binary. The panel downloads a country file the first time a routing rule names it (`EnsureGeofilesAuto` in `web/service/geofile.go`, wired at `web/service/xray.go:808` and on settings save at `web/service/xray_setting.go:41`), and the Geofiles page fetches one by hand. Fresh installs and updates are unaffected; expect each release asset ~118MB smaller.
- **Soft navigation implemented and verified (2026-11-24/25):** `web/assets/js/softnav.js` intercepts panel-link clicks, fetches the destination, swaps only the body and re-runs the page's scripts; the side rail and top bar are frozen as pixel-identical fixed clones (rebuilt after the body swap in the same tick) plus a content-area loader, so the shell never flashes and only the middle reloads. Key pitfalls (do not regress): run inline scripts via indirect `eval` inside try/catch (`appendChild` SyntaxErrors fire async and silently abort the chain); `declVar` rewrites top-level const/let (indent ≤ 4) to `var`; ALL top-level shell consts must be `var` in the HTML sources (`page.html` RESELLER/PERMS/basePath, `aSidebar.html` SIDEBAR_COLLAPSED_KEY, `prompt_modal.html`, `text_modal.html`, `aPersianDatepicker.html`); page Vue apps must re-run on EVERY navigation; never skip "already-run" scripts by hash (shared byte-identical mixins like ClientActionsMixin would be lost).
- **sing-box inbound live e2e verified (2026-11-24):** the panel binary (`/tmp/vpn-ui`, job `panel-preview`, DB `/tmp/xui-run/VPN-AIO.db`) was rebuilt with the `GetXrayConfig` core-fix and exercised end-to-end:
  - A `vless` inbound with `core="singbox"` was created via `POST /panel/api/inbounds/add`. Key semantics: a singbox inbound's `settings` must be a **raw sing-box inbound object** (`{"type":"vless","listen":"127.0.0.1","listen_port":4431,"users":[{"uuid":…}]}`), not the Xray-style `clients/decryption` blob; Xray-style settings produce a config with empty `type` and sing-box refuses it.
  - The singbox core shows `not_installed` on a fresh host until `POST /panel/core/provision {"cores":["singbox"]}` completes (installs `/tmp/bin/sing-box`). After `POST /panel/core/restart/singbox`, the config renders to `/etc/vpn-ui-singbox/server-<id>/config.json` and sing-box runs, listening on the inbound port.
  - Live connection test: sing-box client (vless outbound → panel inbound 4431, mixed inbound on 11080) proxied an HTTP request to a local server successfully (`hello-test-body` returned through `socks5h://127.0.0.1:11080`). Test inbound deleted afterwards; panel left running on port 3000.
  - `go build` + `go vet` clean; `go test ./web/service/ ./xray/ ./web/ -count=1` all ok on 2026-11-24.
- **RTL/Persian inbound-form fixes (2026-11-24, uncommitted):** the running panel (`/tmp/vpn-ui`, job `panel-preview`, DB `/tmp/xui-run`, webPort 3000 in the settings table — the public preview link only exposes port 3000) was rebuilt with these changes, all verified in the browser at 1280px in fa-IR:
  - `web/assets/css/components.css` RTL block (bottom of file): modal form items get vertical label-above-input rows with spacing (fixes Persian labels glued to inputs), the modal close X moves to the left, `ant-btn + ant-btn` gets 8px separation (footer buttons and toolbar pills were touching), `.bo-rtl-reverse` flips the none/TLS/Reality radio pills into Persian reading order, and `.ant-modal-wrap` z-index 1000 keeps modals above the security-alert banner.
  - `web/html/form/tls_settings.html`: the security radio group carries `:class="{ 'bo-rtl-reverse': … }"` for fa/ar.
  - Cause note: an earlier RTL fix pass was invisible because the running binary predated it — **always rebuild+restart `/tmp/vpn-ui` after touching embedded CSS/HTML**.
  - Port 3000 was re-captured twice by the managed static preview server (`preview.py serve --root redesign/`); if the panel "disappears" again, kill that python3 process and restart the `panel-preview` job.
- Full `go test ./...` green (no FAIL) on 2026-11-24 after these edits; `go build` + `go vet` clean.
- **Version is now `4.0.0`** (`config/version`), carrying the new multi-panel Nodes feature (below). Released as https://github.com/robin808far/VPN-AIO/releases/tag/v4.0.0 (Latest). Releases now extend to v5.0.0 (run 45240381545, amd64+arm64+SHA256SUMS, 2026-11-23).
- **Version-pinned install (2026-11-23):** new `install.sh` at the repo root, 3x-ui style: `bash <(curl -Ls https://raw.githubusercontent.com/robin808far/VPN-AIO/main/install.sh) v3.1.0` (optional tag, `v` optional, validated; bare run = latest). It downloads `deploy.sh` from `main` — not from the pinned tag, because tags older than v3.1.0 ship a deploy.sh without `VPN_AIO_VERSION` support — fetches it to a mktemp file before the root check (works under `bash <(curl …)` where `$0` is `bash`), and re-execs it with sudo + `VPN_AIO_VERSION` when not root. All 7 READMEs now show this one-liner (with a versioned example) instead of the old `deploy.sh | sudo bash` pipe.
- **`media/overview.png` replaced (2026-11-23):** now the panel's front (login) page, captured from a live v4.0.0 build at 1395x900 via the built-in browser; pushed to GitHub.
- The GitHub recovery for `robin808far/VPN-AIO` is complete, including its three public releases.
- Happ 3.1.0 is committed/pushed and publicly released: https://github.com/robin808far/VPN-AIO/releases/tag/v3.1.0.

## Nodes (multi-panel) — implemented in 4.0.0

Master–worker like 3X-UI's Nodes tab: this panel (master) registers other VPN-AIO panels (nodes) and manages/probes them; the node side exposes a bearer-token API for the master.

- Data model (`database/model/node.go`): `ApiToken` (SHA-256 hashed bearer token, scopes `node-sync`/`node-admin`, plaintext shown once) and `Node` (name, scheme http/https, address, port, base path, token of the remote, sync mode `all|selected`, enabled, status/version/inbounds/lastSeen). Both auto-migrated in `database/db.go`.
- Node-side API (`web/controller/node.go`, under `<base_path>/nodeapi/*`, Bearer auth with scope check): `GET status` (version, Xray running, inbound/client/traffic counts), `GET inbounds`, `GET traffic`.
- Master-side API (`/panel/nodes/*`, session + super admin): list/add/update/del, `probe/:id`, `inbounds/:id` (remote inbounds), plus token management `tokens`/`tokens/del/:id`.
- Services (`web/service/node_service.go`, `apitoken_service.go`): probe updates status/version/lastSeen (unreachable → `down`), heartbeat over all enabled nodes; add/update probe immediately; empty token on update keeps the stored secret.
- Heartbeat job (`web/job/node_heartbeat_job.go`) scheduled `@every 30s` in `web/web.go`.
- Frontend: new `web/html/nodes.html` (node cards with status tag, probe/edit/delete, remote-inbound table, API-token modal with one-time plaintext alert), sidebar entry (super admin only) in `web/html/component/aSidebar.html`, `pages.nodes.*` i18n keys added to ALL translate.*.toml locales (enforced by `TestTranslationKeyParity`).
- Tests: `web/service/node_service_test.go` (token create/verify/revoke/scope, URL+normalize defaults, add-probes-unreachable → down). All pass.
- Not yet implemented (next steps): inbound/client management on remote nodes and traffic sync pull (map of `node-admin` scope), mTLS/pinned-TLS node security options, sub-node trees.

## Completed work

- Happ 3.1.0 is committed/pushed and publicly released as Latest: https://github.com/robin808far/VPN-AIO/releases/tag/v3.1.0.
- The working copy is on `main` and tracks `origin/main`; this final documentation-only checkpoint is committed/pushed after publication.
- Feature request: add native Happ subscription management and set VPN-AIO version to 3.1.0.

## Completed work

- Confirmed releases `v2.0.0`, `v2.1.0`, and `v3.0.0` already existed on GitHub; did not recreate them.
- Verified all three releases are public (not drafts or prereleases) and their release pages return HTTP 200.
- Compared release asset counts, sizes, and SHA-256 digests against the local recovery backup; all matched.
- Recovery initially confirmed `v3.0.0` as Latest; it remains available and has now been superseded by `v3.1.0`.
- Implemented the 3.1.0 Happ subscription settings/UI and client-specific response headers; release files are drafted locally and not published.

## Release links

- `v2.0.0` — 2 assets: https://github.com/robin808far/VPN-AIO/releases/tag/v2.0.0
- `v2.1.0` — 4 assets: https://github.com/robin808far/VPN-AIO/releases/tag/v2.1.0
- `v3.0.0` — 4 assets: https://github.com/robin808far/VPN-AIO/releases/tag/v3.0.0
- `v3.1.0` — 4 verified assets, Latest: https://github.com/robin808far/VPN-AIO/releases/tag/v3.1.0

## Checks and caveats

- Implementation and release documentation are committed/pushed; final handoff records completed publication (2026-11-22).
- Release visibility, release metadata, asset counts/sizes/digests, and Latest selection passed verification.
- No sensitive credentials or tokens are recorded here.

## Documentation layout review

- Reorganized 34 root-level project documents into `docs/architecture/`, `docs/guides/`, `docs/history/`, `docs/plans/`, `docs/reference/`, and `docs/releases/` (including validation records). No documentation was deleted.
- Added `docs/README.md` as an index. Kept the translated README files and `HANDOFF.md` in the root.
- Updated affected links in English/Persian READMEs, release notes, API reference, validation records, plan notes, and code comments.
- Verification: local Markdown-link check passed; `git diff --check` passed. No test suite run (changes are documentation/path references and comments only).
- Documentation reorganization and the initial `HANDOFF.md` were pushed to GitHub in `6f8ad6d2`; later Happ work remains local.
- The version file reads `3.1.0`; GitHub Latest is now `v3.1.0`. Older recovered releases remain available.

## Happ subscription integration review

- Compared VPN-AIO with the local 3x-ui reference (`5297fd3`, read-only) and Happ's official app-management, Provider ID, and routing documentation.
- VPN-AIO already has a Happ routing toggle/rules field and Happ deep links in the subscription page. Its `SUBController.ApplyCommonHeaders` currently sends routing headers without checking client User-Agent; the server is configured at startup.
- The reference's native Happ integration adds User-Agent detection/opt-in and application settings for subscription banners/expiry, provider ID/failover, tunnel mode/type, route exclusions, ping, auto-connect, per-app proxy, local proxy auth, and related controls. Reference UI groups these into Routing, Appearance, Network, and Failover; VPN-AIO should implement equivalent controls in its existing Vue settings UI rather than copying the reference React frontend.
- Happ's official docs: https://www.happ.su/main/dev-docs/app-management, https://www.happ.su/main/dev-docs/provider-id, https://www.happ.su/main/dev-docs/routing. Advanced app controls require Provider ID; delivery uses documented HTTP response headers and some settings can also be subscription-body comments.
- Implementation notes: preserve current routing-header behavior unless compatibility is explicitly tested; gate newly added Happ-specific headers behind opt-in and Happ User-Agent detection. Add values to `entity.AllSetting`, the settings default map, Vue settings model/form, and subscription response config. Existing key/value settings persistence suggests no schema migration, but add defaults for old installations. Subscription server config is initialized on startup; account for restart/apply UX.
- Added settings for Happ opt-in detection, routing presets, banner/expiry notices, Provider ID, replacement/fallback URLs, TUN, excluded routes/APNs, ping, auto-connect, per-app proxy, local proxy auth, hide-settings and no-limit mode.
- The subscription server emits Happ configuration headers only when auto-detection is enabled and the User-Agent identifies Happ. Existing Routing headers remain unchanged for other clients. Header values strip CR/LF; settings validate enums and text lengths.
- Settings use the existing key/value store and defaults; no database migration is needed. The subscription server reads settings at startup, so save and restart the panel for changes to take effect.
- Verification: `go test -count=1 ./web/entity` passed; `go test -count=1 sub/happ.go sub/happ_header_test.go` executed successfully on Windows. All six Node scripts in `test_unit/export_test/*.test.js` passed, including the actual settings-page preset method.
- Linux/amd64 test binaries for `./sub`, `./web`, and `./web/service` compiled with CGO disabled. This is compile-only verification, not execution or a production build with SQLite enabled.
- All HTML template directories parsed and the Subscription/Happ Go templates rendered successfully; all 13 TOML files parsed; 69 Happ template labels resolved in English/Persian; local Markdown links and `git diff --check` passed. Other languages use the existing English fallback for new Happ labels.
- Follow-up regressions reproduced and fixed: uppercase/whitespace enum values were forwarded unnormalized; inactive per-app lists were sent; routing presets did not synchronize the routing switch. Added URL and theme validation (absolute HTTP(S), no embedded credentials, theme JSON object/resetcolors).
- Added a database reopen/persistence test under `web/service/happ_settings_test.go`; it is compiled but not executed here. Added `docs/guides/HAPP.md` with activation/restart, platform, cached-setting, literal URL, Provider ID/privacy, and client-versus-server enforcement notes.
- Linux verification on the user-authorized SSH host `sweden`: all tests in `./sub`, `./web`, and `./web/service` passed with CGO/SQLite enabled and no test-name filters. Tests ran in private mount/network namespaces with loopback and a copied public-DNS fixture; Xray used the verified bundled geodata directory.
- The first Linux runs found intentionally deferred Happ translation keys and a test-only geodata path issue. Recorded the 64 exact Happ keys in the existing `knownMissing` translation baseline (English/Persian shipped; other languages use English fallback), then corrected the runner's `XRAY_LOCATION_ASSET`. Re-ran entire packages successfully.
- Candidate built on `sweden` in `/home/ubuntu/vpn-aio-happ-3.1.0-20261122` using Go 1.27.4, GCC 13.3, and `bash build.sh --quick --skip-bundle --no-color`. Cached Xray/geodata/native-daemon inputs were verified against the previous release's SHA-256 inventory. Windows shell line endings were normalized only in the staged source copy.
- Candidate `build/out/VPN-AIO-amd64`: 354791696 bytes; SHA-256 `26c0eeb1928828f5378fe90bcd074e7c074a127a69512ea56bfd2affd30d7213`. It is an uncommitted-source verification candidate, not a published release artifact.
- `test_unit/happ-subscription-check.py` passed 53 HTTP checks against the exact candidate in a network-disabled disposable Docker container: version, fresh opt-in default, authenticated setting save/reload, restart application, raw/JSON/Clash Happ headers, unchanged quota/payloads, suppression for other clients, rendered settings page, and opt-out after restart. This simulates Happ User-Agents, not an actual Happ application.
- Removed only the named test container `vpn-aio-happ-check-20261122` after saving its logs. Linux test/build/HTTP logs and the candidate remain under `/home/ubuntu/vpn-aio-happ-3.1.0-20261122/build/out/` on `sweden`.
- Live host panel/binary/database have not been replaced or upgraded.
- Real-device validation: user authorized adding a separate test subscription to the already-open Windows Happ 4.8.3 (640). The app imported `VPN-AIO 3.1.0 TEST`, displayed its 1 GiB quota, 2026-11-29 expiry, 24-hour update interval, and the panel-generated information banner. Updating the disposable fixture banner and clicking that subscription's Refresh displayed the new text in Happ.
- Scope: real app requests were logged with User-Agent `Happ/4.8.3/Windows/2611171045601`. A private test relay allowed only subscription metadata/banner headers and stripped routing, TUN, auto-connect, proxy-auth, Provider ID and all other global controls to preserve the user's app settings. Global overrides, connections, Android/iOS behavior and Provider-ID-dependent functionality were not exercised.
- Cleanup: removed only the device-test Docker container/network and stopped its dedicated SSH tunnel. Logs remain in `build/out/happ-device-check.log` and `happ-device-panel.log` on `sweden`; window captures remain in the approved local temp directory. The imported test profile remains in Happ for the user to inspect; its temporary URL is now offline and it is not a real VPN service.
- Feature commit `272ab016f8a30fe8b590ecf18560df6525275faf` and documentation commit `07d75f88f02f1e4005865c77a2ffd454c20ca209` have been pushed. Annotated tag `v3.1.0` points to the documentation commit; its diff from the build source contains only three Markdown files.

## Historical next step (v3.1.0 era; superseded by the v7.0.0 status above)

Release finalization is complete. Await the next user-requested change or separately authorized server upgrade. Do not upgrade the live `sweden` panel or alter global Happ options without approval. Real Windows Happ import, quota/expiry/banner display and refresh are verified; broader global/platform-specific controls remain untested. The retained test profile's temporary URL is offline. Update this file after each significant milestone.

## Release finalization

- User authorization: commit + push + public v3.1.0 release, without live-server deployment.
- Extended Ubuntu CI to run the Happ preset test and isolated HTTP check. Lifecycle validation receives the version from `config/version`; standalone default is now 3.1.0.
- Local entity/header tests and all six Node regression scripts passed again before commit.
- Feature commit: `272ab016f8a30fe8b590ecf18560df6525275faf`; GitHub Ubuntu CI passed: https://github.com/robin808far/VPN-AIO/actions/runs/45009369333.
- Clean-source build completed in `/home/ubuntu/vpn-aio-release-272ab016` on `sweden`; pinned submodules and cached embedded assets verified, complete Linux Go suite passed, and the full SQLite-enabled binary built.
- Final release asset: `VPN-AIO-amd64`, 354619744 bytes, SHA-256 `907a09d473a74c76e58488d9b9dd9de2524556bbb27ab48850974c2ef0b5736d`. This replaces the earlier uncommitted candidate for publication.
- Exact asset passed 53/53 isolated Happ HTTP checks and 30/30 disposable Ubuntu startup/native-protocol/uninstall lifecycle checks. All test containers were removed after saving logs.
- Browser login, Subscription/Happ navigation, and its enable switch worked on the exact asset's disposable panel. Desktop browser disconnected before completing the tab interactions. The UI container/network/tunnel were removed; the browser test tab may still point to the now-offline fixture.
- Four assets retrieved and SHA-256/size verified locally in `C:\Users\mef\AppData\Local\Temp\opencode\vpn-aio-release-3.1.0-272ab016`: binary, manifest, checksums, embedded-input inventory. No secrets included.
- Uploaded all four assets as a draft and compared every uploaded name, size, SHA-256 digest and state with the verified local files; all matched.
- Published `v3.1.0` as a non-draft, non-prerelease Latest release. Unauthenticated latest API and release page succeeded; all four download URLs returned data, and the binary download begins with the ELF signature.
- Final outcome: https://github.com/robin808far/VPN-AIO/releases/tag/v3.1.0. No live-server deployment or global Happ setting change was performed.

## Nodes (multi-panel) research — 2026-11-23

Request: add a 3x-ui-style "Nodes" tab to VPN-AIO so one panel can manage other VPN-AIO panels (inbounds, clients, traffic) over the API. This section records the research; nothing was implemented yet.

### How 3x-ui Nodes works (upstream, available since v3.0.2)

- Master–Worker star topology: the master panel keeps users/subscriptions/settings and commands worker panels through a node API (`/panel/api/nodes/*`); client online state and traffic stats are synced back periodically.
- A node is another 3x-ui panel registered on the master with: unique name, scheme (http/https), address, port, base path, Bearer API token minted on the node, TLS verify mode (`verify` default, `skip`, `pin` with SHA-256 fingerprint, or `mtls`), inbound sync mode (`all` or `selected` by tag), and an optional `outbound_tag` bridging traffic to the node through a named outbound.
- The master probes reachability (`ensureReachable`) on create/update before persisting a node, then heartbeats every few seconds, keeping `online`/`offline` status and publishing `node.up`/`node.down` events (consumed by the Telegram/Discord bots).
- Nodes are identified by a stable per-panel GUID across restarts; a node can itself manage others (shown as read-only transitive sub-nodes); deleting a node does not auto-migrate its inbounds.
- Master-side UI: node list with online status, CPU/RAM/uptime, Xray version, per-node traffic this month; add/edit/pause/probe/delete; per-node metric history; remote panel self-updater trigger; mTLS CA exchange (`nodes/mtls/ca` + trusted-CA setting on the node). mTLS-only nodes need a `node-admin` scoped token for remote panel updates.
- Upstream code map: `internal/web/controller/node.go`, `internal/web/service/node*.go` (contract, mtls, tree, reset queue), `internal/web/job/node_heartbeat_job.go`, `node_traffic_sync_job.go`, `internal/crypto/nodetoken`, `internal/database/model/node_client_ip.go` / `node_client_traffic.go` / `node_pending_reset.go`, plus a React `NodesPage` frontend and docs at `docs/content/docs/{en,fa}/operations/multi-node.mdx` and `.../reference/api/nodes.mdx`.

### What VPN-AIO needs before this can be built

- API tokens: VPN-AIO's panel API is session-cookie auth only (docs/reference/api-reference.md section 1); 3x-ui added Bearer API tokens (Settings → Security → API Token) in v3.3.0 and Nodes depends on them. This is the first prerequisite.
- Node-side node-API endpoints on every VPN-AIO instance (accept master commands), plus master-side registration/probe/heartbeat/sync jobs in `web/job/`, a nodes controller/service in `web/controller/` and `web/service/`, and a Nodes tab in the Vue UI (`web/html`, `web/assets`).
- Mapping to this fork: "inbound sync" must cover the 19 protocols and the accounts/membership layer, not only Xray inbounds; `xray_traffic_job.go` is the pattern for the traffic-sync job; native protocols (Libreswan/StrongSwan/pppd/SSTP backends) need a decision on how per-node management maps onto them.
- Recommended phases: (1) API tokens with scopes (`node-sync`, `node-admin`); (2) node-side API; (3) master-side Nodes tab (list/add/probe/heartbeat/status); (4) inbound/client management on remote nodes + traffic sync; (5) mTLS and sub-node tree (optional parity extras).

Sources: https://github.com/MHSanaei/3x-ui (tree + raw files, fetched 2026-11-23), https://www.zoio.net/2026/08/3x-ui-multi-node.html, https://library.tf/providers/batonogov/threexui/latest/docs/resources/node. Upstream reference commit locally: `5297fd3` (read-only) predates the Nodes feature; upstream `main` was used instead.

## 2026-11-27 release v6.5.0
- Picker fix 359ee7f4 pushed to main. Candidate 6bdbeb42 passed installer contract 8/8, live UI harness 60/60, disposable detour runtime check 18/18 (select/clear/reselect/delete-target/self-detour). config/version bumped to 6.5.0 and tagged v6.5.0; CI release workflow builds amd64+arm64. VM/systemd/kernel and physical mobile checks were not run. Live listener 12301 untouched.

- Historical checkpoint 15:05 +0330: release run 45991545057 and validation run 45991539428 were in_progress. Both later completed successfully; public Latest assets, hashes, versions and embedded patched cores are now verified in Current status above. No production deployment was made.
