# AI CLI Installer for PowerShell (Windows & PowerShell Core)
# Supports direct zero-clone installation via:
# irm https://raw.githubusercontent.com/kivervinicius/ai-cli/main/install.ps1 | iex

param(
    [switch]$WithMaestro = $false
)

$ErrorActionPreference = 'Stop'

$Repo = "kivervinicius/ai-cli"
$GithubUrl = "https://github.com/$Repo"

Write-Host "=== AI CLI Installer (Zero-Clone for PowerShell) ===" -ForegroundColor Cyan

# 1. Determine platform and target directory
$IsWindowsOS = ($IsWindows -or ($env:OS -like "*Windows*"))
$BinaryName = if ($IsWindowsOS) { "nexus.exe" } else { "nexus" }
$AiAliasName = if ($IsWindowsOS) { "ai.exe" } else { "ai" }

if ($IsWindowsOS) {
    $TargetDir = Join-Path $env:LOCALAPPDATA "Programs\ai-cli"
} else {
    $TargetDir = Join-Path $HOME ".local/bin"
}

if (-not (Test-Path $TargetDir)) {
    New-Item -ItemType Directory -Path $TargetDir -Force | Out-Null
}

$TargetPath = Join-Path $TargetDir $BinaryName
$Installed = $false

# 2. Try downloading pre-built release binary
$Arch = if ([System.Environment]::Is64BitOperatingSystem) {
    if ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture -eq [System.Runtime.InteropServices.Architecture]::Arm64) {
        "arm64"
    } else {
        "x86_64"
    }
} else {
    "i386"
}

$OsName = if ($IsWindowsOS) { "Windows" } elseif ($IsMacOS) { "Darwin" } else { "Linux" }
$ArchiveExt = if ($IsWindowsOS) { "zip" } else { "tar.gz" }
$ArchiveName = "nexus_${OsName}_${Arch}.${ArchiveExt}"
$DownloadUrl = "$GithubUrl/releases/latest/download/$ArchiveName"

$TempDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $TempDir -Force | Out-Null

try {
    Write-Host "Attempting to download latest release: $ArchiveName..." -ForegroundColor Yellow
    $ZipPath = Join-Path $TempDir $ArchiveName
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $ZipPath -UseBasicParsing -ErrorAction SilentlyContinue
    
    if (Test-Path $ZipPath) {
        if ($IsWindowsOS) {
            Expand-Archive -Path $ZipPath -DestinationPath $TempDir -Force
        } else {
            tar -xzf $ZipPath -C $TempDir
        }
        $ExtractedBin = Join-Path $TempDir $BinaryName
        if (-not (Test-Path $ExtractedBin)) {
            $ExtractedBin = Join-Path $TempDir $AiAliasName
        }
        if (Test-Path $ExtractedBin) {
            Copy-Item -Path $ExtractedBin -Destination $TargetPath -Force
            $Installed = $true
        }
    }
} catch {
    # Release binary not reachable, will try source fallback
}

# 3. Fallback: Build from source if Go is installed
if (-not $Installed) {
    if (Get-Command go -ErrorAction SilentlyContinue) {
        Write-Host "Building from source via Go..." -ForegroundColor Yellow
        if (Test-Path "./cmd/nexus/main.go") {
            go build -ldflags="-s -w" -o $TargetPath ./cmd/nexus
            $Installed = $true
        } elseif (Test-Path "./cmd/ai/main.go") {
            go build -ldflags="-s -w" -o $TargetPath ./cmd/ai
            $Installed = $true
        } else {
            Write-Host "Fetching latest source code..." -ForegroundColor Yellow
            $env:GOBIN = $TargetDir
            go install "github.com/$Repo/cmd/nexus@latest" 2>$null
            if ($LASTEXITCODE -eq 0) {
                $Installed = $true
            } else {
                $CloneDir = Join-Path $TempDir "repo"
                git clone --depth 1 "$GithubUrl.git" $CloneDir
                Push-Location $CloneDir
                try {
                    if (Test-Path "./cmd/nexus") {
                        go build -ldflags="-s -w" -o $TargetPath ./cmd/nexus
                    } else {
                        go build -ldflags="-s -w" -o $TargetPath ./cmd/ai
                    }
                    $Installed = $true
                } finally {
                    Pop-Location
                }
            }
        }
    } else {
        Write-Error "Could not download pre-built binary and Go compiler is not installed.`nPlease install Go from https://golang.org or download from $GithubUrl/releases"
        exit 1
    }
}

Remove-Item -Path $TempDir -Recurse -Force -ErrorAction SilentlyContinue

# Create ai alias/copy for backward compatibility
$AiAliasPath = Join-Path $TargetDir $AiAliasName
if (Test-Path $TargetPath) {
    Copy-Item -Path $TargetPath -Destination $AiAliasPath -Force
}

# Check and install Maestro dependency (OPT-IN ONLY)
if ($WithMaestro) {
    Write-Host "`nChecking Orquestrador Maestro dependency (-WithMaestro requested)..." -ForegroundColor Yellow
    if (-not (Get-Command orquestrador-maestro -ErrorAction SilentlyContinue) -and -not (Get-Command maestro -ErrorAction SilentlyContinue)) {
        if (Get-Command npm -ErrorAction SilentlyContinue) {
            Write-Host "Installing Orquestrador Maestro CLI (@iapro/orquestrador-maestro-cli)..." -ForegroundColor Yellow
            npm install -g @iapro/orquestrador-maestro-cli 2>$null
        } else {
            Write-Host "Node.js / npm not detected. Maestro will remain unavailable/degraded." -ForegroundColor Yellow
        }
    }
} else {
    Write-Host "`nMaestro auto-install skipped (Nexus does not silently install third-party packages)." -ForegroundColor Gray
    Write-Host "To install Maestro orchestration capabilities, run with '-WithMaestro' or install manually:" -ForegroundColor Gray
    Write-Host "  npm install -g @iapro/orquestrador-maestro-cli" -ForegroundColor Gray
}

$MaestroCmd = Get-Command orquestrador-maestro -ErrorAction SilentlyContinue
if ($MaestroCmd) {
    $MaestroAliasPath = Join-Path $TargetDir "maestro.cmd"
    Set-Content -Path $MaestroAliasPath -Value "@echo off`r`norquestrador-maestro %*" -Force
    Write-Host "✓ Linked Maestro alias ($MaestroAliasPath)" -ForegroundColor Green
}

Write-Host "✓ Successfully installed IAPro Nexus to $TargetPath" -ForegroundColor Green

# 4. Ensure TargetDir is in User PATH
if ($IsWindowsOS) {
    $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $PathEntries = $UserPath -split ';' | Where-Object { $_ -ne "" }

    if ($PathEntries -notcontains $TargetDir) {
        Write-Host "Adding $TargetDir to user PATH environment variable..." -ForegroundColor Yellow
        $NewPath = if ([string]::IsNullOrEmpty($UserPath)) { $TargetDir } else { "$UserPath;$TargetDir" }
        [Environment]::SetEnvironmentVariable("Path", $NewPath, "User")
        $env:PATH = "$TargetDir;$env:PATH"
        Write-Host "✓ Added $TargetDir to PATH." -ForegroundColor Green
    }
}

Write-Host "`nSetup complete!" -ForegroundColor Green
Write-Host "Run 'nexus doctor' to verify provider and Maestro dependencies." -ForegroundColor White
Write-Host "To start the Workspace OS:" -ForegroundColor Cyan
Write-Host "  nexus web" -ForegroundColor Cyan
