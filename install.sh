#!/usr/bin/env sh
# cvox install script. Detects platform and downloads the latest release binary.

set -eu

# Detect OS
OS="$(uname -s)"
case "$OS" in
  Linux) OS=linux ;;
  Darwin) OS=darwin ;;
  CYGWIN*|MINGW*|MSYS*) OS=windows ;;
  *) echo "Unsupported OS: $OS" >&2; exit 1 ;;
esac

# Detect arch
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# Resolve the latest version from GitHub if CVOX_VERSION is not set.
# GitHub's "latest" redirect is at /releases/latest/download/<asset>, not
# /releases/download/latest/<asset>, so we must fetch the actual tag.
if [ -z "${CVOX_VERSION:-}" ]; then
  if command -v curl >/dev/null 2>&1; then
    VERSION=$(curl -sS https://api.github.com/repos/lmk123/cvox/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' | sed 's/^v//')
  elif command -v wget >/dev/null 2>&1; then
    VERSION=$(wget -qO- https://api.github.com/repos/lmk123/cvox/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' | sed 's/^v//')
  else
    echo "Neither curl nor wget found. Please install one of them." >&2
    exit 1
  fi
  if [ -z "$VERSION" ]; then
    echo "Failed to fetch latest version from GitHub." >&2
    exit 1
  fi
else
  VERSION="$CVOX_VERSION"
fi

# Determine install directory
if [ -d "$HOME/.local/bin" ]; then
  BINDIR="$HOME/.local/bin"
elif [ -d "/usr/local/bin" ]; then
  BINDIR="/usr/local/bin"
else
  BINDIR="$HOME/.local/bin"
  mkdir -p "$BINDIR"
fi

# Filename (no 'v' prefix - goreleaser strips it)
if [ "$OS" = "windows" ]; then
  EXT=".zip"
else
  EXT=".tar.gz"
fi
FILE="cvox_${VERSION}_${OS}_${ARCH}${EXT}"

# Download URL (tag needs 'v' prefix)
TAG="v${VERSION}"
URL="https://github.com/lmk123/cvox/releases/download/${TAG}/${FILE}"

echo "Downloading cvox ${VERSION} for ${OS}/${ARCH}..."
echo "URL: $URL"

# Download
TMPDIR=$(mktemp -d)
cd "$TMPDIR"
if command -v curl >/dev/null 2>&1; then
  curl -fsSL -o "$FILE" "$URL"
elif command -v wget >/dev/null 2>&1; then
  wget -qO "$FILE" "$URL"
else
  echo "Neither curl nor wget found. Please install one of them." >&2
  exit 1
fi

# Extract
if [ "$OS" = "windows" ]; then
  unzip -q "$FILE"
else
  tar -xzf "$FILE"
fi

# Install
chmod +x cvox
mv cvox "$BINDIR/cvox"

# Cleanup
cd -
rm -rf "$TMPDIR"

echo "cvox installed to $BINDIR/cvox"
if ! echo "$PATH" | grep -q "$BINDIR"; then
  echo "Note: $BINDIR is not in your PATH. Add it to your shell profile."
fi
