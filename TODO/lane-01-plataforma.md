# Lane 01 — Plataforma

Objetivo: transformar o `CONDITIONAL_GO` em evidência de runtime completa.

## P0 — Em aberto

- [ ] Executar runtime E2E no Windows (amd64/arm64), cobrindo ConPTY, Named Pipe e PowerShell.
- [ ] Executar runtime E2E no macOS (amd64/arm64), incluindo Apple Silicon, PTY, socket e Web.
- [ ] Publicar a matriz de resultados e atualizar `DEV/NEXUS_WORKSPACE_OS_PLATFORM_MATRIX.md`.

## Critério de aceite

Cada plataforma deve ter logs reproduzíveis, comandos executados, artefatos de teste e status `RUNTIME VERIFIED`; compilação isolada não é suficiente.

