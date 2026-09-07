# IAPro Nexus — Platform Support Matrix

Esta matriz documenta o suporte oficial verificado para as duas superfícies equivalentes (Web e Desktop) em todas as plataformas-alvo.

| Platform | Architecture | Core / CLI | Web Surface | Desktop Shell | Terminal Subsystem | IPC / Sockets | Security / Credentials | Installer | Updater | Native CI | Status |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Linux (Ubuntu 24.04+)** | amd64 (x86_64) | VERIFIED | VERIFIED | UNVERIFIED (native smoke pending) | Unix PTY | Unix Domain Socket | Secret Service / Pass | install.sh, DEB, RPM (trust-chain audit pending) | Ed25519 service; public keyring pending | CORE VERIFIED / DESKTOP UNVERIFIED | **PARTIAL** |
| **Linux (Ubuntu 24.04+)** | arm64 (aarch64)| CROSS-COMPILED | CROSS-COMPILED | UNVERIFIED | Unix PTY | Unix Domain Socket | Secret Service / Pass | install.sh, DEB, RPM (trust-chain audit pending) | Ed25519 service; public keyring pending | CROSS-COMPILED | **UNVERIFIED** |
| **macOS (12+)** | amd64 | UNVERIFIED (historical CI failed) | UNVERIFIED | UNVERIFIED (native smoke pending) | Unix PTY | Unix Domain Socket | macOS Keychain | install.sh, .app, DMG (packaging pending) | Ed25519 service; public keyring pending | FAILED at historical run `34012236345` | **UNVERIFIED** |
| **macOS (Apple Silicon)**| arm64 | UNVERIFIED (historical CI failed) | UNVERIFIED | UNVERIFIED | Unix PTY | Unix Domain Socket | macOS Keychain | install.sh, .app, DMG (packaging pending) | Ed25519 service; public keyring pending | FAILED at historical run `34012236345` | **UNVERIFIED** |
| **Windows (10/11)** | amd64 (x86_64) | UNVERIFIED (historical CI failed) | UNVERIFIED | UNVERIFIED (native smoke pending) | Windows ConPTY | Named Pipes | Credential Manager | install.ps1, NSIS (packaging pending) | Ed25519 service; public keyring pending | FAILED at historical run `34012236345` | **UNVERIFIED** |
| **Windows (10/11)** | arm64 | CROSS-COMPILED | UNVERIFIED | UNVERIFIED | Windows ConPTY | Named Pipes | Credential Manager | install.ps1, NSIS (packaging pending) | Ed25519 service; public keyring pending | CROSS-COMPILED | **UNVERIFIED** |

## Estados do Maestro por Plataforma
- **NOT INSTALLED / DEGRADED**: Nexus opera integralmente nos modos Direto, Terminal e Workspaces.
- **COMPATIBLE**: Orquestração e sugestões de skills habilitadas automaticamente.
- **OUTDATED / INCOMPATIBLE**: Avisos amigáveis em Settings > Updates e Doctor sem travar o Nexus.
