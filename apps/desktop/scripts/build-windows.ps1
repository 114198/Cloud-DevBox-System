# Cloud DevBox Windows Build Script
# Builds MSI and NSIS installers for Windows

param(
    [Parameter(Mandatory=$false)]
    [ValidateSet("msi", "nsis", "all")]
    [string]$Target = "all",
    
    [Parameter(Mandatory=$false)]
    [switch]$Release = $false,
    
    [Parameter(Mandatory=$false)]
    [switch]$Sign = $false,
    
    [Parameter(Mandatory=$false)]
    [string]$Version = "0.1.0"
)

$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent $ScriptDir

function Write-Log {
    param([string]$Message, [string]$Level = "INFO")
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $color = switch ($Level) {
        "INFO" { "White" }
        "SUCCESS" { "Green" }
        "WARNING" { "Yellow" }
        "ERROR" { "Red" }
        default { "White" }
    }
    Write-Host "[$timestamp] [$Level] $Message" -ForegroundColor $color
}

function Test-Prerequisites {
    Write-Log "Checking prerequisites..."
    
    # Check Rust
    if (-not (Get-Command cargo -ErrorAction SilentlyContinue)) {
        throw "Rust/Cargo not found. Please install Rust from https://rustup.rs"
    }
    
    # Check Node.js
    if (-not (Get-Command node -ErrorAction SilentlyContinue)) {
        throw "Node.js not found. Please install Node.js from https://nodejs.org"
    }
    
    # Check Tauri CLI
    $tauriCli = cargo install --list | Select-String "tauri-cli"
    if (-not $tauriCli) {
        Write-Log "Installing Tauri CLI..." "WARNING"
        cargo install tauri-cli
    }
    
    Write-Log "Prerequisites check passed" "SUCCESS"
}

function Build-Frontend {
    Write-Log "Building frontend..."
    
    Push-Location "$ProjectRoot\..\web"
    try {
        npm install
        npm run build
    }
    finally {
        Pop-Location
    }
    
    Write-Log "Frontend build completed" "SUCCESS"
}

function Build-TauriApp {
    param([string]$BuildTarget)
    
    Write-Log "Building Tauri application for $BuildTarget..."
    
    Push-Location $ProjectRoot
    try {
        $buildArgs = @("tauri", "build")
        
        if ($Release) {
            $buildArgs += "--release"
        }
        
        switch ($BuildTarget) {
            "msi" {
                $buildArgs += @("--bundles", "msi")
            }
            "nsis" {
                $buildArgs += @("--bundles", "nsis")
            }
            "all" {
                $buildArgs += @("--bundles", "msi,nsis")
            }
        }
        
        # Set version
        $env:TAURI_SIGNING_PRIVATE_KEY = $env:TAURI_SIGNING_PRIVATE_KEY
        
        & cargo $buildArgs
        
        if ($LASTEXITCODE -ne 0) {
            throw "Tauri build failed"
        }
    }
    finally {
        Pop-Location
    }
    
    Write-Log "Tauri build completed" "SUCCESS"
}

function Sign-Artifacts {
    Write-Log "Signing build artifacts..."
    
    $bundleDir = "$ProjectRoot\target\release\bundle"
    
    # Sign MSI
    $msiFiles = Get-ChildItem -Path "$bundleDir\msi" -Filter "*.msi" -ErrorAction SilentlyContinue
    foreach ($msi in $msiFiles) {
        & "$ScriptDir\sign-windows.ps1" -FilePath $msi.FullName
    }
    
    # Sign NSIS installer
    $nsisFiles = Get-ChildItem -Path "$bundleDir\nsis" -Filter "*.exe" -ErrorAction SilentlyContinue
    foreach ($nsis in $nsisFiles) {
        & "$ScriptDir\sign-windows.ps1" -FilePath $nsis.FullName
    }
    
    # Sign main executable
    $exeFile = "$ProjectRoot\target\release\Cloud DevBox.exe"
    if (Test-Path $exeFile) {
        & "$ScriptDir\sign-windows.ps1" -FilePath $exeFile
    }
    
    Write-Log "Signing completed" "SUCCESS"
}

function Get-BuildArtifacts {
    $bundleDir = "$ProjectRoot\target\release\bundle"
    $artifacts = @()
    
    # MSI files
    $msiFiles = Get-ChildItem -Path "$bundleDir\msi" -Filter "*.msi" -ErrorAction SilentlyContinue
    $artifacts += $msiFiles
    
    # NSIS files
    $nsisFiles = Get-ChildItem -Path "$bundleDir\nsis" -Filter "*.exe" -ErrorAction SilentlyContinue
    $artifacts += $nsisFiles
    
    return $artifacts
}

# Main execution
try {
    Write-Log "Starting Windows build process"
    Write-Log "Target: $Target, Release: $Release, Sign: $Sign"
    
    # Check prerequisites
    Test-Prerequisites
    
    # Build frontend
    Build-Frontend
    
    # Build Tauri app
    Build-TauriApp -BuildTarget $Target
    
    # Sign if requested
    if ($Sign) {
        Sign-Artifacts
    }
    
    # List artifacts
    Write-Log "Build artifacts:"
    $artifacts = Get-BuildArtifacts
    foreach ($artifact in $artifacts) {
        $size = [math]::Round($artifact.Length / 1MB, 2)
        Write-Log "  - $($artifact.Name) ($size MB)" "SUCCESS"
    }
    
    Write-Log "Windows build completed successfully!" "SUCCESS"
    exit 0
}
catch {
    Write-Log "Build failed: $_" "ERROR"
    exit 1
}
