# Project Progress & State

Last updated: 2026-11-27 (status checkpoint saved; panel-managed ShadowTLS chaining, full Linux tests/vet/build, and final isolated amd64 candidate verified; no commit/push/release/deploy)
Path: `C:\Users\mef\Documents\Default Project\VPN-AIO-review-6.1.0`

## Goal

Finish the local, uncommitted sing-box managed-account remediation in
`VPN-AIO-review-6.1.0`: raw sing-box `users` map to `Account`,
`AccountInbound`, and `ClientTraffic`; account edits project back into native
`users`; quota/expiry enforcement works. Speed and IP limits are now enforced
for sing-box through the shared `speedlimits.json` sidecar (consumed by a
patched bundled sing-box via `XRAY_SPEEDLIMIT_FILE`); physical-device counting
is not supported and is rejected. A separate `StreamLimit` field caps concurrent
routed streams/UDP associations. Then confirm and fix defects in the external
report `C:\Users\mef\Desktop\Bug.md` against this checkout, changing only
issues reproduced in current code. No production/live-database, commit, push, or
release action was made for this local remediation. Earlier audit-host
provisioning, performed separately, is documented in `HANDOFF.md`.

New active requirement (2026-11-27 continuation): implement panel-managed
ShadowTLS multi-inbound chaining. Sing-box runs one process/config per inbound
today; the chaining must be modeled as a detour graph (`SingboxDetourInboundID`
plus raw `detour`), resolved into connected process groups, with one root owner
per group for process/config/stats API port. Cycles, self-detours, missing
targets, and non-sing-box targets must be rejected. Deleted targets clear
detour references and old standalone processes are stopped before group startup.
Chained process user stats must use `InboundId = 0` to avoid misattributing
account traffic, while per-member inbound rows remain emitted.

## Current State

- Panel-managed ShadowTLS multi-inbound chaining is implemented in the local
  uncommitted tree. A chained ShadowTLS front plus its Shadowsocks detour target
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
- The earlier candidate hash `a079aeba5b2db399814fe5512cc83be8c4ee5dd066f4022104c791aa69601401`
  (271,609,888 bytes) is superseded by the rebuilt candidate above.
- Previously recorded state: four pre-existing review/audit containers and
  port 12301 owned by PID 724485. This continuation used private namespaces
  and did not access those containers or the listener.
- No commit, push, release, or production deployment has been made.

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

### Active
- Optionally browser-check the detour selector against the refreshed candidate;
  Node coverage now verifies target filtering and select/clear behavior, but no
  browser run has been done for this final candidate.
- Keep preserving the dirty tree; any release/deployment decision requires
  separate authorization and additional platform qualification.
- ARM64 packaging lacks `aarch64-linux-gnu-gcc`, QEMU, and binfmt support.
- Physical mobile/systemd/kernel qualification remains pending.

### Blocked
- ARM64 packaging lacks `aarch64-linux-gnu-gcc`, QEMU, and binfmt support.
- Physical mobile/systemd/kernel qualification remains pending.
- The live listener on port 12301 must remain untouched; network tests require
  private namespaces.

## Next Move
1. If continuing UI qualification, launch only the refreshed candidate in a
   disposable isolated panel and verify detour target selection, save, and
   group restart without touching the live listener.
2. For broader release readiness, use a runner with ARM64 cross-compiler/QEMU
   and separately qualify systemd/kernel and physical mobile behavior.
3. Do not commit, push, release, or deploy without explicit authorization.

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
- `web/service/singbox_protocol_runtime_test.go`: managed multi-inbound
  runtime test.

## Important Context
- Worktree remains dirty and uncommitted at
  `60951c62e3d5624cb1694868ac1387ae6d1d2144`; unrelated changes must be
  preserved.
- The earlier exact candidate
  `a079aeba5b2db399814fe5512cc83be8c4ee5dd066f4022104c791aa69601401` is
  superseded by the final candidate hash recorded above.
- The previously recorded state had four review/audit containers and port
  12301 owned by PID 724485. This continuation did not access either.
- No commit, push, release, or production deployment has been made.

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

- The underlying worktree is detached, dirty, and uncommitted at
  `60951c62e3d5624cb1694868ac1387ae6d1d2144`; preserve unrelated user changes.
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
  results apply to that earlier candidate; the new candidate has not yet had
  those disposable browser/lifecycle checks repeated.
- An intermediate chaining build (`650197fc...`) is superseded because its
  isolated copy lacked the two new UI templates. The final candidate is
  `cb476c47...` as recorded above; disposable browser/lifecycle checks remain
  pending for this final artifact.
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
- No commit, push, release, or production deployment has been made.

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
