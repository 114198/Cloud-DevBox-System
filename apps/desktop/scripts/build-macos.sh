#!/bin/bash
# Cloud DevBox macOS Build Script
# Builds Universal Binary (Intel + Apple Silicon) DMG with code signing and notarization

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
APP_NAME="Cloud DevBox"
BUNDLE_ID="io.devbox.desktop"
VERSION="${VERSION:-0.1.0}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
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

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
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
    
    # Check Xcode command line tools
    if ! xcode-select -p &> /dev/null; then
        log_error "Xcode command line tools not found. Run: xcode-select --install"
        exit 1
    fi
    
    # Check for both architectures
    if ! rustup target list --installed | grep -q "aarch64-apple-darwin"; then
        log_warn "Adding aarch64-apple-darwin target..."
        rustup target add aarch64-apple-darwin
    fi
    
    if ! rustup target list --installed | grep -q "x86_64-apple-darwin"; then
        log_warn "Adding x86_64-apple-darwin target..."
        rustup target add x86_64-apple-darwin
    fi
    
    log_info "Prerequisites check passed"
}

# Build frontend
build_frontend() {
    log_info "Building frontend..."
    
    cd "$PROJECT_ROOT/../web"
    npm install
    npm run build
    cd "$PROJECT_ROOT"
    
    log_info "Frontend build completed"
}

# Build for a specific architecture
build_arch() {
    local arch=$1
    log_info "Building for $arch..."
    
    cd "$PROJECT_ROOT"
    
    cargo tauri build --target "$arch" --bundles app
    
    log_info "Build for $arch completed"
}

# Create Universal Binary
create_universal_binary() {
    log_info "Creating Universal Binary..."
    
    local arm64_app="$PROJECT_ROOT/target/aarch64-apple-darwin/release/bundle/macos/$APP_NAME.app"
    local x86_64_app="$PROJECT_ROOT/target/x86_64-apple-darwin/release/bundle/macos/$APP_NAME.app"
    local universal_app="$PROJECT_ROOT/target/universal-apple-darwin/release/bundle/macos/$APP_NAME.app"
    
    # Create output directory
    mkdir -p "$(dirname "$universal_app")"
    
    # Copy the arm64 app as base
    cp -R "$arm64_app" "$universal_app"
    
    # Create universal binary using lipo
    local arm64_binary="$arm64_app/Contents/MacOS/$APP_NAME"
    local x86_64_binary="$x86_64_app/Contents/MacOS/$APP_NAME"
    local universal_binary="$universal_app/Contents/MacOS/$APP_NAME"
    
    lipo -create -output "$universal_binary" "$arm64_binary" "$x86_64_binary"
    
    # Verify universal binary
    log_info "Verifying Universal Binary..."
    file "$universal_binary"
    lipo -info "$universal_binary"
    
    log_info "Universal Binary created successfully"
}

# Sign the application
sign_app() {
    log_info "Signing application..."
    
    local app_path="$PROJECT_ROOT/target/universal-apple-darwin/release/bundle/macos/$APP_NAME.app"
    local signing_identity="${APPLE_SIGNING_IDENTITY:-}"
    local entitlements="$PROJECT_ROOT/entitlements.plist"
    local child_entitlements="$PROJECT_ROOT/entitlements.child.plist"
    
    if [ -z "$signing_identity" ]; then
        log_warn "APPLE_SIGNING_IDENTITY not set. Skipping code signing."
        return 0
    fi
    
    # Sign all frameworks and helpers first
    find "$app_path/Contents/Frameworks" -type f -name "*.dylib" -exec \
        codesign --force --options runtime --sign "$signing_identity" \
        --entitlements "$child_entitlements" {} \;
    
    find "$app_path/Contents/Frameworks" -type d -name "*.framework" -exec \
        codesign --force --options runtime --sign "$signing_identity" \
        --entitlements "$child_entitlements" {} \;
    
    # Sign helper apps
    find "$app_path/Contents/MacOS" -type f ! -name "$APP_NAME" -exec \
        codesign --force --options runtime --sign "$signing_identity" \
        --entitlements "$child_entitlements" {} \;
    
    # Sign the main app
    codesign --force --options runtime --sign "$signing_identity" \
        --entitlements "$entitlements" "$app_path"
    
    # Verify signature
    codesign --verify --deep --strict --verbose=2 "$app_path"
    
    log_info "Application signed successfully"
}

