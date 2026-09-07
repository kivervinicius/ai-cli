# IAPro Nexus Installer for PowerShell (Windows & PowerShell Core)
# Usage after download: .\install.ps1 -Version v0.5.0-beta.23

param(
    [switch]$WithMaestro = $false,
    [switch]$NoDesktop = $false,
    [string]$Version = $env:NEXUS_VERSION,
    [switch]$BuildFromSource = $false,
    [string]$SourceRef = $env:NEXUS_SOURCE_REF
)

$ErrorActionPreference = 'Stop'
$InstallDesktop = -not $NoDesktop

function Ensure-GoCompiler {
    if (Get-Command go -ErrorAction SilentlyContinue) {
        return
    }

    if (-not $IsWindowsOS) {
        throw "Go >=1.25 is required for source builds. Install Go manually and run the installer again."
    }

    $Winget = Get-Command winget -ErrorAction SilentlyContinue
    if (-not $Winget) {
        throw "Go >=1.25 is required for source builds, but winget is unavailable. Install Go from https://go.dev/dl/ and run the installer again."
    }

    Write-Host "Go was not found. Installing the official Go package with WinGet..." -ForegroundColor Yellow
    & $Winget.Source install --id GoLang.Go --exact --accept-source-agreements --accept-package-agreements
    if ($LASTEXITCODE -ne 0) {
        throw "WinGet could not install Go. Install Go from https://go.dev/dl/ and run the installer again."
    }

    # WinGet updates PATH for future processes only. Refresh this process so
    # the source build can run immediately after the installation.
    $MachinePath = [Environment]::GetEnvironmentVariable("Path", "Machine")
    $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $env:Path = "$UserPath;$MachinePath"
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        throw "Go was installed but is not available in PATH yet. Restart PowerShell and run the installer again."
    }
    Write-Host "✓ Go compiler is available." -ForegroundColor Green
}

$Repo = "kivervinicius/ai-cli"
$GithubUrl = "https://github.com/$Repo"

Write-Host "=== IAPro Nexus Installer (Pinned Release) ===" -ForegroundColor Cyan

if ([string]::IsNullOrWhiteSpace($Version) -and -not $BuildFromSource) {
    throw "A version is required for verified installation. Use -Version vX.Y.Z or -BuildFromSource from a checkout."
}
if (-not [string]::IsNullOrWhiteSpace($Version)) {
    if ($Version -notmatch '^v?[0-9]+\.[0-9]+\.[0-9]+([.-][A-Za-z0-9.-]+)?$') {
        throw "Invalid Nexus version: $Version"
    }
    $Version = $Version.TrimStart('v')
}

# 1. Determine platform and target directory
$IsWindowsOS = ($IsWindows -or ($env:OS -like "*Windows*"))
$BinaryName = if ($IsWindowsOS) { "nexus.exe" } else { "nexus" }
$AiAliasName = if ($IsWindowsOS) { "ai.exe" } else { "ai" }

if ($IsWindowsOS) {
    $TargetDir = Join-Path $env:LOCALAPPDATA "Programs\IAPro Nexus"
    $LegacyTargetDir = Join-Path $env:LOCALAPPDATA "Programs\ai-cli"
} else {
    $TargetDir = Join-Path $HOME ".local/bin"
    $LegacyTargetDir = ""
}

if (-not (Test-Path $TargetDir)) {
    New-Item -ItemType Directory -Path $TargetDir -Force | Out-Null
}

