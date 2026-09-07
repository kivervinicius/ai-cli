# Sessão de decisões — Terminal, quota e continuidade

Data: 2026-09-07  
Contrato: `DEV/SPECS/NEXUS_TERMINAL_CONTINUITY_LUNA.md`  
Base local: `1899ca6334576d859056d48a394e51d03758f313`

Esta sessão registra decisões tomadas durante a execução do contrato Luna. Ela
separa o que foi implementado e verificado localmente do que ainda depende de
runner nativo, credencial, CI remoto ou integração externa.

## Decisões aceitas

### Quota não observada nunca é saldo

`UNKNOWN`, `UNSUPPORTED`, `ERROR` e observação vencida não são convertidos em
0%, 100% ou verde. Verde é reservado a uma observação `LIVE` com janela e
percentual válidos. `CACHED`/`ESTIMATED` permanecem explicitamente degradados.
Isso também explica por que uma conta Codex sem observação confirmada não gera
alerta: emitir alerta de quota nesse caso seria fabricar evidência.

### Alertas usam episódio, não texto

O alerta é identificado por provider, domínio de quota, conta/pool, grupo,
janela, limiar e episódio. O lease interprocesso do monitor impede três
processos Nexus de coletarem e notificarem o mesmo evento. Uma queda de quota
para um limiar mais grave pode gerar um novo episódio; polls repetidos não.

### Conta agrupada, credencial separada

Web/Desktop exibem um grupo por provider e deixam os perfis expandidos para
diagnóstico e seleção manual. Agrupar visualmente não soma percentuais nem
mistura credenciais. A identidade de quota continua sendo resolvida pelo
backend; e-mail isolado não prova que duas contas têm limite independente.

### Prioridade é política configurável

O default é Codex (1), AGY (2), OpenCode (3), mas a seleção não trata isso como
capacidade. Um provider com quota confirmada pode vencer um provider prioritário
sem quota; `UNKNOWN` não é verde. Troca automática entre providers continua
dependendo da política persistida e da transação de handoff, não de `--yolo`.

### Supervisão não é padrão ainda

`nexus <provider> --supervised` usa o SessionHost e torna os controles Nexus
disponíveis; o caminho direto permanece compatibilidade até haver evidência
nativa de paridade em Linux, macOS e Windows. `--yolo` autoriza a execução do
provider, mas não concede autorização implícita para trocar provider, conta,
modelo ou executar fallback.

### Continuidade precisa ser honesta

O contrato distingue sessão lógica, geração de processo, checkpoint salvo,
checkpoint entregue e reidratação confirmada. Trocar o rótulo de provider não é
continuidade. Se o destino não confirmar contexto e autoridade de escrita, o
estado deve permanecer recuperável/pendente; não pode ser anunciado como
`LIVE`.

## Decisões rejeitadas

- `time.Sleep`/retry fixo como correção de readiness de named pipe.
- `UNKNOWN → LIVE` ou `quota_remaining` sem fonte reconhecida.
- Somar duas contas em “160% de quota”.
- Fallback automático acionado por qualquer exit code diferente de zero.
- Alterar binários dos providers para injetar slash commands.
- Declarar macOS/Windows verdes com cross-build, Wine ou logs históricos.
- Tornar o modo supervisionado default antes da matriz nativa.

## Evidência desta sessão

- `make quality` terminou com `make_quality_exit=0` após os ajustes de lint e
  formatação.
- `go test -race -count=1 ./...`, `go vet ./...` e `cd web && bun run verify`
  passaram na execução desta sessão antes da alteração visual final.
- TypeScript e os testes focados do modelo de quota passaram após a alteração:
  `bun run typecheck` e `bun run test --run src/features/work/directSessionModel.test.ts`.
- A auditoria local de providers encontrou Codex e AGY autenticados, mas sem
  observação numérica atual (`UNKNOWN`/sem `fetched_at`); OpenCode instalado sem
  autenticação e Gemini instalado sem profiles. Isso é diagnóstico, não prova de
  saldo.

## Pendências que bloqueiam promoção oficial

1. Reexecutar a matriz nativa Windows/macOS, incluindo ConPTY, named pipe,
   resize, cleanup e worktree identity.
2. Integrar o agrupamento de pools ao contrato backend e concluir handoff
   transacional/reconexão antes de ligar fallback interativo automático.
3. Executar o CI remoto no mesmo SHA; cross-compilação Linux não substitui esses
   gates.
4. Só depois configurar branch protection/required checks com autorização
   administrativa e preparar tag.
