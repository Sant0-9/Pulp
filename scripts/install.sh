#!/bin/sh
set -e

REPO="sant0-9/pulp"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "${GREEN}Installing Pulp...${NC}"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    darwin) OS="darwin" ;;
    linux) OS="linux" ;;
    mingw*|msys*|cygwin*) OS="windows" ;;
    *) echo "${RED}Unsupported OS: $OS${NC}"; exit 1 ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "${RED}Unsupported architecture: $ARCH${NC}"; exit 1 ;;
esac

# Get latest version
echo "Fetching latest version..."
VERSION=$(curl -sL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$VERSION" ]; then
    echo "${RED}Failed to fetch latest version${NC}"
    exit 1
fi

echo "Latest version: $VERSION"

# Download
EXT="tar.gz"
if [ "$OS" = "windows" ]; then
    EXT="zip"
fi

FILENAME="pulp_${VERSION#v}_${OS}_${ARCH}.${EXT}"
URL="https://github.com/$REPO/releases/download/$VERSION/$FILENAME"

echo "Downloading $URL..."
TMP_DIR=$(mktemp -d)
cd "$TMP_DIR"

if ! curl -fsSL "$URL" -o "$FILENAME"; then
    echo "${RED}Failed to download $URL${NC}"
    exit 1
fi

# Extract
echo "Extracting..."
if [ "$EXT" = "zip" ]; then
    unzip -q "$FILENAME"
else
    tar xzf "$FILENAME"
fi

# Install
echo "Installing to $INSTALL_DIR..."
if [ -w "$INSTALL_DIR" ]; then
    mv pulp "$INSTALL_DIR/"
else
    sudo mv pulp "$INSTALL_DIR/"
fi

# Install Python bridge
PULP_SHARE="${HOME}/.local/share/pulp"
mkdir -p "$PULP_SHARE/python"
if [ -d "python" ]; then
    cp -r python/* "$PULP_SHARE/python/"
fi

# Cleanup
cd /
rm -rf "$TMP_DIR"

# Verify
if command -v pulp >/dev/null 2>&1; then
    echo "${GREEN}Pulp installed successfully!${NC}"
    echo ""
    pulp --version
    echo ""
    echo "Run 'pulp' to get started."
else
    echo "${YELLOW}Pulp installed but not in PATH.${NC}"
    echo "Add $INSTALL_DIR to your PATH."
fi