if ($IsWindowsOS -and (Test-Path $LegacyTargetDir)) {
    Write-Host "Legacy ai-cli installation found at $LegacyTargetDir; leaving it untouched while installing IAPro Nexus to $TargetDir." -ForegroundColor Gray
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
$DownloadUrl = "$GithubUrl/releases/download/v$Version/$ArchiveName"
$ChecksumsUrl = "$GithubUrl/releases/download/v$Version/checksums.txt"

$TempDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $TempDir -Force | Out-Null

try {
    Write-Host "Attempting to download Nexus v${Version}: $ArchiveName..." -ForegroundColor Yellow
    $ZipPath = Join-Path $TempDir $ArchiveName
    if (-not [string]::IsNullOrWhiteSpace($Version)) {
        Invoke-WebRequest -Uri $DownloadUrl -OutFile $ZipPath -UseBasicParsing
        $ChecksumsPath = Join-Path $TempDir "checksums.txt"
        Invoke-WebRequest -Uri $ChecksumsUrl -OutFile $ChecksumsPath -UseBasicParsing
        $Expected = ((Get-Content $ChecksumsPath | Where-Object { $_ -match [regex]::Escape($ArchiveName) } | Select-Object -First 1) -split '\s+')[0]
        $Actual = (Get-FileHash -Path $ZipPath -Algorithm SHA256).Hash.ToLowerInvariant()
        if ([string]::IsNullOrWhiteSpace($Expected) -or $Expected.ToLowerInvariant() -ne $Actual) {
            throw "Release checksum verification failed for $ArchiveName."
        }
    }
    
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
    if (-not $BuildFromSource) {
        throw
    }
    Write-Host "Verified release unavailable; explicit source build requested." -ForegroundColor Yellow
}

# 3. Explicit source build only; never resolve a mutable latest ref implicitly.
if (-not $Installed -and $BuildFromSource) {
    Ensure-GoCompiler
    Write-Host "Building from source via Go..." -ForegroundColor Yellow
    if ((Test-Path "./go.mod") -and (Test-Path "./cmd/nexus")) {
        go build -ldflags="-s -w" -o $TargetPath ./cmd/nexus
        $Installed = $true
    } elseif (-not [string]::IsNullOrWhiteSpace($SourceRef)) {
        if ($SourceRef -notmatch '^[A-Za-z0-9._/-]+$') { throw "Invalid source ref." }
        Write-Host "Cloning explicitly requested source ref $SourceRef..." -ForegroundColor Yellow
        $CloneDir = Join-Path $TempDir "repo"
        git clone --depth 1 --branch $SourceRef "$GithubUrl.git" $CloneDir
        Push-Location $CloneDir
        try {
            go build -ldflags="-s -w" -o $TargetPath ./cmd/nexus
            $Installed = $true
        } finally {
            Pop-Location
        }
    } else {
        throw "-BuildFromSource requires a Nexus checkout or -SourceRef=<tag-or-commit>."
    }
}

if (-not $Installed) {
    throw "Installation failed: no verified release artifact was installed."
}

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

# Install the native Wails shell and create a normal Windows Desktop shortcut.
# Older releases may not publish the separate desktop artifact yet; keep the
# verified CLI install successful in that case.
if ($InstallDesktop -and $IsWindowsOS -and -not [string]::IsNullOrWhiteSpace($Version)) {
    $DesktopArchiveName = "nexus-desktop_Windows_${Arch}.zip"
    $DesktopDownloadUrl = "$GithubUrl/releases/download/v$Version/$DesktopArchiveName"
    $DesktopChecksumsUrl = "$GithubUrl/releases/download/v$Version/desktop-checksums.txt"
    $DesktopZipPath = Join-Path $TempDir $DesktopArchiveName
    $DesktopChecksumsPath = Join-Path $TempDir "desktop-checksums.txt"
    try {
        Write-Host "Attempting to install the native IAPro Nexus Desktop shell..." -ForegroundColor Yellow
        Invoke-WebRequest -Uri $DesktopDownloadUrl -OutFile $DesktopZipPath -UseBasicParsing
        Invoke-WebRequest -Uri $DesktopChecksumsUrl -OutFile $DesktopChecksumsPath -UseBasicParsing
        $ExpectedDesktop = ((Get-Content $DesktopChecksumsPath | Where-Object { $_ -match [regex]::Escape($DesktopArchiveName) } | Select-Object -First 1) -split '\s+')[0]
        $ActualDesktop = (Get-FileHash -Path $DesktopZipPath -Algorithm SHA256).Hash.ToLowerInvariant()
        if ([string]::IsNullOrWhiteSpace($ExpectedDesktop) -or $ExpectedDesktop.ToLowerInvariant() -ne $ActualDesktop) {
            throw "Native Desktop checksum verification failed for $DesktopArchiveName."
        }
        $DesktopExtractDir = Join-Path $TempDir "desktop"
        Expand-Archive -Path $DesktopZipPath -DestinationPath $DesktopExtractDir -Force
        $DesktopBinary = Join-Path $DesktopExtractDir "nexus-desktop.exe"
        if (-not (Test-Path $DesktopBinary)) { throw "Native Desktop executable missing from $DesktopArchiveName." }
        $DesktopTargetPath = Join-Path $TargetDir "nexus-desktop.exe"
        Copy-Item -Path $DesktopBinary -Destination $DesktopTargetPath -Force

        $DesktopShortcutPath = Join-Path ([Environment]::GetFolderPath('Desktop')) "IAPro Nexus.lnk"
        $Shell = New-Object -ComObject WScript.Shell
        $Shortcut = $Shell.CreateShortcut($DesktopShortcutPath)
        $Shortcut.TargetPath = $DesktopTargetPath
        $Shortcut.WorkingDirectory = $TargetDir
        $Shortcut.Description = "IAPro Nexus Workspace OS"
        $Shortcut.Save()
        Write-Host "✓ Native Desktop installed to $DesktopTargetPath" -ForegroundColor Green
        Write-Host "✓ Shortcut created at $DesktopShortcutPath" -ForegroundColor Green
    } catch {
        Write-Host "⚠️  Native Desktop artifact unavailable or unverifiable; CLI installation is complete." -ForegroundColor Yellow
    }
}

Remove-Item -Path $TempDir -Recurse -Force -ErrorAction SilentlyContinue

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
