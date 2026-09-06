# IAPro Nexus — Platform Support Matrix

Esta matriz documenta o suporte oficial verificado para as duas superfícies equivalentes (Web e Desktop) em todas as plataformas-alvo.

| Platform | Architecture | Core / CLI | Web Surface | Desktop Shell | Terminal Subsystem | IPC / Sockets | Security / Credentials | Installer | Updater | Native CI | Status |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Linux (Ubuntu 24.04+)** | amd64 (x86_64) | VERIFIED | VERIFIED | VERIFIED (Wails/WebKitGTK) | Unix PTY | Unix Domain Socket | Secret Service / Pass | install.sh, DEB, RPM | Update Service (Ed25519) | VERIFIED | **VERIFIED** |
| **Linux (Ubuntu 24.04+)** | arm64 (aarch64)| VERIFIED | VERIFIED | VERIFIED (Wails/WebKitGTK) | Unix PTY | Unix Domain Socket | Secret Service / Pass | install.sh, DEB, RPM | Update Service (Ed25519) | VERIFIED | **VERIFIED** |
| **macOS (12+)** | amd64 | VERIFIED | VERIFIED | VERIFIED (Wails/WKWebView) | Unix PTY | Unix Domain Socket | macOS Keychain | install.sh, .app, DMG | Update Service (Ed25519) | VERIFIED | **VERIFIED** |
| **macOS (Apple Silicon)**| arm64 | VERIFIED | VERIFIED | VERIFIED (Wails/WKWebView) | Unix PTY | Unix Domain Socket | macOS Keychain | install.sh, .app, DMG | Update Service (Ed25519) | VERIFIED | **VERIFIED** |
| **Windows (10/11)** | amd64 (x86_64) | VERIFIED | VERIFIED | VERIFIED (Wails/WebView2)  | Windows ConPTY | Named Pipes | Credential Manager | install.ps1, NSIS | Update Service (Ed25519) | VERIFIED | **VERIFIED** |
| **Windows (10/11)** | arm64 | VERIFIED | VERIFIED | VERIFIED (Wails/WebView2)  | Windows ConPTY | Named Pipes | Credential Manager | install.ps1, NSIS | Update Service (Ed25519) | VERIFIED | **VERIFIED** |

## Estados do Maestro por Plataforma
- **NOT INSTALLED / DEGRADED**: Nexus opera integralmente nos modos Direto, Terminal e Workspaces.
- **COMPATIBLE**: Orquestração e sugestões de skills habilitadas automaticamente.
- **OUTDATED / INCOMPATIBLE**: Avisos amigáveis em Settings > Updates e Doctor sem travar o Nexus.
