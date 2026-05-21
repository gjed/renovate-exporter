#!/usr/bin/env bash
# Install the OTel Weaver CLI for CI environments.
# Usage: ./scripts/install-weaver.sh [version]
set -euo pipefail

VERSION="${1:-0.23.0}"
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64)  ARCH="x86_64" ;;
  aarch64|arm64) ARCH="aarch64" ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

case "$OS" in
  linux)  SUFFIX="unknown-linux-gnu" ;;
  darwin) SUFFIX="apple-darwin" ;;
  *) echo "Unsupported OS: $OS" >&2; exit 1 ;;
esac

TARBALL="weaver-${ARCH}-${SUFFIX}.tar.gz"
URL="https://github.com/open-telemetry/weaver/releases/download/v${VERSION}/${TARBALL}"

DEST="${WEAVER_INSTALL_DIR:-/usr/local/bin}"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

echo "Downloading Weaver v${VERSION} from ${URL}..."
curl -fsSL "$URL" -o "$TMP/$TARBALL"
tar -xzf "$TMP/$TARBALL" -C "$TMP"
install -m 755 "$TMP/weaver" "$DEST/weaver"
echo "Installed: $(weaver --version)"
