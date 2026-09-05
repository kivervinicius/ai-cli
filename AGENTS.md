# IAPro Nexus — Contrato local para agentes

## Estado do produto

O Nexus ainda não está em produção. São permitidas refatorações estruturais
compatíveis com o objetivo do produto; não preserve implementações duplicadas
somente por compatibilidade interna sem uma necessidade explícita.

## Quality Baseline

Antes de declarar qualquer tarefa concluída, execute:

```bash
make quality
```

Este é o contrato local rápido. Ele executa:
- format:check (Prettier + gofmt)
- lint:frontend (ESLint)
- lint:styles (Stylelint)
- typecheck (TypeScript)
- lint:go (golangci-lint)
- test:go + test:frontend

Para validação completa:

```bash
make quality:full
```

## Frontend (web)

### Comandos

```bash
make lint:frontend   # ESLint
make lint:styles     # Stylelint
make typecheck       # TypeScript
make test:frontend   # Vitest
make format          # Prettier
```

### Regras

- Após qualquer alteração em `web/`, rode `make quality` antes de declarar
  conclusão.
- Arrays vindos da API Go podem ser JSON `null`. Nunca use `.length`/`.map` direto
  em `dependencies`, `phases`, `packages`, `generations`, etc. — normalize com
  `asArray` (`web/src/lib/safeArray.ts`) ou `(value || [])`.
- Não crie novas primitivas de UI se já existirem em `web/src/design-system/primitives/`.
- Não crie utilities globais sem necessidade real.
- Não adicione `@ts-ignore` sem justificativa documentada.
- Não adicione suppressões de lint para passar CI.

### Arquitetura

```text
web/src/
├── app/           # Composition root
├── workbench/     # IDE shell (orchestration)
├── features/      # Product capabilities
├── shared/        # Truly cross-cutting code
└── assets/        # Static assets
```

`shared/` nunca importa de `features/`. Features não importam internals umas das outras.

## Backend (Go)

### Comandos

```bash
make lint:go         # golangci-lint
make test:go         # go test -v ./...
make test:e2e        # E2E tests
make security        # govulncheck
```

### Regras

- Entry points em `cmd/` — apenas wiring.
- Lógica de negócio em `internal/` packages por capacidade.
- Código específico de plataforma: `*_unix.go` / `*_windows.go`.
- Não criar estrutura Java (`controllers/services/repositories`).
- Toda nova suppression `//nolint` deve conter nome do linter e justificativa.

## Interface de terminal

- Todo fluxo visual/interativo de terminal deve usar o stack Charm já adotado:
  Bubble Tea para ciclo de interface, Bubbles para componentes e Lip Gloss para
  estilos. Não criar tabelas interativas com `fmt.Printf` ou bibliotecas
  concorrentes.
- Dados de integração ou automação devem oferecer `--json`; o modo humano é a
  interface Charm padrão, sem flag visual adicional.
- Para tabelas, incluir navegação por teclado e filtro/pesquisa quando houver
  mais de uma linha ou quando o usuário precise localizar uma identidade.

## Coding Agents

Antes de modificar:
1. Entenda o owner da feature
2. Não crie utility global sem necessidade
3. Não crie nova primitiva se já existir
4. Não ignore lint
5. Não adicione suppressões para passar CI

Antes de terminar:
1. Execute `make quality`
2. Verifique que testes passam
3. Verifique formatação

Agents não podem redefinir o padrão do projeto.

## Git Commits

Use Conventional Commits:

```
type(scope): description

Types: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert
Scopes: terminal, chat, flow, agents, providers, quota, workspace, settings, web, go, infra, security, ui, deps, release
```

Veja `docs/engineering/ENGINEERING_STANDARDS.md` para o catálogo completo.
