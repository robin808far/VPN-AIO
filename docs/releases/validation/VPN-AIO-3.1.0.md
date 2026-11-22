# VPN-AIO 3.1.0 validation record

## Release artifact

- Source commit: `272ab016f8a30fe8b590ecf18560df6525275faf`, clean checkout.
- Build host: authorized `sweden`, Ubuntu 24.04 amd64.
- Toolchain: Go 1.27.4, GCC 13.3.
- Command: `GOMAXPROCS=2 GOFLAGS=-p=1 bash build.sh --quick --skip-bundle --no-color` (CGO/SQLite enabled).
- Asset: `VPN-AIO-amd64`, 354619744 bytes.
- SHA-256: `907a09d473a74c76e58488d9b9dd9de2524556bbb27ab48850974c2ef0b5736d`.
- Xray pin: `e0a162b8eb10634e297afecedfc569308c490014`; telemt pin: `1e6549df23136a676ef8ec36ef0f1d9af2a1083d`.
- Complete cached Xray/geodata/native-daemon inputs verified against the previous release's SHA-256 inventory. No upstream-tip update.
- The release includes `BUILD-MANIFEST.json`, `SHA256SUMS`, and `embedded-inputs.SHA256SUMS`. Later documentation-only commits may refer to this exact tested source.

## Checks performed

- Complete Linux `CGO_ENABLED=1 go test -count=1 -timeout=10m ./...`: passed, without test-name exclusions. Private mount/network namespaces prevented conflicts with the live host; the public-DNS fixture and verified bundled geodata were supplied.
- GitHub Ubuntu CI passed for the source commit: https://github.com/robin808far/VPN-AIO/actions/runs/45009369333.
- Exact release asset: **53/53** isolated HTTP checks passed for authenticated settings save/reload, restart application, raw/JSON/Clash metadata, quota/config preservation, non-Happ suppression, rendered settings HTML, and opt-out.
- Exact release asset: **30/30** disposable Ubuntu lifecycle checks passed for fresh login, embedded Xray/native protocol/config smoke, SQLite creation, and isolated uninstall. Both test containers were removed afterward.
- Local entity/header tests and all six Node regression scripts passed. All HTML templates parsed/rendered and all 13 translation TOML files parsed.
- Happ labels ship in English/Persian. The exact 64 deferred keys are recorded in the existing translation baseline; other languages use English fallback.

## Real Happ and interactive scope

- Windows Happ 4.8.3 (640) imported/refreshed a separate pre-commit test candidate and displayed its title, 1 GiB quota, expiry, update interval, and changing information banner.
- Its test relay stripped global overrides. No user-app TUN, routing, auto-connect, proxy-auth, Provider ID, or connection changes were exercised.
- On the exact release asset, browser login, navigation to Subscription/Happ, and the Happ enable switch were exercised in a disposable panel. Desktop browser disconnection prevented completing all interactive tab checks.
- Android/iOS, Provider-ID-dependent behavior, and global client overrides remain unverified. No fresh cross-distribution VM/kernel/DKMS matrix or live-host upgrade/rollback is claimed.
- The live `sweden` panel was not replaced. Logs remain under `/home/ubuntu/vpn-aio-release-272ab016/build/out/`; no credentials or private configuration are included in release assets.
