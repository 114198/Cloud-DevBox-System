#!/bin/bash
# Build Debian package for DevBox CLI

set -e

VERSION="${VERSION:-0.1.0}"
PACKAGE_NAME="devbox-cli"
ARCH="${ARCH:-amd64}"
BUILD_DIR="build-deb"

echo "Building Debian package for DevBox CLI v${VERSION}"
echo "=================================================="

# Clean and create build directory
rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}/DEBIAN"
mkdir -p "${BUILD_DIR}/usr/bin"
mkdir -p "${BUILD_DIR}/usr/share/doc/${PACKAGE_NAME}"
mkdir -p "${BUILD_DIR}/usr/share/bash-completion/completions"
mkdir -p "${BUILD_DIR}/usr/share/zsh/vendor-completions"
mkdir -p "${BUILD_DIR}/usr/share/fish/vendor_completions.d"

# Copy control file
cp debian/control "${BUILD_DIR}/DEBIAN/"
sed -i "s/Version: .*/Version: ${VERSION}/" "${BUILD_DIR}/DEBIAN/control"
sed -i "s/Architecture: .*/Architecture: ${ARCH}/" "${BUILD_DIR}/DEBIAN/control"

# Build binary
echo "Building binary..."
cargo build --release

# Copy binary
cp "../../target/release/devbox" "${BUILD_DIR}/usr/bin/"
chmod 755 "${BUILD_DIR}/usr/bin/devbox"

# Generate shell completions
echo "Generating shell completions..."
"${BUILD_DIR}/usr/bin/devbox" completions bash > "${BUILD_DIR}/usr/share/bash-completion/completions/devbox" 2>/dev/null || true
"${BUILD_DIR}/usr/bin/devbox" completions zsh > "${BUILD_DIR}/usr/share/zsh/vendor-completions/_devbox" 2>/dev/null || true
"${BUILD_DIR}/usr/bin/devbox" completions fish > "${BUILD_DIR}/usr/share/fish/vendor_completions.d/devbox.fish" 2>/dev/null || true

# Create copyright file
cat > "${BUILD_DIR}/usr/share/doc/${PACKAGE_NAME}/copyright" << EOF
Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/
Upstream-Name: devbox-cli
Source: https://github.com/your-org/cloud-devbox

Files: *
Copyright: 2024 Cloud DevBox Team
License: MIT
EOF

# Build package
echo "Building .deb package..."
dpkg-deb --build "${BUILD_DIR}" "dist/${PACKAGE_NAME}_${VERSION}_${ARCH}.deb"

echo ""
echo "Package created: dist/${PACKAGE_NAME}_${VERSION}_${ARCH}.deb"
