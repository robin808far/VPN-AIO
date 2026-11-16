# Vendored AmneziaWG kernel module

- Source: https://github.com/amnezia-vpn/amneziawg-linux-kernel-module
- Tag: `v3.1.20261113-02`
- Commit: `63ed35de3d3f135b17e8eae0b84949e7700303be`
- Imported directory: `src/`
- Local packaging change: `dkms.conf` PACKAGE_VERSION and `Makefile`
  WIREGUARD_VERSION match `version.h` and `awgsrc.Version`, so DKMS uses the
  correct source directory and can distinguish this build from older modules.

Upstream copyright and SPDX notices are retained in each source file.
Kernel build and connection validation on Ubuntu 24.04 are required before release.
