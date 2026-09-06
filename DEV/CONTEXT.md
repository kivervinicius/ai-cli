# Contexto atual

## Estado

- Produto: IAPro Nexus Workspace OS.
- Branch ativa: `feat/nexus-maximum-delivery`.
- Versão-fonte: arquivo `VERSION`.
- Launcher destacado usa envelopes privados de uso único para argumentos opacos.
- Motor de Flags Canônicas e Normalização (`internal/control/flags`): mapeamento transparente de `--yolo`, `-y`, `--continue`, `-c`, `--resume`, `-r`, `--print`, `-p`, `--effort`, `--plan` e suporte a `Merged Help` (`nexus <provider> --help`).
- Quotas em tempo real para Codex: parsing dinâmico de rollouts de sessão (`rollout-*.jsonl`) com rate limits primário (5h) e secundário (semanal) associados ao grupo `claude_gpt`.
- Chaveiro seguro de sessões D-Bus: isolamento de diretório de controle temporário exclusivo (`--control-directory`), eliminando conflito com o keyring do host e timeout de 25s de `org.freedesktop.secrets`.
- Perfis Codex, AGY e OpenCode podem reaproveitar artefatos locais de conversa sem copiar credenciais.

## Comandos

- Build: `make build` ou `go build -o /home/desenvolvedor/.local/bin/nexus ./cmd/nexus`.
- Testes focados: `go test ./internal/control/flags ./internal/control/driver ./internal/core/provider/adapters/codex ./internal/core/quota`.
- Web local: `yarn dev` ou `yarn web:dev`.

## Restrições

- Web local deve escutar em `127.0.0.1`; wildcard exige desenho remoto explícito.
- Dados de quota sem fonte/data verificável devem ser tratados como `UNKNOWN`.
- O worktree contém alterações não commitadas; não sobrescrever mudanças alheias.

