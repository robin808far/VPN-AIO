#!/usr/bin/env bash
# VPN-AIO installer, 3x-ui style one-liner:
#
#   bash <(curl -Ls https://raw.githubusercontent.com/robin808far/VPN-AIO/main/install.sh)              # latest
#   bash <(curl -Ls https://raw.githubusercontent.com/robin808far/VPN-AIO/main/install.sh) v3.1.0      # a specific release
#
# The optional argument is the release tag (v prefix optional) and pins the panel
# binary to that release via VPN_AIO_VERSION, exactly like 3x-ui.
set -euo pipefail

REPO="robin808far/VPN-AIO"
BRANCH="main"

if (( $# > 1 )); then
    printf 'usage: bash <(curl -Ls https://raw.githubusercontent.com/%s/%s/install.sh) [ vX.Y.Z ]\n' "$REPO" "$BRANCH" >&2
    exit 2
fi

VERSION="${1:-latest}"
if [[ "$VERSION" != "latest" ]]; then
    if [[ ! "$VERSION" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.-]+)?$ ]]; then
        printf 'error: invalid version "%s" — use a release tag like v3.1.0 (see https://github.com/%s/releases)\n' "$VERSION" "$REPO" >&2
        exit 2
    fi
    VERSION="v${VERSION#v}"
fi

# deploy.sh writes /opt/VPN-AIO and drives systemctl, so it needs root. Re-exec
# with sudo when we are not root (the curl one-liner itself needs no sudo).
# Fetch deploy.sh before the root check: when run through `bash <(curl ...)`,
# $0 is just "bash", so re-execing this script by name would not work — we exec
# the downloaded deploy.sh instead.

DEPLOY_URL="https://raw.githubusercontent.com/$REPO/$BRANCH/deploy.sh"
TMP_DEPLOY="$(mktemp /tmp/vpn-aio-deploy.XXXXXX.sh)"
trap 'rm -f "$TMP_DEPLOY"' EXIT
curl -fsSL "$DEPLOY_URL" -o "$TMP_DEPLOY" \
    || { printf 'error: could not download %s\n' "$DEPLOY_URL" >&2; exit 1; }

if [[ $EUID -ne 0 ]]; then
    if command -v sudo >/dev/null 2>&1; then
        if [[ "$VERSION" == "latest" ]]; then
            exec sudo bash "$TMP_DEPLOY"
        else
            exec sudo env VPN_AIO_VERSION="$VERSION" bash "$TMP_DEPLOY"
        fi
    fi
    printf 'error: this installer needs root (run it as root, or with sudo)\n' >&2
    exit 1
fi

if [[ "$VERSION" == "latest" ]]; then
    exec bash "$TMP_DEPLOY"
else
    exec env VPN_AIO_VERSION="$VERSION" bash "$TMP_DEPLOY"
fi
