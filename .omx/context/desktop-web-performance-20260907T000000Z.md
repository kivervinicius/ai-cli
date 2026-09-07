# Ralph context snapshot — Desktop/Web performance

## Task statement

Ajustar a lentidão percebida da versão Desktop do IAPro Nexus em relação à Web.

## Desired outcome

Identificar e corrigir os gargalos de startup, bootstrap, renderização,
terminal, Flow e polling que forem sustentados por evidência, preservando a
arquitetura compartilhada Web/Desktop.

## Known facts/evidence

- Desktop e Web compartilham frontend React, Core Go e API REST/WebSocket.
- `cmd/nexus-desktop/main.go` inicia Core, aguarda readiness, cria sessão e
  então inicia Wails.
- `web/src/platform/desktopBridge.ts` faz retry de binding Wails por até 1 s.
- `web/src/api.ts` faz bootstrap Desktop e verificação de sessão.
- `web/src/nexus/AgentTerminal.tsx` processa WebSocket, xterm, RAF e resize.
- `web/src/features/work/FlowCanvas.tsx` reconstrói nodes/edges e usa `find`
  durante mudanças de node.
- Linux Desktop usa WebKitGTK e ainda consta como native smoke pending.
- A build frontend é minificada e code-split pelo `web/scripts/build.mjs`.

## Constraints

- Preservar alterações existentes e não fazer commit/push automático.
- Seguir `AGENTS.md`, padrões de engenharia, SCSS Modules/i18n e TypeScript
  strict.
- Não declarar correção sem testes/build e sem registrar limitações nativas.
- Não reescrever Wails, duplicar Core ou introduzir backend paralelo.

## Unknowns/open questions

- Causa dominante ainda não medida: startup, WebView/GPU, React, terminal,
  Flow, polling ou backend.
- Não há benchmark comparativo automatizado Web/Desktop no repositório.
- Windows/macOS não possuem execução nativa disponível neste ambiente.

## Likely touchpoints

- `cmd/nexus-desktop/main.go`
- `internal/desktop/`
- `web/src/platform/desktopBridge.ts`
- `web/src/api.ts`
- `web/src/app/NexusWorkspaceApp.tsx`
- `web/src/nexus/AgentTerminal.tsx`
- `web/src/workspace/PtyLiveChromeContext.tsx`
- `web/src/features/work/FlowCanvas.tsx`
- `web/scripts/`
- `Makefile`
- `DEV/validation/`
