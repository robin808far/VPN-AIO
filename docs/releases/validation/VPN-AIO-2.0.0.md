# VPN-AIO 2.0.0 integration status

Target: fresh installations on Ubuntu 24.04, Linux amd64.
Release validation record for the 2.0.0 build.

## Integrated

- Application version: 2.0.0; application name: VPN-AIO.
- Release repository: robin808far/VPN-AIO; asset: VPN-AIO-amd64.
- Installation directory: /opt/VPN-AIO; management command and default service: VPN-AIO.
- Database basename: VPN-AIO.db.
- Xray source: robin808far/Xray-core, pinned to the merge of upstream v26.11.9
  with the native AnyTLS, TUIC and NaiveProxy additions.
- AmneziaWG kernel source: v3.1.20261113-02 (see awgsrc/UPSTREAM.md).
- AmneziaWG control library: github.com/Jipok/wgctrl-go v1.3.1.

## Scope and remaining coverage

- AWG 3.1 UI and client export are part of the 2.1.0 development branch; this
  2.0.0 release intentionally retains the compatibility profile.
- The full cross-distribution/protocol matrix, complete quota/device-limit flows,
  and destructive uninstall were not exercised for this release.
- Internal daemon paths retain upstream names for bundle compatibility. Public
  installation, service, application, release and primary UI branding use VPN-AIO.

## Completed runtime checks

On Ubuntu 24.04.5 amd64 with kernel 7.0.0-1017-aws:

- DKMS build/install/load of AmneziaWG 3.1.20261113-02 succeeded.
- The opt-in `TestAWGKernelHandshakeAndPeerRemoval` passed: actual kernel
  handshake, encrypted tunnel ping between network namespaces, nonzero receive
  and transmit counters, and traffic rejection after peer removal.
- `test_unit/native-core-smoke.py` passed against Xray 26.11.9: AnyTLS TCP/UoT,
  TUIC TCP and both UDP relay modes, and Naive HTTP/2 plus HTTP/3 carriers.
  Every case checked per-user traffic counters and wrong-password rejection.
- All panel Go tests (`go test ./...`) and the native-core smoke tests passed
  in GitHub Actions on Ubuntu 24.04; the CGO/sqlite build also passed.
- The full Linux amd64 bundle built successfully, including all static VPN
  daemons and geo databases. The resulting binary is approximately 338 MiB.
- Fresh binary installation, randomized login, management-menu installation,
  systemd startup, primary pages and authenticated status/config/inbounds APIs
  passed on the test host. The embedded core reported Xray 26.11.9.

The full Incus VM matrix is not available on this small test host (no /dev/kvm).
These targeted runtime tests do not claim coverage of every bundled protocol.

The Windows cross-compile with CGO disabled is a compile check only: it is not a
deployable release (sqlite requires CGO). Real kernel testing requires Linux.

VPN-AIO derives from Sir-MmD/vpn-ui and 3x-ui. The existing GPL-3.0 license and
third-party copyright/license notices remain applicable.
