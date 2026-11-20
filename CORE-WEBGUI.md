# Core WebGUI integration

Working branch: `feature/core-webgui`.

Target version: **2.1.0**. Published 2.0.0 tags and assets must remain intact.
The test host `sweden` currently runs 2.0.0; back up its database and configuration
before deploying a test build, then verify a real upgrade with existing settings.

The installer accepts `VPN_AIO_VERSION=2.0.0` (or `v2.0.0`) to install a specific
published version. Omission retains the latest-release behavior. Example:

```sh
curl -fsSL https://raw.githubusercontent.com/robin808far/VPN-AIO/main/deploy.sh -o deploy.sh
sudo env VPN_AIO_VERSION=2.0.0 bash deploy.sh
```

This selector is introduced on this development branch and becomes available on
main when merged. It selects the binary; it does not migrate a newer database
backward. Restore the matching backup when rolling back an existing installation.

Scope approved: all relevant new Xray core GUI options plus AmneziaWG 3.1
configuration, preserving the existing Vue interface and installed configurations.

References:
- Pinned Xray core source in `third_party/Xray-core` (26.11.9 integration).
- Current MHSanaei/3x-ui frontend schemas and adapters (React implementation).
- Vendored AmneziaWG 3.1 kernel UAPI and wgctrl-go v1.3.1.

## Work items

- [x] Audit upstream fields against current models and serializers for the delivered XDRIVE,
      MASQUE and FinalMask paths; remaining unrelated upstream fields stay out of scope.
- [x] XDRIVE transport: initial local/Google Drive/S3 forms, defaults and
      inbound/outbound serialization. Core config validation covers all three.
- [x] MASQUE outbound: model, TLS/auth/WARP form controls, h2/h3 core config
      validation and live TCP/UDP/accounting/authentication smoke tests.
- [x] MASQUE inbound: account management, usable address pools, TLS/ALPN inbound form,
      generated `pass` credentials, live add/remove/re-enable, accounting and JSON
      subscription output are covered. No raw URI is emitted because MASQUE has no
      standardized share-link format.
- [x] FinalMask current QUIC parameters: BBR profile, Brutal compensation,
      receive windows, timeouts, GSO, Chrome parrot, stateless reset and UDP hop
      round-trip through the Vue model and form.
- [x] FinalMask TCP/UDP inventory/UI audit against the pinned core: legacy mKCP aliases,
      filter, XMC, xDNS structured domains/resolvers, Realm, UDP hop, Sudoku, custom
      headers, Noise expressions/scripts and Gecko packet sizing are represented with
      legacy migrations preserved.
- [x] Audit XHTTP, TLS/REALITY, socket, DNS and routing settings against the pinned
      schema. Structured TLS/REALITY/socket/sniffing models preserve unedited core
      keys; the routing modal retains hidden selectors during edits. Browserless
      round-trip tests and pinned-core parsing cover inbound/outbound XHTTP with
      TLS/REALITY, socket options, DNS and routing.
- [x] AWG S3/S4, I1-I5, header protection, padding/timing ranges,
      random trailers and cookie control: inbound form/model, outbound parser,
      kernel netlink, client config and Clash option serialization.
- [x] AWG parameter validation, compatible defaults and MTU handling.
- [x] Round-trip tests ensuring edits do not discard existing settings.
- [x] Actual pinned-core configuration validation and UI tests for XDRIVE,
      MASQUE outbound, FinalMask and AWG model paths.
- [x] Linux core-config checks and live native-protocol smoke tests for MASQUE
      (h2/h3), FinalMask, AnyTLS, TUIC and Naive.
- [x] Controlled authenticated upgrade/rollback on `sweden`: full-geo build,
      live authenticated API checks (34/34), test-fixture cleanup and exact pre-test
      binary/systemd unit/SQLite restoration with hash and integrity verification.
- [x] Install the full-geo 2.1.0 candidate on `sweden` and route certificate-
      verified HTTPS through an existing VLESS/XHTTP account; Xray uplink and
      downlink counters both increased. Restore and verify the exact pre-test
      binary, service unit and SQLite files; service/Xray/AWG healthy afterward.
- [x] Verify AWG 3.1 exports with upstream Linux `amneziawg-tools` and the 3.1
      kernel module: a live panel-generated profile imports and round-trips, while
      a full-option config completes a real handshake and bidirectional tunnel ping.
      Export booleans now use the `on`/`off` syntax required by the client parser.
- [x] Full-candidate container userland lifecycle matrix: 13/13 distributions
      passed authenticated startup, native Xray smoke, XDRIVE/FinalMask config
      checks and isolated uninstall/file/firewall/policy-route cleanup.
- [x] Full uninstall on `sweden` stopped/disabled the systemd unit, removed app
      files/data-plane routes and the AmneziaWG DKMS registration. It exposed a
      surviving `awg1`; uninstall now explicitly stops manifest-owned WireGuard,
      AmneziaWG and GRE kernel interfaces before removing their backing features.
      The rebuilt candidate then passed all host uninstall assertions and the exact
      pre-test files/database/service were restored.
- [x] Repeated the full-host uninstall cycle with the kernel-interface cleanup fix
      and a certificate-verified HTTPS tunnel request. Result: 39/39 checks passed,
      including systemd stop/disable, TCP/UDP listener release, AWG interface/DKMS
      cleanup and exact restoration of the original service/binary/database.
- [x] Regression test verifies that uninstall attempts each daemon-less kernel
      core cleanup even when one cleanup reports an error.
- [x] Per operator choice, skip the cross-distro VM tunnel/kernel/DKMS matrix.
      The userland matrix shares one kernel and does not claim tunnel support on
      untested distro kernels; the live Ubuntu/AWG checks remain verified.

Do not copy React components into the Vue application. Use upstream schemas as
reference while checking every option against the pinned executable's actual API.
Preserve original licenses and attribution for any adapted source. Do not mark
unsupported subscription formats as successfully exported.
