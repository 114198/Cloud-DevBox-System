#!/bin/bash
# Cloud DevBox Linux Build Script
# Builds AppImage, DEB, and RPM packages for multiple distributions

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
APP_NAME="cloud-devbox"
VERSION="${VERSION:-0.1.0}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_step "Checking prerequisites..."
    
    # Check Rust
    if ! command -v cargo &> /dev/null; then
        log_error "Rust/Cargo not found. Please install Rust from https://rustup.rs"
        exit 1
    fi
    
    # Check Node.js
    if ! command -v node &> /dev/null; then
        log_error "Node.js not found. Please install Node.js"
        exit 1
    fi
    
    # Check Tauri CLI
    if ! cargo install --list | grep -q "tauri-cli"; then
        log_warn "Installing Tauri CLI..."
        cargo install tauri-cli
    fi
    
    # Check for required system libraries
    local missing_deps=()
    
    # Check for webkit2gtk
    if ! pkg-config --exists webkit2gtk-4.1 2>/dev/null; then
        missing_deps+=("libwebkit2gtk-4.1-dev (Ubuntu/Debian) or webkit2gtk4.1-devel (Fedora/RHEL)")
    fi
    
    # Check for GTK3
    if ! pkg-config --exists gtk+-3.0 2>/dev/null; then
        missing_deps+=("libgtk-3-dev (Ubuntu/Debian) or gtk3-devel (Fedora/RHEL)")
    fi
    
    # Check for AppIndicator
    if ! pkg-config --exists ayatana-appindicator3-0.1 2>/dev/null && \
       ! pkg-config --exists appindicator3-0.1 2>/dev/null; then
        missing_deps+=("libayatana-appindicator3-dev (Ubuntu/Debian) or libappindicator-gtk3-devel (Fedora/RHEL)")
    fi
    
    if [ ${#missing_deps[@]} -gt 0 ]; then
        log_error "Missing dependencies:"
        for dep in "${missing_deps[@]}"; do
            echo "  - $dep"
        done
        echo ""
        echo "Install on Ubuntu/Debian:"
        echo "  sudo apt install libwebkit2gtk-4.1-dev libgtk-3-dev libayatana-appindicator3-dev"
        echo ""
        echo "Install on Fedora/RHEL:"
        echo "  sudo dnf install webkit2gtk4.1-devel gtk3-devel libappindicator-gtk3-devel"
        exit 1
    fi
    
    log_info "Prerequisites check passed"
}

# Build frontend
build_frontend() {
    log_step "Building frontend..."
    
    cd "$PROJECT_ROOT/../web"
    npm install
    npm run build
    cd "$PROJECT_ROOT"
    
    log_info "Frontend build completed"
}

# Build Tauri application
build_tauri() {
    local targets="$1"
    
    log_step "Building Tauri application..."
    
    cd "$PROJECT_ROOT"
    
    local build_args=("tauri" "build")
    
    if [ -n "$targets" ]; then
        build_args+=("--bundles" "$targets")
    fi
    
    cargo "${build_args[@]}"
    
    log_info "Tauri build completed"
}

# Install Linux-specific files
install_linux_files() {
    log_step "Installing Linux-specific files..."
    
    local bundle_dir="$PROJECT_ROOT/target/release/bundle"
    
    # Copy desktop file
    if [ -f "$PROJECT_ROOT/linux/cloud-devbox.desktop" ]; then
        cp "$PROJECT_ROOT/linux/cloud-devbox.desktop" "$bundle_dir/" 2>/dev/null || true
    fi
    
    # Copy metainfo
    if [ -f "$PROJECT_ROOT/linux/cloud-devbox.metainfo.xml" ]; then
        cp "$PROJECT_ROOT/linux/cloud-devbox.metainfo.xml" "$bundle_dir/" 2>/dev/null || true
    fi
    
    # Copy MIME type definition
    if [ -f "$PROJECT_ROOT/linux/cloud-devbox.xml" ]; then
        cp "$PROJECT_ROOT/linux/cloud-devbox.xml" "$bundle_dir/" 2>/dev/null || true
    fi
    
    log_info "Linux files installed"
}

# Build AppImage
build_appimage() {
    log_step "Building AppImage..."
    
    cd "$PROJECT_ROOT"
    cargo tauri build --bundles appimage
    
    local appimage_dir="$PROJECT_ROOT/target/release/bundle/appimage"
    
    if [ -d "$appimage_dir" ]; then
        log_info "AppImage built successfully"
        ls -lh "$appimage_dir"/*.AppImage 2>/dev/null || true
    else
        log_warn "AppImage directory not found"
    fi
}

# Build DEB package
build_deb() {
    log_step "Building DEB package..."
    
    cd "$PROJECT_ROOT"
    cargo tauri build --bundles deb
    
    local deb_dir="$PROJECT_ROOT/target/release/bundle/deb"
    
    if [ -d "$deb_dir" ]; then
        log_info "DEB package built successfully"
        ls -lh "$deb_dir"/*.deb 2>/dev/null || true
        
        # Verify package
        for deb in "$deb_dir"/*.deb; do
            if [ -f "$deb" ]; then
                log_info "Package info for $(basename "$deb"):"
                dpkg-deb --info "$deb" 2>/dev/null || true
            fi
        done
    else
        log_warn "DEB directory not found"
    fi
}

# Build RPM package
build_rpm() {
    log_step "Building RPM package..."
    
    # Check if rpmbuild is available
    if ! command -v rpmbuild &> /dev/null; then
        log_warn "rpmbuild not found. Skipping RPM build."
        log_warn "Install with: sudo apt install rpm (Ubuntu) or sudo dnf install rpm-build (Fedora)"
        return 0
    fi
    
    cd "$PROJECT_ROOT"
    cargo tauri build --bundles rpm
    
    local rpm_dir="$PROJECT_ROOT/target/release/bundle/rpm"
    
    if [ -d "$rpm_dir" ]; then
        log_info "RPM package built successfully"
        ls -lh "$rpm_dir"/*.rpm 2>/dev/null || true
        
        # Verify package
        for rpm in "$rpm_dir"/*.rpm; do
            if [ -f "$rpm" ]; then
                log_info "Package info for $(basename "$rpm"):"
                rpm -qip "$rpm" 2>/dev/null || true
            fi
        done
    else
        log_warn "RPM directory not found"
    fi
}

# Create checksums
create_checksums() {
    log_step "Creating checksums..."
    
    local bundle_dir="$PROJECT_ROOT/target/release/bundle"
    local checksum_file="$bundle_dir/checksums.sha256"
    
    > "$checksum_file"
    
    # Find all packages
    find "$bundle_dir" -type f \( -name "*.AppImage" -o -name "*.deb" -o -name "*.rpm" \) | while read -r file; do
        sha256sum "$file" >> "$checksum_file"
    done
    
    if [ -s "$checksum_file" ]; then
        log_info "Checksums created:"
        cat "$checksum_file"
    fi
}

# List all artifacts
list_artifacts() {
    log_step "Build artifacts:"
    
    local bundle_dir="$PROJECT_ROOT/target/release/bundle"
    
    echo ""
    echo "AppImage:"
    find "$bundle_dir" -name "*.AppImage" -exec ls -lh {} \; 2>/dev/null || echo "  (none)"
    
    echo ""
    echo "DEB packages:"
    find "$bundle_dir" -name "*.deb" -exec ls -lh {} \; 2>/dev/null || echo "  (none)"
    
    echo ""
    echo "RPM packages:"
    find "$bundle_dir" -name "*.rpm" -exec ls -lh {} \; 2>/dev/null || echo "  (none)"
    
    echo ""
}

# Print usage
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --target TARGET    Build specific target (appimage, deb, rpm, all)"
    echo "  --skip-frontend    Skip frontend build"
    echo "  --version VERSION  Set version number"
    echo "  --help             Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                      # Build all packages"
    echo "  $0 --target appimage    # Build only AppImage"
    echo "  $0 --target deb         # Build only DEB package"
    echo "  $0 --target rpm         # Build only RPM package"
}

# Main build process
main() {
    local target="all"
    local skip_frontend=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --target)
                target="$2"
                shift 2
                ;;
            --skip-frontend)
                skip_frontend=true
                shift
                ;;
            --version)
                VERSION="$2"
                shift 2
                ;;
            --help)
                usage
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
    
    log_info "Starting Linux build process"
    log_info "Version: $VERSION, Target: $target"
    
    # Check prerequisites
    check_prerequisites
    
    # Build frontend
    if [ "$skip_frontend" = false ]; then
        build_frontend
    fi
    
    # Build based on target
    case $target in
        appimage)
            build_appimage
            ;;
        deb)
            build_deb
            ;;
        rpm)
            build_rpm
            ;;
        all)
            build_appimage
            build_deb
            build_rpm
            ;;
        *)
            log_error "Unknown target: $target"
            usage
            exit 1
            ;;
    esac
    
    # Install Linux-specific files
    install_linux_files
    
    # Create checksums
    create_checksums
    
    # List artifacts
    list_artifacts
    
    log_info "Linux build completed successfully!"
}

main "$@"
