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

# Latest version (can be overridden with CVOX_VERSION)
VERSION="${CVOX_VERSION:-latest}"

# Determine install directory
if [ -d "$HOME/.local/bin" ]; then
  BINDIR="$HOME/.local/bin"
elif [ -d "/usr/local/bin" ]; then
  BINDIR="/usr/local/bin"
else
  BINDIR="$HOME/.local/bin"
  mkdir -p "$BINDIR"
fi

# Filename
if [ "$OS" = "windows" ]; then
  EXT=".zip"
else
  EXT=".tar.gz"
fi
FILE="cvox_${VERSION}_${OS}_${ARCH}${EXT}"

# Download URL
URL="https://github.com/lmk123/cvox/releases/download/${VERSION}/${FILE}"

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
