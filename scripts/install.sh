#!/bin/sh
# GitX installer
# Usage: curl -fsSL https://raw.githubusercontent.com/deveasyclick/gitx/main/scripts/install.sh | sh
#        GITX_INSTALL_DIR=/custom/path curl -fsSL https://raw.githubusercontent.com/deveasyclick/gitx/main/scripts/install.sh | sh
#        curl -fsSL https://raw.githubusercontent.com/deveasyclick/gitx/main/scripts/install.sh | sudo sh

set -eu

REPO="deveasyclick/gitx"
detect_bin_dir() {
  # Use explicit override if set
  if [ -n "${GITX_INSTALL_DIR:-}" ]; then
    echo "$GITX_INSTALL_DIR"
    return
  fi

  # Check if /usr/local/bin is writable (no sudo needed)
  if [ -w "/usr/local/bin" ] 2>/dev/null; then
    echo "/usr/local/bin"
    return
  fi

  # Fall back to user local bin
  echo "${HOME}/.local/bin"
}

BIN_DIR="$(detect_bin_dir)"

# Detect OS and arch
detect_platform() {
  OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
  ARCH="$(uname -m)"

  case "$OS" in
    linux)   OS="linux" ;;
    darwin)  OS="darwin" ;;
    mingw*|msys*|cygwin*) OS="windows" ;;
    *)       echo "Unsupported OS: $OS"; exit 1 ;;
  esac

  case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *)            echo "Unsupported architecture: $ARCH"; exit 1 ;;
  esac
}

# Fetch the latest release tag from GitHub
latest_tag() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null \
      | grep '"tag_name"' | cut -d'"' -f4
  elif command -v wget >/dev/null 2>&1; then
    wget -qO- "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null \
      | grep '"tag_name"' | cut -d'"' -f4
  else
    echo "Need curl or wget to fetch releases"; exit 1
  fi
}

install() {
  detect_platform
  echo "Detected: $OS/$ARCH"

  TAG="$(latest_tag)"
  if [ -z "$TAG" ]; then
    echo "Could not determine latest release"
    exit 1
  fi
  echo "Latest release: $TAG"

  # Build archive URL (goreleaser publishes tar.gz for linux/darwin, zip for windows)
  ARCHIVE="gitx_${TAG#v}_${OS}_${ARCH}"
  if [ "$OS" = "windows" ]; then
    ARCHIVE="${ARCHIVE}.zip"
    UNPACK="unzip"
  else
    ARCHIVE="${ARCHIVE}.tar.gz"
    UNPACK="tar xzf"
  fi
  URL="https://github.com/$REPO/releases/download/$TAG/$ARCHIVE"

  # Verify unpack tool is available
  if ! command -v "${UNPACK%% *}" >/dev/null 2>&1; then
    echo "Need $UNPACK to extract the archive"
    exit 1
  fi

  # Download to temp
  TMP="$(mktemp -d)"
  trap 'rm -rf "$TMP"' EXIT

  echo "Downloading $URL ..."
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$URL" -o "$TMP/$ARCHIVE"
  elif command -v wget >/dev/null 2>&1; then
    wget -q "$URL" -O "$TMP/$ARCHIVE"
  fi

  # Extract
  (
    cd "$TMP"
    $UNPACK "$ARCHIVE"
  )

  chmod +x "$TMP/gitx"

  # Install
  if [ ! -d "$BIN_DIR" ]; then
    mkdir -p "$BIN_DIR"
  fi
  mv "$TMP/gitx" "$BIN_DIR/gitx"

  echo ""
  echo "Installed gitx to $BIN_DIR/gitx"
  echo "Make sure $BIN_DIR is in your PATH."
  echo ""
  echo "Run 'gitx setup' to configure your AI provider."
}

install
