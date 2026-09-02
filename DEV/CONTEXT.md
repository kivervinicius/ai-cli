# Contexto atual

## Estado

- Produto: IAPro Nexus Workspace OS.
- Branch ativa: `feat/nexus-maximum-delivery`.
- Versão-fonte: arquivo `VERSION`.
- Launcher destacado usa envelopes privados de uso único para argumentos opacos.
- Perfis Codex, AGY e OpenCode podem reaproveitar artefatos locais de conversa sem copiar credenciais.

## Comandos

- Build: `make build`.
- Testes focados: `go test ./internal/control/driver ./internal/control/launcher ./internal/core/quota ./internal/core/scheduler`.
- Web local: `yarn dev` ou `yarn web:dev`.

## Restrições

- Web local deve escutar em `127.0.0.1`; wildcard exige desenho remoto explícito.
- Dados de quota sem fonte/data verificável devem ser tratados como `UNKNOWN`.
- O worktree contém alterações não commitadas; não sobrescrever mudanças alheias.
