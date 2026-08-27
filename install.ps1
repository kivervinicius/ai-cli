# AI CLI Installer for PowerShell (Windows & PowerShell Core)
# Requires Go 1.22+ to build from source

$ErrorActionPreference = 'Stop'

Write-Host "=== AI CLI Installer for PowerShell ===" -ForegroundColor Cyan

# 1. Check Go compiler
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Error "Go (>= 1.22) is required to build from source. Please install Go from https://golang.org"
    exit 1
}

# 2. Determine binary name and target directory
$IsWindowsOS = ($IsWindows -or ($env:OS -like "*Windows*"))
$BinaryName = if ($IsWindowsOS) { "ai.exe" } else { "ai" }

Write-Host "Building $BinaryName from source..." -ForegroundColor Yellow
go build -ldflags="-s -w" -o $BinaryName ./cmd/ai
if ($LASTEXITCODE -ne 0) {
    Write-Error "Build failed with exit code $LASTEXITCODE."
    exit 1
}

# Target directory selection
if ($IsWindowsOS) {
    $TargetDir = Join-Path $env:LOCALAPPDATA "Programs\ai-cli"
} else {
    $TargetDir = Join-Path $HOME ".local/bin"
}

if (-not (Test-Path $TargetDir)) {
    New-Item -ItemType Directory -Path $TargetDir -Force | Out-Null
}

$TargetPath = Join-Path $TargetDir $BinaryName
Copy-Item -Path $BinaryName -Destination $TargetPath -Force

Write-Host "✓ Installed $BinaryName to $TargetPath" -ForegroundColor Green

# 3. Ensure TargetDir is in User PATH
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
$PathEntries = $UserPath -split ';' | Where-Object { $_ -ne "" }

if ($PathEntries -notcontains $TargetDir) {
    Write-Host "Adding $TargetDir to user PATH environment variable..." -ForegroundColor Yellow
    $NewPath = if ([string]::IsNullOrEmpty($UserPath)) { $TargetDir } else { "$UserPath;$TargetDir" }
    [Environment]::SetEnvironmentVariable("Path", $NewPath, "User")
    $env:PATH = "$TargetDir;$env:PATH"
    Write-Host "✓ Added $TargetDir to PATH." -ForegroundColor Green
}

Write-Host "`nSetup complete!" -ForegroundColor Green
Write-Host "Run 'ai doctor' to verify provider dependencies." -ForegroundColor White
Write-Host "To enable shell completion in PowerShell, add this to your `$PROFILE:" -ForegroundColor Gray
Write-Host "  ai completion powershell | Out-String | Invoke-Expression" -ForegroundColor Cyan
