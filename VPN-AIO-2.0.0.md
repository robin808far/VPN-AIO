# VPN-AIO 2.0.0 integration status

Target: fresh installations on Ubuntu 24.04, Linux amd64.
This branch is work in progress, not a published release.

## Integrated

- Application version: 2.0.0; application name: VPN-AIO.
- Release repository: robin808far/VPN-AIO; asset: VPN-AIO-amd64.
- Installation directory: /opt/VPN-AIO; management command and default service: VPN-AIO.
- Database basename: VPN-AIO.db.
- Xray source: robin808far/Xray-core, pinned to the merge of upstream v26.11.9
  with the native AnyTLS, TUIC and NaiveProxy additions.
- AmneziaWG kernel source: v3.1.20261113-02 (see awgsrc/UPSTREAM.md).
- AmneziaWG control library: github.com/Jipok/wgctrl-go v1.3.1.

## Required before release

- Complete visible branding and documentation review; retain upstream credits/licenses.
- Validate AWG 3.1 with the current compatibility configuration. Exposing new
  obfuscation parameters requires matching UI, validation and client export support.
- Linux panel build with CGO/sqlite enabled and all daemon/geodata bundles.
- Targeted Go tests and Ubuntu 24.04 DKMS, connection, traffic and user-limit tests.
- Validate install, management menu, service, update URLs and uninstall on a fresh host.
- Publish the tested release asset only after these checks.

The Windows cross-compile with CGO disabled is a compile check only: it is not a
deployable release (sqlite requires CGO). Real kernel testing requires Linux.

VPN-AIO derives from Sir-MmD/vpn-ui and 3x-ui. The existing GPL-3.0 license and
third-party copyright/license notices remain applicable.
