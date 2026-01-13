# Cloud DevBox Windows Code Signing Script
# This script signs Windows executables and installers using a code signing certificate

param(
    [Parameter(Mandatory=$false)]
    [string]$CertificatePath = $env:WINDOWS_CERTIFICATE_PATH,
    
    [Parameter(Mandatory=$false)]
    [string]$CertificatePassword = $env:WINDOWS_CERTIFICATE_PASSWORD,
    
    [Parameter(Mandatory=$false)]
    [string]$CertificateThumbprint = $env:WINDOWS_CERTIFICATE_THUMBPRINT,
    
    [Parameter(Mandatory=$false)]
    [string]$TimestampServer = "http://timestamp.digicert.com",
    
    [Parameter(Mandatory=$true)]
    [string]$FilePath,
    
    [Parameter(Mandatory=$false)]
    [switch]$DualSign = $false
)

$ErrorActionPreference = "Stop"

function Write-Log {
    param([string]$Message, [string]$Level = "INFO")
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    Write-Host "[$timestamp] [$Level] $Message"
}

function Test-SignTool {
    $signToolPaths = @(
        "C:\Program Files (x86)\Windows Kits\10\bin\10.0.22621.0\x64\signtool.exe",
        "C:\Program Files (x86)\Windows Kits\10\bin\10.0.19041.0\x64\signtool.exe",
        "C:\Program Files (x86)\Windows Kits\10\bin\x64\signtool.exe",
        "C:\Program Files (x86)\Microsoft SDKs\ClickOnce\SignTool\signtool.exe"
    )
    
    foreach ($path in $signToolPaths) {
        if (Test-Path $path) {
            return $path
        }
    }
    
    # Try to find in PATH
    $signTool = Get-Command signtool.exe -ErrorAction SilentlyContinue
    if ($signTool) {
        return $signTool.Source
    }
    
    throw "SignTool not found. Please install Windows SDK."
}

function Sign-File {
    param(
        [string]$SignToolPath,
        [string]$File,
        [string]$DigestAlgorithm = "sha256"
    )
    
    Write-Log "Signing file: $File with $DigestAlgorithm"
    
    $arguments = @("sign", "/fd", $DigestAlgorithm, "/tr", $TimestampServer, "/td", $DigestAlgorithm)
    
    if ($CertificateThumbprint) {
        # Sign using certificate from Windows Certificate Store
        $arguments += @("/sha1", $CertificateThumbprint)
    }
    elseif ($CertificatePath -and $CertificatePassword) {
        # Sign using PFX file
        $arguments += @("/f", $CertificatePath, "/p", $CertificatePassword)
    }
    else {
        throw "No certificate specified. Provide either CertificateThumbprint or CertificatePath with CertificatePassword."
    }
    
    $arguments += $File
    
    $process = Start-Process -FilePath $SignToolPath -ArgumentList $arguments -Wait -PassThru -NoNewWindow
    
    if ($process.ExitCode -ne 0) {
        throw "Failed to sign file: $File (Exit code: $($process.ExitCode))"
    }
    
    Write-Log "Successfully signed: $File" "SUCCESS"
}

function Verify-Signature {
    param(
        [string]$SignToolPath,
        [string]$File
    )
    
    Write-Log "Verifying signature: $File"
    
    $process = Start-Process -FilePath $SignToolPath -ArgumentList @("verify", "/pa", "/v", $File) -Wait -PassThru -NoNewWindow
    
    if ($process.ExitCode -ne 0) {
        throw "Signature verification failed for: $File"
    }
    
    Write-Log "Signature verified: $File" "SUCCESS"
}

# Main execution
try {
    Write-Log "Starting Windows code signing process"
    
    # Validate file exists
    if (-not (Test-Path $FilePath)) {
        throw "File not found: $FilePath"
    }
    
    # Find SignTool
    $signToolPath = Test-SignTool
    Write-Log "Using SignTool: $signToolPath"
    
    # Dual signing (SHA1 + SHA256) for compatibility with older Windows versions
    if ($DualSign) {
        Write-Log "Performing dual signing (SHA1 + SHA256)"
        
        # First sign with SHA1 (for Windows 7 compatibility)
        Sign-File -SignToolPath $signToolPath -File $FilePath -DigestAlgorithm "sha1"
        
        # Append SHA256 signature
        $arguments = @("sign", "/as", "/fd", "sha256", "/tr", $TimestampServer, "/td", "sha256")
        
        if ($CertificateThumbprint) {
            $arguments += @("/sha1", $CertificateThumbprint)
        }
        elseif ($CertificatePath -and $CertificatePassword) {
            $arguments += @("/f", $CertificatePath, "/p", $CertificatePassword)
        }
        
        $arguments += $FilePath
        
        $process = Start-Process -FilePath $signToolPath -ArgumentList $arguments -Wait -PassThru -NoNewWindow
        
        if ($process.ExitCode -ne 0) {
            throw "Failed to append SHA256 signature"
        }
    }
    else {
        # Single SHA256 signature (recommended for modern Windows)
        Sign-File -SignToolPath $signToolPath -File $FilePath -DigestAlgorithm "sha256"
    }
    
    # Verify signature
    Verify-Signature -SignToolPath $signToolPath -File $FilePath
    
    Write-Log "Code signing completed successfully" "SUCCESS"
    exit 0
}
catch {
    Write-Log "Error: $_" "ERROR"
    exit 1
}
