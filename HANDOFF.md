# HANDOFF

Last updated: 2026-11-25

## Current status

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

## Next step

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
