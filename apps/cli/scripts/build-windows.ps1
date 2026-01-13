# Windows build script for DevBox CLI
# Builds Windows binaries (x64 and ARM64)

$ErrorActionPreference = "Stop"

# Get version from Cargo.toml
$cargoToml = Get-Content "..\..\Cargo.toml" -Raw
if ($cargoToml -match 'version\s*=\s*"([^"]+)"') {
    $VERSION = $matches[1]
} else {
    $VERSION = "0.1.0"
}

$OUTPUT_DIR = "dist"
$BINARY_NAME = "devbox"

Write-Host "Building DevBox CLI v$VERSION for Windows" -ForegroundColor Cyan
Write-Host "=========================================="

# Create output directory
New-Item -ItemType Directory -Force -Path $OUTPUT_DIR | Out-Null

# Define Windows targets
$TARGETS = @(
    "x86_64-pc-windows-msvc",
    "aarch64-pc-windows-msvc"
)

foreach ($target in $TARGETS) {
    Write-Host ""
    Write-Host "Building for $target..." -ForegroundColor Yellow
    
    # Check if target is installed
    $installed = rustup target list --installed | Select-String $target
    if (-not $installed) {
        Write-Host "  Installing target $target..."
        rustup target add $target
        if ($LASTEXITCODE -ne 0) {
            Write-Host "  Warning: Could not install target $target, skipping..." -ForegroundColor Red
            continue
        }
    }
    
    # Build
    cargo build --release --target $target
    if ($LASTEXITCODE -ne 0) {
        Write-Host "  Warning: Build failed for $target, skipping..." -ForegroundColor Red
        continue
    }
    
    $srcBinary = "..\..\target\$target\release\$BINARY_NAME.exe"
    $dstBinary = "$OUTPUT_DIR\$BINARY_NAME-$VERSION-$target.exe"
    
    if (Test-Path $srcBinary) {
        Copy-Item $srcBinary $dstBinary
        Write-Host "  Created: $dstBinary" -ForegroundColor Green
        
        # Create ZIP archive
        $zipFile = "$OUTPUT_DIR\$BINARY_NAME-$VERSION-$target.zip"
        Compress-Archive -Path $dstBinary -DestinationPath $zipFile -Force
        Write-Host "  Created: $zipFile" -ForegroundColor Green
    }
}

Write-Host ""
Write-Host "Build complete! Binaries are in $OUTPUT_DIR\" -ForegroundColor Cyan
Get-ChildItem $OUTPUT_DIR
