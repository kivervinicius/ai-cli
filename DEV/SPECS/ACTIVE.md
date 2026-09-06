# Especificação ativa: IAPro Nexus — Implementação Completa Web + Desktop Multiplataforma

## Objetivo

Tornar oficialmente o IAPro Nexus uma aplicação com duas superfícies de execução equivalentes:
1. Web (acessada pelo navegador via `nexus web` / `nexus serve`)
2. Desktop (aplicação nativa Wails v2 estável para Windows, macOS e Linux)

Ambas utilizando:
- O mesmo frontend React (`web/src`)
- O mesmo Nexus Core Go (`internal/app`, `internal/control`, `internal/core`, `internal/nexus`)
- Os mesmos contratos REST e WebSocket
- A mesma arquitetura de terminal (PTY / ConPTY -> WebSocket -> xterm.js)
- O mesmo Update Service unificado (`internal/update`) com suporte a InstallationMethod
- A mesma versão Nexus, canal e Git SHA
- Integração Maestro desacoplada e 100% opcional (sem instalação automática silenciosa)

## Aceitação

1. **Baseline Zero-Red**:
   - `make quality`, `web-verify`, `go test ./...` 100% verdes.
2. **Core Lifecycle & Discovery**:
   - Ciclo de vida desacoplado e reutilizável (`Core` / `ControlCenter`) inicializável tanto por CLI/Web quanto pela shell Desktop.
   - Descoberta e autenticação loopback segura via token/descritor de runtime.
3. **PlatformBridge Frontend**:
   - `PlatformBridge` abstrato em `web/src/platform/` consumido pela UI React.
   - `WebBridge` (browser) e `DesktopBridge` (Wails) sem `window.wails` solto no código.
   - Suporte nativo a seletores de arquivo/pasta, notificações nativas, deep links (`nexus://`), autostart, window management e system theme.
4. **Wails Desktop Shell**:
   - Entrypoint em `cmd/nexus-desktop/main.go` e pacote `internal/desktop/`.
   - Sem duplicação de React, runtime, PTY ou updater na shell Wails.
5. **Update Service Unificado**:
   - `internal/update/` gerencia manifestos assinados Ed25519, SHA256, canais e `InstallationMethod` (STANDALONE, NSIS, DEB, RPM, HOMEBREW, etc.).
   - Instalações gerenciadas por pacotes não sobrescrevem binários indevidamente.
   - CLI e Desktop UI utilizam exatamente o mesmo serviço.
6. **Maestro Isolado & Opcional**:
   - Removida instalação silenciosa do Maestro em `install.sh` e `install.ps1`.
   - Flag explícita `--with-maestro` / `-WithMaestro` nos instaladores.
   - Modo degradado gracioso (`MAESTRO_DEGRADED`) quando ausente, sem falhar o Nexus.
   - Separação clara de status e atualização entre Nexus e Maestro em Settings > Integrations e CLI.
7. **Suporte Multiplataforma**:
   - Windows (ConPTY, WebView2, NSIS), macOS (PTY, WKWebView, DMG/app), Linux (PTY, WebKitGTK, DEB/RPM).
   - Matriz de compatibilidade documentada e `nexus doctor` expandido com diagnósticos de desktop e SO.
8. **Documentação & ADRs**:
   - ADRs de Desktop (Wails v2), Update Architecture e Maestro Integration.
   - Relatórios em `DEV/validation/` e `docs/superpowers/reports/`.
