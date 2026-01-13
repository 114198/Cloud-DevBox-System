# Cloud DevBox Desktop Packaging Guide

This document describes how to build and package the Cloud DevBox desktop application for Windows, macOS, and Linux.

## Prerequisites

### All Platforms
- [Rust](https://rustup.rs/) (stable)
- [Node.js](https://nodejs.org/) (v18+)
- [Tauri CLI](https://tauri.app/): `cargo install tauri-cli`

### Windows
- Visual Studio Build Tools 2019+ with C++ workload
- [WiX Toolset](https://wixtoolset.org/) v3.11+ (for MSI)
- [NSIS](https://nsis.sourceforge.io/) v3.08+ (for NSIS installer)
- Windows SDK (for code signing)

### macOS
- Xcode Command Line Tools: `xcode-select --install`
- Apple Developer account (for code signing and notarization)

### Linux
- WebKit2GTK 4.1: `libwebkit2gtk-4.1-dev` (Ubuntu) or `webkit2gtk4.1-devel` (Fedora)
- GTK 3: `libgtk-3-dev` (Ubuntu) or `gtk3-devel` (Fedora)
- AppIndicator: `libayatana-appindicator3-dev` (Ubuntu) or `libappindicator-gtk3-devel` (Fedora)
- RPM tools (optional): `rpm` (Ubuntu) or `rpm-build` (Fedora)

## Building

### Quick Build (All Platforms)

```bash
# Build for current platform with all bundle types
cd apps/desktop
cargo tauri build
```

### Windows

```powershell
# Build MSI installer
.\scripts\build-windows.ps1 -Target msi

# Build NSIS installer
.\scripts\build-windows.ps1 -Target nsis

# Build both with code signing
.\scripts\build-windows.ps1 -Target all -Sign -Release
```

#### Code Signing (Windows)

Set environment variables:
```powershell
$env:WINDOWS_CERTIFICATE_PATH = "path/to/certificate.pfx"
$env:WINDOWS_CERTIFICATE_PASSWORD = "your-password"
# OR use certificate thumbprint from Windows Certificate Store
$env:WINDOWS_CERTIFICATE_THUMBPRINT = "your-thumbprint"
```

### macOS

```bash
# Build Universal Binary (Intel + Apple Silicon)
./scripts/build-macos.sh

# Build for specific architecture
./scripts/build-macos.sh --arch arm64
./scripts/build-macos.sh --arch x86_64

# Skip notarization (for testing)
./scripts/build-macos.sh --skip-notarize
```

#### Code Signing and Notarization (macOS)

Set environment variables:
```bash
export APPLE_SIGNING_IDENTITY="Developer ID Application: Your Name (TEAM_ID)"
export APPLE_ID="your@email.com"
export APPLE_TEAM_ID="TEAM_ID"
export APPLE_APP_PASSWORD="app-specific-password"
```

### Linux

```bash
# Build all formats (AppImage, DEB, RPM)
./scripts/build-linux.sh

# Build specific format
./scripts/build-linux.sh --target appimage
./scripts/build-linux.sh --target deb
./scripts/build-linux.sh --target rpm
```

## Output Locations

After building, packages are located in:

```
apps/desktop/target/release/bundle/
├── msi/                    # Windows MSI installer
├── nsis/                   # Windows NSIS installer
├── dmg/                    # macOS DMG
├── macos/                  # macOS .app bundle
├── appimage/               # Linux AppImage
├── deb/                    # Debian/Ubuntu package
└── rpm/                    # Fedora/RHEL package
```

## Package Details

### Windows Packages

| Format | Description | Use Case |
|--------|-------------|----------|
| MSI | Windows Installer | Enterprise deployment, Group Policy |
| NSIS | Nullsoft Installer | Consumer distribution, smaller size |

Features:
- Multi-language support (English, Chinese)
- Start Menu and Desktop shortcuts
- File association (.devbox)
- Protocol handler (devbox://)
- Uninstaller

### macOS Package

| Format | Description | Use Case |
|--------|-------------|----------|
| DMG | Disk Image | Standard macOS distribution |

Features:
- Universal Binary (Intel + Apple Silicon)
- Code signed and notarized
- Drag-and-drop installation
- File association and protocol handler
- Retina display support

### Linux Packages

| Format | Description | Use Case |
|--------|-------------|----------|
| AppImage | Portable application | Universal Linux distribution |
| DEB | Debian package | Ubuntu, Debian, Linux Mint |
| RPM | Red Hat package | Fedora, RHEL, CentOS |

Features:
- Desktop integration
- MIME type registration
- AppStream metadata
- System tray support

## CI/CD

The GitHub Actions workflow (`.github/workflows/build-release.yml`) automatically:

1. Builds for all platforms on tag push
2. Signs Windows and macOS builds
3. Notarizes macOS builds
4. Creates GitHub Release with all artifacts
5. Generates checksums

### Required Secrets

| Secret | Platform | Description |
|--------|----------|-------------|
| `WINDOWS_CERTIFICATE` | Windows | Base64-encoded PFX certificate |
| `WINDOWS_CERTIFICATE_PASSWORD` | Windows | Certificate password |
| `APPLE_CERTIFICATE` | macOS | Base64-encoded P12 certificate |
| `APPLE_CERTIFICATE_PASSWORD` | macOS | Certificate password |
| `APPLE_SIGNING_IDENTITY` | macOS | Signing identity name |
| `APPLE_ID` | macOS | Apple ID email |
| `APPLE_TEAM_ID` | macOS | Apple Developer Team ID |
| `APPLE_APP_PASSWORD` | macOS | App-specific password |
| `TAURI_SIGNING_PRIVATE_KEY` | All | Tauri update signing key |
| `TAURI_SIGNING_PRIVATE_KEY_PASSWORD` | All | Signing key password |

## Troubleshooting

### Windows

**Error: SignTool not found**
- Install Windows SDK from Visual Studio Installer

**Error: WiX not found**
- Install WiX Toolset and add to PATH

### macOS

**Error: Code signing failed**
- Ensure certificate is in Keychain
- Check signing identity name matches exactly

**Error: Notarization failed**
- Verify Apple ID and app-specific password
- Check entitlements are correct

### Linux

**Error: webkit2gtk not found**
- Install `libwebkit2gtk-4.1-dev` (Ubuntu) or `webkit2gtk4.1-devel` (Fedora)

**Error: AppImage creation failed**
- Ensure `patchelf` is installed

## Version Management

Update version in:
1. `apps/desktop/Cargo.toml`
2. `apps/desktop/tauri.conf.json`
3. `apps/desktop/linux/cloud-devbox.metainfo.xml` (add release entry)

## Distribution

### Windows
- Microsoft Store (future)
- Direct download from website
- Chocolatey package (future)
- Winget package (future)

### macOS
- Mac App Store (future)
- Direct download from website
- Homebrew Cask (future)

### Linux
- Direct download (AppImage)
- APT repository (future)
- DNF/YUM repository (future)
- Flatpak (future)
- Snap (future)
