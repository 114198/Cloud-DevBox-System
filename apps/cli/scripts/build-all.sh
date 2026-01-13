#!/bin/bash
# Cross-platform build script for DevBox CLI
# Builds binaries for Windows, macOS, and Linux

set -e

VERSION="${VERSION:-$(grep '^version' ../../Cargo.toml | head -1 | cut -d'"' -f2)}"
OUTPUT_DIR="dist"
BINARY_NAME="devbox"

echo "Building DevBox CLI v${VERSION}"
echo "================================"

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Define targets
TARGETS=(
    "x86_64-unknown-linux-gnu"
    "x86_64-unknown-linux-musl"
    "aarch64-unknown-linux-gnu"
    "x86_64-apple-darwin"
    "aarch64-apple-darwin"
    "x86_64-pc-windows-msvc"
    "x86_64-pc-windows-gnu"
)

# Build for each target
for target in "${TARGETS[@]}"; do
    echo ""
    echo "Building for ${target}..."
    
    # Check if target is installed
    if ! rustup target list --installed | grep -q "${target}"; then
        echo "  Installing target ${target}..."
        rustup target add "${target}" || {
            echo "  Warning: Could not install target ${target}, skipping..."
            continue
        }
    fi
    
    # Build
    if cargo build --release --target "${target}" 2>/dev/null; then
        # Determine output filename
        case "${target}" in
            *windows*)
                src_binary="../../target/${target}/release/${BINARY_NAME}.exe"
                dst_binary="${OUTPUT_DIR}/${BINARY_NAME}-${VERSION}-${target}.exe"
                ;;
            *)
                src_binary="../../target/${target}/release/${BINARY_NAME}"
                dst_binary="${OUTPUT_DIR}/${BINARY_NAME}-${VERSION}-${target}"
                ;;
        esac
        
        if [ -f "${src_binary}" ]; then
            cp "${src_binary}" "${dst_binary}"
            echo "  Created: ${dst_binary}"
            
            # Create compressed archive
            case "${target}" in
                *windows*)
                    if command -v zip &> /dev/null; then
                        (cd "${OUTPUT_DIR}" && zip -q "${BINARY_NAME}-${VERSION}-${target}.zip" "$(basename ${dst_binary})")
                        echo "  Created: ${OUTPUT_DIR}/${BINARY_NAME}-${VERSION}-${target}.zip"
                    fi
                    ;;
                *)
                    tar -czf "${OUTPUT_DIR}/${BINARY_NAME}-${VERSION}-${target}.tar.gz" -C "${OUTPUT_DIR}" "$(basename ${dst_binary})"
                    echo "  Created: ${OUTPUT_DIR}/${BINARY_NAME}-${VERSION}-${target}.tar.gz"
                    ;;
            esac
        fi
    else
        echo "  Warning: Build failed for ${target}, skipping..."
    fi
done

echo ""
echo "Build complete! Binaries are in ${OUTPUT_DIR}/"
ls -la "${OUTPUT_DIR}/"
