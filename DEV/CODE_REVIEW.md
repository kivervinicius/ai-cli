# Code Review — alterações atuais

Data: 2026-09-02
Escopo: branch `feat/nexus-maximum-delivery` (working tree + commits vs merge-base)
Revisão: segunda rodada após achados independentes (autonomia, broker, approve, terminal, hardening)

## Veredito

**CORREÇÕES APLICADAS NESTA RODADA.** Os bloqueadores críticos identificados no review independente foram tratados; gates locais passam.

## Status das correções

| ID | Achado | Status |
|---|---|---|
| C1 | Guardas de autonomia não bloqueavam rede/segredos/pagos | **Corrigido** — `blockedPatterns` reordenado; `__nexus_block_all__` bloqueia incondicionalmente |
| C2 | CSPRNG bootstrap | **Já corrigido** (mantido) |
| C3 | Broker deadlock/panic/writers concorrentes | **Corrigido** — unlock em early-return, Detach sem double-unlock, `writeMu` compartilhado |
| C4 | Expansão `~/` apontava para `/` | **Corrigido** — `expandHomePath()` |
| C5 | Approve & Run sem `plan_revision` | **Corrigido** — `StartMissionRunApproved` obrigatório |
| C6 | `startAgent` implícito no mount do terminal | **Corrigido** — removido de `AgentTerminal` |
| H1 | I/O sob `SessionHost.mu` | **Corrigido** — writes/resize fora do lock |
| H2 | PTY sem dreno final | **Corrigido** — drain Unix; fechamento ConPTY Windows |
| H3 | AUTO/EXISTING reutilizavam mesmo Agent em paralelo | **Corrigido** — reserva por run |
| H4 | Dirty fingerprint não detectava reedição | **Corrigido** — hash de conteúdo |
| H5 | WorkReceipt atribuía diff global | **Mitigado** — prioriza `git diff` base→result |
| H6 | AUTO com quota UNKNOWN sem LRU | **Corrigido** — fallback LRU em `RecommendResources` |
| H7 | Pause/Cancel sem lease | **Corrigido** — `AcquireLease` em pause/resume/cancel |
| H8 | Checkout alterava `DefaultBranch` | **Corrigido** — removida persistência indevida |
| H9 | Composer sem gate de Context Readiness | **Corrigido** — `PlanBuilderSurface` |
| H10 | Lease incompleto no AgentTerminal | **Corrigido** — Take Control / Release |
| H11 | Launcher canônico incompleto | **Corrigido** — New AI Session + Flow Runs ativos |
| H12 | Split Right/Down só refocava | **Corrigido** — dedupe por `surface.id` |
| H13 | Copy canônico desatualizado | **Corrigido** — i18n Composer/Flow Runs |

## Evidências executadas

- `go test ./...`: **PASS**
- `go vet ./...`: **PASS**
- `yarn --cwd web typecheck`: **PASS**
- `yarn --cwd web test`: **PASS** — 33 arquivos / 115 testes

## Recomendação

**READY FOR PLATFORM VALIDATION.** Merge ainda depende de validação externa (CI same-SHA, Windows/macOS nativos, providers reais, E2E browser completo) conforme `DEV/NEXUS_V0_FINAL_AUDIT.md`.
