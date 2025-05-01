#!/bin/bash
# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: Copyright 2025, Scott Friedman and Project Contributors

# Cross-platform build script for AWS Instance Finder

VERSION="0.9.0"
OUTPUT_DIR="dist"

# Platforms to build for
declare -a PLATFORMS=(
  "windows:amd64"
  "windows:arm64"
  "darwin:amd64"
  "darwin:arm64"
  "linux:amd64"
  "linux:arm64"
)

# Create output directories
mkdir -p $OUTPUT_DIR

# Ensure we're in the right directory
cd "$(dirname "$0")" || exit 1

echo "==> Building AWS Instance Finder v$VERSION"

# Build Go version for each platform
for platform in "${PLATFORMS[@]}"; do
  IFS=":" read -r os arch <<< "$platform"
  
  echo "==> Building for $os/$arch..."
  
  # Determine binary name with extension
  binary_name="instancefinder"
  if [ "$os" == "windows" ]; then
    binary_name="${binary_name}.exe"
  fi
  
  # Full file path for the binary
  binary_path="$OUTPUT_DIR/instancefinder-${os}-${arch}"
  if [ "$os" == "windows" ]; then
    binary_path="$OUTPUT_DIR/instancefinder-${os}-${arch}.exe"
  fi
  
  # Build the binary
  echo "    Building binary..."
  env GOOS=$os GOARCH=$arch go build -o "$binary_path" \
    -ldflags "-X main.version=$VERSION" \
    ./instancefinder.go
  
  # Create archive
  echo "    Creating archive..."
  if [ "$os" == "windows" ]; then
    (cd "$OUTPUT_DIR" && zip -q "instancefinder-${os}-${arch}-v${VERSION}.zip" "instancefinder-${os}-${arch}.exe" -j "../LICENSE" "../README.md")
  else
    (cd "$OUTPUT_DIR" && tar -czf "instancefinder-${os}-${arch}-v${VERSION}.tar.gz" "instancefinder-${os}-${arch}" -C .. LICENSE README.md)
  fi
  
  echo "    Done!"
done

# Generate SHA256 checksums
echo "==> Generating checksums..."
(cd "$OUTPUT_DIR" && shasum -a 256 *.zip *.tar.gz > SHA256SUMS.txt)

echo "==> Build complete!"
echo "Binaries and archives are in the '$OUTPUT_DIR' directory."
echo "Checksums are in '$OUTPUT_DIR/SHA256SUMS.txt'."