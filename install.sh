#!/bin/sh

set -e

# Define the version and binary name
VERSION="1.1.2"
BINARY_NAME="pretty-alias"
# DOWNLOAD_URL="https://github.com/sharon-xa/pretty-alias/releases/download/v$VERSION/$BINARY_NAME-linux-amd64"
DOWNLOAD_URL="https://github.com/sharon-xa/pretty-alias/releases/download/v$VERSION/$BINARY_NAME"

# Download the binary
echo "Downloading $BINARY_NAME from $DOWNLOAD_URL..."
if ! curl -Lo /usr/local/bin/$BINARY_NAME "$DOWNLOAD_URL"; then
    echo
    echo "Error: Failed to download $BINARY_NAME"
    exit 1
fi

# Make the binary executable
echo
echo "Making $BINARY_NAME executable..."
if ! chmod +x /usr/local/bin/$BINARY_NAME; then
    echo
    echo "Error: Failed to make $BINARY_NAME executable"
    exit 1
fi

echo "$BINARY_NAME installed successfully!"
