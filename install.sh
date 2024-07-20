#!/bin/sh

set -e

# Define the version and binary name
VERSION="1.0.0"
BINARY_NAME="pretty-alias"
DOWNLOAD_URL="https://github.com/yourusername/your-repo/releases/download/v$VERSION/$BINARY_NAME-linux-amd64"

# Download the binary
echo "Downloading $BINARY_NAME from $DOWNLOAD_URL..."
if ! curl -Lo /usr/local/bin/$BINARY_NAME "$DOWNLOAD_URL"; then
    echo "Error: Failed to download $BINARY_NAME"
    exit 1
fi

# Make the binary executable
echo "Making $BINARY_NAME executable..."
if ! chmod +x /usr/local/bin/$BINARY_NAME; then
    echo "Error: Failed to make $BINARY_NAME executable"
    exit 1
fi

echo "$BINARY_NAME installed successfully!"
