# AGY Handoff — IAPro Nexus Product Finalization

Você é o executor da campanha de finalização do IAPro Nexus. Trabalhe a partir do repositório:

`/projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/ai-manager/.worktrees/feat-nexus-product-finalization`

Branch esperada: `feat/nexus-product-finalization`.

Leia integralmente antes de alterar qualquer coisa:

1. `AGENTS.md`
2. `docs/engineering/ENGINEERING_STANDARDS.md`
3. `docs/engineering/WEB_ENGINEERING_STANDARDS.md`
4. `docs/superpowers/specs/2026-09-05-nexus-product-finalization-audit.md`
5. `docs/superpowers/plans/2026-09-05-nexus-product-finalization-implementation.md`
6. `docs/superpowers/plans/2026-09-05-nexus-platform-stabilization-productization.md`
7. Este arquivo

## Objetivo

Fechar os gaps restantes comprovados, principalmente:

- routing semântico, deep-link, refresh, back/forward e popout por projeto;
- layout durável com backend como autoridade, revisão otimista e restart E2E;
- preflight obrigatório e enforcement de segurança fail-closed;
- integração durável de Activity, Attention, runtime, Flow e Capacity Monitor;
- Playwright, axe, acessibilidade por teclado e regressão visual;
- execução, instalação e smoke tests nativos em Linux, Windows e macOS;
- release/update com manifestos Ed25519, checksum, SBOM, provenance, confirmação e rollback.

## Decisões já aprovadas

- Use `react-router-dom`; não crie router artesanal.
- URL é autoridade de localização semântica; `localStorage` é apenas fallback/cache.
- SQLite é a autoridade única de layout e Activity.
- Layout canônico é um envelope v4 contendo `model` e `presentation`, com `revision` monotônica e `409 Conflict` em conflito.
- Popout pode ler layout, mas não salva automaticamente.
- `PlanPreflight` é read-only; `ExecutionAdmission` reserva e revalida antes do dispatch.
- `ADVISORY` nunca satisfaz requisito `HARD`.
- Eventos do bus atual devem alimentar o histórico durável existente; não crie um segundo event bus concorrente.
- Capacity Monitor continua separado até integração explícita da branch `feat/capacity-monitor-notifications`; não faça merge/cherry-pick automático.
- Plataforma sem execução nativa não será anunciada como suportada; fica `build-only/experimental`.
- Assinatura beta: Ed25519 em GitHub Environment protegido; chave privada somente no workflow de release; nunca em PR.
- Não faça push, merge, tag, release ou publicação.

## Método obrigatório

Use TDD: escreva teste RED, implemente o menor comportamento, rode GREEN, faça review e só então avance. Preserve capabilities já verificadas. Não reescreva Mission Runner, DAG, terminal ou Workspace sem evidência de regressão.

Execute em ondas nesta ordem:

1. Router e separação global/projeto.
2. Layout v4, persistência backend, concorrência e restart E2E.
3. Preflight, admission, worktree, autonomy e segurança.
4. Activity/Attention e ações contextuais.
5. Playwright, axe, keyboard e visual regression.
6. CI nativo e install smoke por plataforma.
7. Manifesto assinado, updater, confirmação, receipt e rollback.
8. North Star E2E, gates G0–G10 e relatório final.

Para cada onda:

- registre evidências em `.superpowers/sdd/nexus-product-finalization/progress.md`;
- use arquivos reais descobertos no repositório;
- faça commits Conventional Commits pequenos;
- execute os gates focados antes da próxima onda;
- marque `VERIFIED`, `PARTIAL`, `MISSING`, `REGRESSED` ou `BLOCKED_BY_ENVIRONMENT` somente com evidência.

## Gates mínimos

```bash
npm --prefix web run quality:full
npm --prefix web run test:e2e
npm --prefix web run test:a11y
npm --prefix web run test:visual
go test ./...
go test -race ./...
go vet ./...
make lint-go
make security
make quality
goreleaser check
goreleaser release --snapshot --clean
git diff --check
```

Se algum comando não existir, descubra o equivalente real e registre. Não fabrique evidência nativa: diferencie `built`, `tested` e `natively_verified`.

Ao terminar, produza/atualize:

- `docs/superpowers/specs/2026-09-05-nexus-product-finalization-design.md`
- `docs/superpowers/plans/2026-09-05-nexus-product-finalization-implementation.md`
- `docs/superpowers/reports/2026-09-05-nexus-product-finalization-report.md`

O relatório final deve conter SHA base/final, branch, worktree, matriz de capabilities e evidências L1–L8, commits, testes, E2E, cross-platform, segurança, UX, riscos, G0–G10 e verdict `GO`, `CONDITIONAL_GO` ou `NO_GO`.

Comece agora pelo reconhecimento do estado atual e pelo ledger. Não pare apenas após escrever documentação: implemente os gaps comprovados e verifique cada claim.
