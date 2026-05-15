#!/usr/bin/env bash
# DayWrap CLI installer
# Usage: curl -fsSL https://raw.githubusercontent.com/arjun921/daywrap/main/cli/install.sh | bash
set -euo pipefail

REPO="arjun921/daywrap"
BINARY_NAME="daywrap"
INSTALL_DIR="${DAYWRAP_INSTALL_DIR:-}"

# Detect OS
OS="$(uname -s)"
case "$OS" in
  Darwin) OS="darwin" ;;
  Linux)  OS="linux"  ;;
  *)
    echo "error: unsupported OS: $OS" >&2
    exit 1
    ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64)        ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    echo "error: unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

ASSET="${BINARY_NAME}-${OS}-${ARCH}"
URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"

# Pick install directory: prefer ~/.local/bin (no sudo), fall back to /usr/local/bin
if [ -z "$INSTALL_DIR" ]; then
  if [ -d "$HOME/.local/bin" ] || mkdir -p "$HOME/.local/bin" 2>/dev/null; then
    INSTALL_DIR="$HOME/.local/bin"
  else
    INSTALL_DIR="/usr/local/bin"
  fi
fi

TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT

echo "Downloading daywrap (${OS}/${ARCH})..."
if ! curl -fsSL "$URL" -o "$TMP"; then
  echo "error: download failed from $URL" >&2
  exit 1
fi
chmod +x "$TMP"

DEST="${INSTALL_DIR}/${BINARY_NAME}"
if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP" "$DEST"
else
  echo "Installing to $INSTALL_DIR (requires sudo)..."
  sudo mv "$TMP" "$DEST"
fi

echo ""
echo "  daywrap installed → $DEST"
echo ""

# Verify
if "$DEST" --version >/dev/null 2>&1; then
  echo "  $("$DEST" --version)"
fi

# PATH hint
case ":$PATH:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    echo ""
    echo "  Add ${INSTALL_DIR} to your PATH:"
    echo "    export PATH=\"\$PATH:${INSTALL_DIR}\""
    ;;
esac
echo ""
