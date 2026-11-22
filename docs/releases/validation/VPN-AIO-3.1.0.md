# VPN-AIO 3.1.0 validation record

## Scope

- Native Happ subscription settings and Happ-only response headers.
- Version metadata and operator-facing release documentation.

## Checks

- `go test -count=1 ./web/entity` passed, including Happ text/enum validation, HTTP(S) URL checks, and theme validation.
- `go test -count=1 sub/happ.go sub/happ_header_test.go` executed successfully on Windows: configured headers, User-Agent/opt-in gating, off values, sanitation, enum normalization, and inactive per-app list omission.
- All six `test_unit/export_test/*.test.js` Node scripts passed, including the Happ presets using the actual Vue settings-page method.
- Linux/amd64 test binaries for `./sub`, `./web`, and `./web/service` compiled successfully from Windows with CGO disabled. These are compile-only checks, not production builds.
- All HTML template directories parsed successfully with Go `html/template`; the Subscription/Happ templates also rendered successfully.
- All 13 translation TOML files parsed; the English and Persian settings keys referenced by the Happ UI are present.
- Local Markdown links resolve and `git diff --check` passed.
- All tests in `./sub`, `./web`, and `./web/service` subsequently executed and passed on the authorized Linux host `sweden`, with CGO/SQLite enabled and no test-name filters, inside private mount/network namespaces.
- The translation parity test uses the project's explicit `knownMissing` baseline for 64 intentionally deferred Happ labels outside English/Persian; no test was skipped. The runner's bundled geodata path was fixed before rerunning the service suite.
- Full embedded candidate binary built using Go 1.27.4, GCC 13.3, and `bash build.sh --quick --skip-bundle --no-color`; cached embedded inputs verified with SHA-256. Staged shell source was normalized to Unix line endings.
- Candidate size: 354791696 bytes; SHA-256: `26c0eeb1928828f5378fe90bcd074e7c074a127a69512ea56bfd2affd30d7213`.
- The same candidate passed 53 HTTP checks in a network-disabled Ubuntu Docker container: authenticated settings save/reload, restart, Happ metadata on raw/JSON/Clash, quota/payload preservation, non-Happ suppression, embedded settings-page rendering, and opt-out after restart.
- HTTP coverage uses simulated Happ User-Agents, not the Happ application. The temporary container was removed after saving logs; no live-host panel upgrade occurred.
- Real Windows Happ 4.8.3 (640) imported a separate test subscription and displayed its title, 1 GiB quota, 2026-11-29 expiry, update interval and information banner. Refreshing that subscription displayed changed banner text.
- The real-device fixture stripped all global controls (routing, TUN, auto-connect, proxy-auth, Provider ID, etc.). Those controls were not exercised in the user's app; Android/iOS and Provider-ID-dependent behavior remain unverified. No test VPN connection was initiated.
- Device-test container/network/tunnel were removed afterward. Candidate is based on uncommitted local source and is not a published release; interactive panel-browser verification remains pending.

## Platform scope

- Target remains Ubuntu 24.04 amd64.
- No cross-distribution VM/kernel matrix is claimed by this record.
