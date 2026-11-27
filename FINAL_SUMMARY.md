# Session Summary: Sing-Box Managed-Account Remediation and Build Completion

## Date: 2026-11-26

## Objective
Finish the uncommitted sing-box managed-account remediation in `VPN-AIO-review-6.1.0`, verify the capped amd64 full bundle build, and triage defects from the external bug report `C:\Users\mef\Desktop\Bug.md` — without production, database, commit, push, or release changes.

## Requirements Fulfilled

### Build Completion (PID 640536)
- The capped amd64 full release build (`bash ./build.sh --skip-submodules --geo-lean`) completed successfully on the isolated copy at `/home/ubuntu/vpn-aio-managed-accounts-20261125`
- Artifact `build/out/VPN-AIO-amd64` verified: **271,554,248 bytes**, SHA-256: `407f2fd5067d5b55a7853ef0af1cef16467968844428ef3fa4882f485d586467`
- Embedded components:
  - **sing-box 1.12.14** with `with_v2ray_api` (tags: with_gvisor,with_quic,with_dhcp,with_wireguard,with_utls,with_acme,with_clash_api,with_tailscale,with_v2ray_api, revision f56d9ab94569d46a48bfc2bcd97eea566ad365c8)
  - **telemt 3.4.23** (MTProto proxy)
  - **geoip.dat** (16M) + **geosite.dat** (12M)
  - **Xray core** (32M, static stripped ELF)
  - OpenVPN, strongSwan, ocserv, and other VPN daemons in `backend/bin/amd64/` and `corebundle/core/amd64/`
- Build caps: 1 CPU, 1.5 GiB memory, 2.8 GiB memory+swap via Docker

### Bug Report Triage (BUG-01 through BUG-10)
- **BUG-06** — Fixed: `ImportDB` now runs `MigrateDB`, `AccountService.MigrationAccounts`, `ReconcileAllWireguardXrayKeys`, then restarts **both** cores through `restartImportedCoreServices`; `ImportForeignDB` uses same helper; import controller retries both cores on failure
- **BUG-07** — Fixed: `mergeBaseConfig` rejects base text containing `"inbounds"`; `SaveBaseConfig` validates proposed base against every enabled inbound before storing; `checkConfigText` creates config directory under `MkdirAll(singboxBaseDir())`
- **BUG-08** — Fixed: controller binds separate `nodeForm` (json + form tags) carrying `token`; empty token on add returns HTTP 400 (`ErrNodeTokenRequired`); `NodeService.Update` persists `Name`, rejects empty address, brackets bare IPv6 via `net.JoinHostPort`
- **BUG-09** — Fixed: `ApiTokenService.Create` normalizes scopes (trim, split on `,`, dedupe), rejects except `node-sync`/`node-admin` with `ErrInvalidTokenScope`; nil-safe `HasScope`; controller returns 400 for invalid scope/empty token name; form has `form` tags
- **BUG-10** — Fixed: `SettingController.updateSetting` loads stored settings first and binds submitted JSON/form on top, so omitted fields keep their values; now-unused `web/entity` import removed
- **BUG-03/04/05** — Already addressed in this checkout (not re-changed)
- **BUG-01/02** — Left unchanged: raw sing-box settings path does not use Xray defaults that produce those keys; no reproduced failure warrants change

### Go Suite Verification
- On `sweden` (isolated copy), `CGO_ENABLED=1 go test -count=1 -timeout=10m ./...` passed in private network namespace with namespace-local dnsmasq fixture and `GODEBUG=netdns=go`
- `go vet ./...` and `go build ./...` passed on the same revision
- Opt-in runtime tests passed: `TestSingboxAuditBundledTUICAndAnyTLSRuntime`, `TestSingboxAuditBundledShadowTLSV3StartsListener`
- New test `TestSingboxAuditBundledHysteria2ShadowsocksVMessAndTrojanRuntime` — compile-only verified (runtime execution against stamped core pending)

### Key Files Updated
- `PROGRESS.md` — Full build result, artifact details, and updated "Not yet tested" section
- `HANDOFF.md` — Build completion documented; managed-account remediation verification updated

### Preserved Constraints
- **No production, database, commit, push, or release changes** made
- Live host sing-box listener on port 12301 untouched throughout
- Remote test invocation uses private network namespace with loopback; `sudo -n unshare -n` used
- Unrelated dirty changes (79 git status entries) preserved; not committed or reverted
- `go test ./...` cannot build on Windows-native due to Linux-only symbols (`unix.RT_TABLE_MAIN`, `syscall.Ucred`, etc.); verification is Linux-based

### Not Yet Verified (Open Items)
- ShadowTLS authentication/chained traffic — only listener-bind smoke coverage exists
- Hysteria2/Shadowsocks/VMess/Trojan runtime execution — opt-in test compile-only, not yet run against stamped core
- ARM64 packaging
- NC/OPN items (product decisions or real-network environments)
- Full release bundle/package behavior validation

## Artifact Verification Output
```
$ ssh sweden 'sha256sum /home/ubuntu/vpn-aio-managed-accounts-20261125/build/out/VPN-AIO-amd64'
407f2fd5067d5b55a7853ef0af1cef16467968844428ef3fa4882f485d586467  /home/ubuntu/vpn-aio-managed-accounts-20261125/build/out/VPN-AIO-amd64

$ ssh sweden '/home/ubuntu/vpn-aio-managed-accounts-20261125/build/out/VPN-AIO-amd64 --version'
(VPN-AIO binary; sing-box version checked separately)

$ ssh sweden '/tmp/vpn-aio-managed-accounts-singbox-stamped-20261126/sing-box version'
sing-box version 1.12.14
Environment: go1.23.1 linux/amd64
Tags: with_gvisor,with_quic,with_dhcp,with_wireguard,with_utls,with_acme,with_clash_api,with_tailscale,with_v2ray_api
Revision: f56d9ab94569d46a48bfc2bcd97eea566ad365c8
CGO: disabled
```