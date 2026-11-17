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

- [ ] Audit upstream fields against current models and serializers.
- [x] XDRIVE transport: initial local/Google Drive/S3 forms, defaults and
      inbound/outbound serialization. Core config validation covers all three.
- [ ] MASQUE protocol and transport: core configuration, account management,
      inbound/outbound forms and supported subscription output.
- [ ] FinalMask TCP/UDP mask options and current QUIC parameters.
- [ ] Audit updated XHTTP, TLS/REALITY, socket, DNS and routing settings.
- [x] AWG S3/S4, I1-I5, header protection, padding/timing ranges,
      random trailers and cookie control: inbound form/model, outbound parser,
      kernel netlink, client config and Clash option serialization.
- [ ] AWG parameter validation, compatible defaults and MTU handling.
- [ ] Round-trip tests ensuring edits do not discard existing settings.
- [ ] Actual pinned-core configuration validation and UI tests.
- [ ] Linux runtime checks and release notes with client compatibility limits.

Do not copy React components into the Vue application. Use upstream schemas as
reference while checking every option against the pinned executable's actual API.
Preserve original licenses and attribution for any adapted source. Do not mark
unsupported subscription formats as successfully exported.
