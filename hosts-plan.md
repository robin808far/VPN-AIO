# VPN-AIO 3.0.0 — managed Hosts

## Scope

Implement upstream-style managed Host groups in the existing Vue panel. Preserve
2.1.0 configuration, account credentials, accounting, permissions and native
protocol exports. Adapted behavior references MHSanaei/3x-ui commit
5297fd3c8495d80fab97ae511553fa545c69e913 (GPL-3.0; retain project attribution).

## Work plan

- [x] Add the Host schema, repeat-safe legacy migration and lifecycle cleanup.
- [x] Add validated group CRUD, ordering, bulk operations and scoped APIs.
- [x] Render Host overrides consistently in raw, JSON and Clash subscriptions.
- [x] Support per-account Host selection with fail-closed deleted bindings.
- [x] Build the Hosts page and editor using the current Vue/Ant Design interface.
- [x] Keep panel links, QR codes, config downloads and backup/import consistent.
- [x] Add migration, permission, subscription and frontend regression coverage.
- [x] Validate on Linux in isolation and document results and coverage limits.

## Contracts

- Hostless inbounds keep their existing output and externalProxy behavior.
- Enabled applicable Hosts supersede the legacy endpoint list; address and port
  may inherit the inbound. Format exclusion and disabled hosts do not erase data.
- Each group may target several addresses and inbounds. Edits preserve row IDs
  wherever possible so account bindings survive.
- Only authorized admins manage Hosts. Shared groups cannot be changed through
  a partially authorized inbound. Resellers cannot edit shared endpoint policy.
- Host selection never changes credentials, runtime listeners, quota or traffic.
- Settings without a supported output consumer are not advertised as working.
- Preserve unsupported protocol/format exclusions, including MASQUE raw links.
- Published v2.0.0 and v2.1.0 tags, releases and assets remain untouched.
- The previously waived cross-distribution VM/kernel matrix remains unexecuted;
  it must never be reported as a passed check.

## Validation record

Implementation and the final validation run are complete. No 3.0.0 build or
release has been published.

The implementation also fixes compatibility failures found by the regressions:
fresh-database seeder ordering must not rehash bcrypt passwords, Host fanout must
not multiply legacy quota, IPv6 URI endpoints require brackets, group edits must
preserve exact endpoint IDs before matching replacements, imported bindings need
fresh IDs, and plaintext exports must remove Vision/TLS-only state.

Pinned-core validation determined the JSON TLS contract: certificate pins are a
comma-separated string, and the removed `allowInsecure` switch is restricted to
compatible RAW/Clash clients. Legacy VLESS settings without an explicit outbound
encryption value export `encryption: none`.

Browser checks cover create/edit/group enable state, unique link names and a
390px viewport layout. A raster screenshot could not be captured because the
desktop browser window was not visible; numeric DOM/layout checks are recorded
instead of claiming screenshot-based visual verification.

### Final results

- Full Linux `go test -count=1 -timeout=10m ./...`: passed without test-name
  exclusions, including the pinned-core Host JSON acceptance subtest.
- Go ran as the unprivileged test user in a private network/mount namespace to
  avoid the live SSH SOCKS listener on port 12301. `XRAY_LOCATION_ASSET` pointed
  at the bundled geodata; `example.com` used a captured public-DNS hosts fixture
  inside that namespace for the pre-existing URL-validation test.
- Node Hosts link/selection/CIDR/IPv6/SNI regressions: passed. Existing core GUI
  and routing round-trip scripts also passed.
- Fresh candidate startup/authentication, native Xray smoke/configuration,
  uninstall and namespace cleanup: **30/30**, Docker process exit **0**.
- Browser: create a two-address group, edit it without changing row IDs, toggle
  the group off/on, render distinct domain/IPv6 links, and load the account Host
  picker. Hosts, Inbounds and Clients pages/editor reported no JavaScript errors
  on the final candidate.
- Mobile layout: viewport/document width 390px; modal width 374px and right edge
  378px. Screenshot limitation is recorded above.
- The original `sweden` service remained active, Xray/AWG running, and its binary
  SHA-256 remained `e0865aa6d1884b5b95a484e6974d1ef35171d05fa5328a0fbcb4b256439e3695`.
- Existing v2.0.0/v2.1.0 tag identities were retained. No tag, release or asset
  publishing operation was performed.

The development test binary (not a published/static release artifact) is at
`/home/ubuntu/vpn-aio-hosts-v3-20261120/build/out/VPN-AIO-3.0.0-check` on `sweden`.
Its SHA-256 is
`5f38f08d0af197a742d27cf4aee3a3e791e13b21820c4ac5b269e2b9aeb89520`.
Logs are `build/hosts-v3-go-tests.log`, `build/hosts-v3-image.log` and
`build/hosts-v3-lifecycle.log` in the same isolated source directory.

These checks establish migration, authorization, exports, parser acceptance and
isolated userland behavior. They do not claim a live third-party CDN test, a new
kernel/VM matrix, a GitHub CodeQL run, or public-release readiness.