# Create DMG
create_dmg() {
    log_info "Creating DMG..."
    
    local app_path="$PROJECT_ROOT/target/universal-apple-darwin/release/bundle/macos/$APP_NAME.app"
    local dmg_path="$PROJECT_ROOT/target/universal-apple-darwin/release/bundle/dmg"
    local dmg_name="CloudDevBox_${VERSION}_universal.dmg"
    local temp_dmg="$dmg_path/temp.dmg"
    local final_dmg="$dmg_path/$dmg_name"
    
    mkdir -p "$dmg_path"
    
    # Create temporary DMG
    hdiutil create -srcfolder "$app_path" -volname "$APP_NAME" \
        -fs HFS+ -fsargs "-c c=64,a=16,e=16" -format UDRW "$temp_dmg"
    
    # Mount the DMG
    local device=$(hdiutil attach -readwrite -noverify "$temp_dmg" | \
        egrep '^/dev/' | sed 1q | awk '{print $1}')
    
    # Wait for mount
    sleep 2
    
    # Set up DMG appearance
    local volume_path="/Volumes/$APP_NAME"
    
    # Create Applications symlink
    ln -s /Applications "$volume_path/Applications"
    
    # Set background image if exists
    local background_image="$PROJECT_ROOT/dmg-background.png"
    if [ -f "$background_image" ]; then
        mkdir -p "$volume_path/.background"
        cp "$background_image" "$volume_path/.background/background.png"
        
        # Set DMG window appearance using AppleScript
        osascript <<EOF
tell application "Finder"
    tell disk "$APP_NAME"
        open
        set current view of container window to icon view
        set toolbar visible of container window to false
        set statusbar visible of container window to false
        set bounds of container window to {400, 100, 1060, 500}
        set viewOptions to the icon view options of container window
        set arrangement of viewOptions to not arranged
        set icon size of viewOptions to 128
        set background picture of viewOptions to file ".background:background.png"
        set position of item "$APP_NAME.app" of container window to {180, 170}
        set position of item "Applications" of container window to {480, 170}
        close
        open
        update without registering applications
        delay 2
    end tell
end tell
EOF
    fi
    
    # Unmount
    sync
    hdiutil detach "$device"
    
    # Convert to compressed DMG
    hdiutil convert "$temp_dmg" -format UDZO -imagekey zlib-level=9 -o "$final_dmg"
    
    # Clean up
    rm -f "$temp_dmg"
    
    log_info "DMG created: $final_dmg"
}

# Notarize the DMG
notarize_dmg() {
    log_info "Notarizing DMG..."
    
    local dmg_path="$PROJECT_ROOT/target/universal-apple-darwin/release/bundle/dmg/CloudDevBox_${VERSION}_universal.dmg"
    local apple_id="${APPLE_ID:-}"
    local team_id="${APPLE_TEAM_ID:-}"
    local app_password="${APPLE_APP_PASSWORD:-}"
    
    if [ -z "$apple_id" ] || [ -z "$team_id" ] || [ -z "$app_password" ]; then
        log_warn "Apple notarization credentials not set. Skipping notarization."
        log_warn "Set APPLE_ID, APPLE_TEAM_ID, and APPLE_APP_PASSWORD to enable notarization."
        return 0
    fi
    
    # Submit for notarization
    log_info "Submitting for notarization..."
    xcrun notarytool submit "$dmg_path" \
        --apple-id "$apple_id" \
        --team-id "$team_id" \
        --password "$app_password" \
        --wait
    
    # Staple the notarization ticket
    log_info "Stapling notarization ticket..."
    xcrun stapler staple "$dmg_path"
    
    # Verify
    xcrun stapler validate "$dmg_path"
    
    log_info "Notarization completed successfully"
}

# Main build process
main() {
    local skip_frontend=false
    local skip_sign=false
    local skip_notarize=false
    local arch="universal"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --skip-frontend)
                skip_frontend=true
                shift
                ;;
            --skip-sign)
                skip_sign=true
                shift
                ;;
            --skip-notarize)
                skip_notarize=true
                shift
                ;;
            --arch)
                arch="$2"
                shift 2
                ;;
            --version)
                VERSION="$2"
                shift 2
                ;;
            *)
                log_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    log_info "Starting macOS build process"
    log_info "Version: $VERSION, Architecture: $arch"
    
    # Check prerequisites
    check_prerequisites
    
    # Build frontend
    if [ "$skip_frontend" = false ]; then
        build_frontend
    fi
    
    # Build for target architecture(s)
    case $arch in
        universal)
            build_arch "aarch64-apple-darwin"
            build_arch "x86_64-apple-darwin"
            create_universal_binary
            ;;
        arm64)
            build_arch "aarch64-apple-darwin"
            ;;
        x86_64)
            build_arch "x86_64-apple-darwin"
            ;;
        *)
            log_error "Unknown architecture: $arch"
            exit 1
            ;;
    esac
    
    # Sign the application
    if [ "$skip_sign" = false ]; then
        sign_app
    fi
    
    # Create DMG
    create_dmg
    
    # Notarize
    if [ "$skip_notarize" = false ]; then
        notarize_dmg
    fi
    
    log_info "macOS build completed successfully!"
    
    # List artifacts
    log_info "Build artifacts:"
    find "$PROJECT_ROOT/target" -name "*.dmg" -exec ls -lh {} \;
}

main "$@"
