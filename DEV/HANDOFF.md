# Handoff

## Atualização 2026-09-07 — Auth inválida ao abrir `nexus web`

A causa era o comando abrir apenas a URL base depois que o token deixou de ser
colocado na URL. A SPA recebia `/api/v1/session` sem cookie e entrava como não
autenticada.

Agora `Server.BootstrapURL()` entrega o token em fragmento
`#nexus_bootstrap=...`; `web/src/api.ts` troca esse token por POST, remove o
fragmento com `history.replaceState` e então valida a sessão normalmente.
`nexus web open` e `nexus web url` usam o BootstrapURL persistido, inclusive
para estados antigos que ainda armazenam apenas o token separado.

Validação real concluída com `nexus web --port 0 --no-open`: bootstrap, cookie,
sessão autenticada e CSRF foram confirmados. O token não aparece em query string
nem em URL enviada ao servidor.

## Atualização 2026-09-07 — Mídia documental sem dados reais

O capturador documental agora trabalha somente com dados temporários e
sintéticos. Ele cria `Nexus Demo Workspace`, inicia um Project Shell real com
prompt neutro, exige retorno do marcador `__NEXUS_DOCS_TERMINAL_OK__` e falha
quando encontra erro, recuperação, desconexão ou dados conhecidos do host.

O modo `NEXUS_DOCS_CAPTURE=1` também substitui o inventário de providers por
`demo-provider`/`synthetic-fixture`, evitando que versões de Codex, Claude,
AGY ou outros binários instalados apareçam na documentação. O manifesto e o
verificador exigem classificação `SYNTHETIC`; VIS-005 contém evidência de
terminal e VIS-011 permanece explicitamente como tela inicial Desktop sem
projeto/runtime.

Validação concluída: captura Web isolada PASS, teste Go de inventário sintético
PASS e `node scripts/docs-verify.mjs` PASS. Não considerar o screenshot Desktop
como prova de terminal nativo; ele documenta apenas o estado inicial seguro.

## Atualização 2026-09-07 — Memoização do Flow

`FlowTaskNode` foi memoizado após a revisão final do caminho de renderização.
O frontend foi revalidado com `verify` 10/10. O próximo passo continua sendo
benchmark nativo; não há evidência suficiente para afirmar FPS ou GPU.

## Atualização 2026-09-07 — Primeiras correções de desempenho

Foram aplicadas três correções: Flow Canvas não reconstrói o grafo ao mudar
somente a seleção e usa `applyNodeChanges`; DesktopBridge coalesce bootstrap
concorrente; AgentTerminal agrupa saída WebSocket por frame antes de escrever
no xterm. Frontend, Go, race/vet e build Wails de produção passaram.

Próxima ação exata: implementar/executar o benchmark comparativo do plano
`.omx/plans/desktop-web-performance-parity.md` e confirmar se WebView/GPU,
startup ou outra superfície ainda excede a meta de 1,20x da Web.

## Atualização 2026-09-07 — Plano de paridade Web/Desktop

O plano executável para diagnosticar e corrigir a lentidão percebida do Desktop
está em `.omx/plans/desktop-web-performance-parity.md`. Ele exige baseline na
mesma máquina/SHA, instrumentação do caminho crítico e benchmark comparativo
antes de otimizar. A primeira execução deve realizar as Fases 0–2; somente
depois escolher as duas trilhas de causa com maior contribuição ao p95.

Não houve alteração de código nem confirmação de causa nesta etapa. A meta
proposta é Desktop <= 1,20x Web nos fluxos críticos; Linux requer smoke nativo,
e Windows/macOS continuam sem afirmação de correção até execução nativa.

## Atualização 2026-09-07 — AGY sem keyring no monitor de quota

O probe de quota do AGY não inicia mais Secret Service nem keyring privado. O
keyring isolado continua reservado ao fluxo interativo/login (`Run`). Quando o
probe não consegue usar o armazenamento em arquivo, a quota permanece
`UNKNOWN`/última observação e não é usada como capacidade atual. O binário em
`/home/desenvolvedor/.local/bin/nexus` foi recompilado e instalado; a Web foi
reiniciada em `127.0.0.1:3000` e o health check passou.

Também foi removido o menu duplicado `Novo` do cabeçalho de Terminais. O único
ponto global de criação é o menu `Criar`.

Próxima validação manual: abrir o Nexus, observar por pelo menos um ciclo de
quota e confirmar que não aparece solicitação de senha; abrir uma sessão AGY
interativa e confirmar que login/execução continuam funcionando.

O Desktop Linux também foi recompilado e iniciado como
`/home/desenvolvedor/.local/bin/nexus-desktop`. A Web foi iniciada novamente em
`127.0.0.1:3000`. Os runtimes AGY, Codex e shells já existentes permaneceram
ativos; não houve encerramento em massa.

## Execução Luna em andamento — 2026-09-07

O plano `DEV/SPECS/NEXUS_TERMINAL_CONTINUITY_LUNA.md` está sendo executado.
P0/P1 local passaram: Go completo/vet e frontend 10/10. A implementação já
inclui `nexus <provider> --supervised` opt-in, completion de flags, checkpoint
Maestro schema 4 com digest dos arquivos DEV e lease interprocesso para eleger
um coletor de quota. Testes focados passaram.

Pacotes confirmados nesta sessão: P0/P1 local; P3 lease do monitor; P4 prioridade
cross-provider; P6 `--supervised` opt-in; P7 sugestões `/nexus<TAB>` e completion
shell; P8 checkpoint Maestro. Host, handoff e nexus também compilam cruzado para
Windows amd64 e macOS arm64; isso não substitui execução nativa.

Ainda não declarar conclusão: pools entre providers, fallback automático,
autocomplete interno, transação/reconexão completa e execução nativa Windows/
macOS continuam pendentes. Na próxima sessão, reidratar este handoff e o plano;
rodar o gate completo após cada pacote. A decisão de manter supervisão opt-in
até evidência nativa deve ser revista somente após P6/P12.

Quota atual: a auditoria de 2026-09-07 encontrou contas AGY/Codex autenticadas,
porém sem fonte/janela observada (`UNKNOWN`/`NONE`); OpenCode não autenticado e
Cursor sem Usage. Não usar esses números para anunciar capacidade ou fallback.

## Plano para Luna — 2026-09-06

Leia [`SPECS/NEXUS_TERMINAL_CONTINUITY_LUNA.md`](SPECS/NEXUS_TERMINAL_CONTINUITY_LUNA.md)
para executar a próxima campanha. Contém P0–P12, evidência de código, contratos,
matriz nativa e prompt. Esta etapa alterou somente documentação; nenhum novo PASS
de runtime/CI foi produzido. Começar pelo inventário do worktree e baseline.
Correções às promessas anteriores: quota não exposta é desconhecida; checkpoint
não prova reidratação; handoff exige reconexão e recuperação real; singleton por
processo não evita alerta duplicado entre instâncias. Flags novas são propostas.

## Atualização 2026-09-06 — Fechamento local dos reds do CI

O run remoto `34060911997` falhou em Frontend (Prettier), Windows (teste Go
completo) e macOS (race test). O acesso aos logs detalhados exige permissão de
admin, mas os jobs/steps falhos foram confirmados pela API.

As causas reproduzíveis foram corrigidas no worktree: `QuotaDropMonitor` aceita
os estados legados `OK`/`RATE_LIMITED` sem reabrir estados desconhecidos e
restaura a recomendação de failover; os quatro arquivos reportados pelo
Prettier foram formatados; e `agyQuotaGroupAvailable` não tem mais retorno
nomeado inútil para o golangci-lint v2.

Evidência fresca: `bun run verify` 10/10 e 306 testes; `go test ./...`; `go
test -race ./...`; `go vet ./...`; golangci-lint v2.12.2 com 0 issues; teste e
binário compilados para Windows amd64 e Darwin arm64; Wails Linux; e
GoReleaser v2.18.0 snapshot. Próxima ação exata: revisar o diff, criar o commit
`fix(ci): close cross-platform runtime failures` e fazer push manual para
disparar uma nova matriz CI. Não houve commit/push automático.

## Atualização 2026-09-06 — Monitor residente de quotas

O monitor de quotas agora é residente no processo Nexus: Core Web/Desktop o
inicia e encerra com o contexto; execuções diretas `nexus agy|codex|cursor` e
terminais/Agentes também o ativam de forma idempotente. A primeira coleta é
imediata e as seguintes usam ticker de 60s e cache de cinco minutos.

Alertas são independentes por provedor, perfil, grupo, janela e ciclo de reset.
Somente `LIVE`/`CACHED` alertam consumo; leituras degradadas emitem no máximo
um evento de monitoramento degradado e a recuperação rearma o estado. O estado
de supressão fica em `StateDir/quota-monitor-state.json` com escrita atômica.

O histórico sem `runtime_id` agora agrega eventos de todos os runtimes, e a UI
exibe alertas globais, grupo/janela e idade da leitura. A aba Recursos não é
mais responsável por iniciar o monitor.

Verificação concluída: `go test ./...`, `go vet ./...`, typecheck, 306 testes
Vitest, lint/style gates e build Web passaram. Não houve commit ou push.

## Atualização 2026-09-06 — Quota AGY e seleção de conta

O cache de quota agora respeita TTL de cinco minutos para todos os providers;
cache AGY expirado vira `UNKNOWN` e não pode ser usado como capacidade atual.
O adaptador AGY tenta leitura ao vivo quando a cache expirou, e o scheduler usa
a fotografia por perfil capturada durante a seleção.

Contas sem quota conhecida não vencem contas com quota positiva conhecida. Com
os dados atuais, `kiver.omegasistemas@gmail.com` está `UNKNOWN`, enquanto
`kivervinicius@gmail.com` reporta 28% Gemini e 32% Claude/GPT; `nexus explain agy`
seleciona `kivervinicius-gmail`.

Web e Desktop foram recompilados; a Web em `127.0.0.1:3000` foi reiniciada com
o binário atualizado.

## Atualização 2026-09-06 — Correção reforçada de abas e xterm

O sincronizador de rota agora só reabre uma superfície quando a rota realmente
mudou; isso impede que a rota antiga `/overview` sobrescreva o clique em
`Terminal`. As abas também usam o mesmo callback de abertura da aplicação para
atualizar estado e URL de forma atômica.

`AgentTerminal` e `TerminalPane` não escrevem, focam ou fazem `fit` enquanto o
painel está oculto. A saída fica pendente e é descarregada após `data-active` ser
ativado, evitando o erro de `Viewport._innerRefresh` com `dimensions` indefinido.

O frontend foi recompilado e o `nexus-desktop` local foi reinstalado com essa
correção. Próximo teste manual: abrir o Desktop atualizado, clicar em
`Visão geral`, depois `Terminal`, e alternar as abas algumas vezes observando o
console.

## Atualização 2026-09-06 — Auditoria de fechamento atual

O baseline autoritativo continua em `feat/nexus-maximum-delivery`, HEAD
`1899ca6334576d859056d48a394e51d03758f313`, sem commits locais à frente do
remoto e com alterações não commitadas preservadas. A auditoria atual está em
[`validation/CURRENT_STATE_AUDIT.md`](validation/CURRENT_STATE_AUDIT.md) e o
relatório final em
[`validation/FINAL_PLATFORM_RELEASE_DESKTOP_REPORT.md`](validation/FINAL_PLATFORM_RELEASE_DESKTOP_REPORT.md).

Estado verificável: frontend 10/10, Go test/vet/race, Desktop Linux build e
diff check verdes. O Update Service agora falha fechado para manifesto sem
assinatura e o Desktop não anuncia capabilities não implementadas. O veredito
continua **NO-GO** porque o tree remoto commitado no SHA base falhou nos jobs
nativos Windows e macOS do CI `34012236345`; o worktree atual contém fixes
adicionais não publicados. Logs detalhados estão bloqueados por permissão de admin,
e `make security` reportou `No vulnerabilities found` usando o fallback
versionado quando o binário standalone não está no `PATH`.

Próxima ação exata: obter logs completos ou executar os runners Windows/macOS,
reproduzir os testes vermelhos por nome, corrigir causa raiz e repetir o mesmo
SHA; depois integrar a trust chain dos instaladores e executar browser,
Desktop e GoReleaser somente após os gates upstream verdes.

## Atualização 2026-09-05 — Implementação Completa Web + Desktop Multiplataforma

A implementação da arquitetura multiplataforma equivalente Web + Desktop foi concluída com sucesso no branch `feat/nexus-desktop-multiplatform`:
- **Dual Surface Equivalence**: Web (`nexus web`) e Desktop (`nexus-desktop` via Wails v2 estável) consomem exatamente o mesmo Core em Go (`internal/app/core.go`), o mesmo frontend React (`web.EmbeddedDistFS()`) e a mesma API REST / WebSocket.
- **PlatformBridge Abstrato**: Contrato unificado em `web/src/platform/` eliminando condicionais de plataforma espalhadas e integrando seletores de pastas, notificações, tema do SO e deep links.
- **Update Service Único**: Centralizado em `internal/update/service.go` com verificação estrita de assinatura Ed25519, SHA256 e respeito ao `InstallationMethod`.
- **Maestro Opcional**: Remoção de instalações silenciosas em `install.sh` / `install.ps1`, suporte à flag explícita `--with-maestro` e degradação graciosa para `MAESTRO_DEGRADED`.
- **Verificação**: `bun run quality` (8/8 gates, 280 testes unitários), `go test ./...` e `go vet ./...` 100% verdes. Sem auto-commit ou auto-push.

## Atualização 2026-09-05 — Autopilot Platform Stabilization

A campanha `01a07224-1c17-7863-881a-6b1963a6ce43` está em execução no branch
`feat/nexus-maximum-delivery`, HEAD `ab88fcbfddb5d99cf77c5e6f651aa2e5aa281770`.
T0 foi concluído. T1 corrigiu lockfile e pinos da CI, mas o lint reproduz 59
achados existentes; isso mantém o release bloqueado. T2 criou o
`ResolvedCommand`; T3/T4 corrigiram o teste ConPTY bloqueante, a semântica de
`ClosePseudoConsole`, a ordem IPC-before-provider e os metadados de startup.

Próxima ação: executar/verificar Job Objects no runner Windows, depois fechar
path identity, credenciais e doctor. Não anunciar suporte Windows/macOS nem
fazer release enquanto a matriz nativa não estiver verde. Não houve commit ou
push automático.

O incremento seguinte adicionou `PathRef`/`FilesystemIdentity`, migração SQLite
aditiva `0012_path_identity.sql` e capacidades explícitas de isolamento de
credenciais por SO. Os testes Go e vet continuam verdes. A próxima ação é
consolidar `nexus doctor` e produzir bundle diagnóstico redigido; a evidência
nativa Windows/macOS continua obrigatória.

`nexus doctor` agora usa um relatório read-only compartilhado para texto/JSON e
ZIP allowlisted (`report.json` + `MANIFEST.txt`). `nexus control doctor` só faz
limpeza com `--repair`. Próximo passo: expor o mesmo relatório na Web e obter
execução nativa Windows/macOS antes do checkpoint de runtime.

## Atualização 2026-09-04

Baseline do working tree revalidado após correções pequenas de P0. `make
web-verify`, `go test -count=1 ./...`, `go vet ./...` e `git diff --check`
estão verdes. O CI agora fixa Bun `1.3.9`; Composer possui rota/UI para
aceitar ou dispensar Maestro skills, envia skills aceitas ao PromptArtifact e
exibe Prompt Readiness, Unknowns e Assumptions.

As alterações locais anteriores foram preservadas; não houve commit/push.
PromptArtifact → Flow e o DAG visual editável já foram implementados neste
ciclo. Próxima ação: validar manualmente drag/zoom/conexão em flows de 20, 50 e
100 nodes e formalizar `FlowMaterializationRequest` caso o contrato externo
exija esse tipo nomeado.

## Current state

Nexus was rebuilt from the current source for local validation. Machine-local
installation paths and binary locations are intentionally omitted from this
durable handoff.
O servidor `nexus web --port 3000` está ativo com o bundle mais recente incorporando o seletor de temas em Accordion com paleta visual, densidade dinâmica real (`compact` vs `comfortable`), Topbar com proteção de largura e Terminal prioritário sem clipping.
Bootstrap ativo: `[REDACTED — ephemeral local secret; do not persist]`.

## Verification

- `make web-verify`: 8/8 gates aprovados (typecheck, lint, null-arrays, vitest, i18n, build, embed-sync, ui-markers).
- `npm --prefix web run test:e2e-hardening`: 100% aprovado cobrindo 6 viewports (320px até 1440px), verificação de não-obstrução do botão Terminal e delta de densidade.
- `make build`: Binário compilado com assets embarcados sincronizados e instalado em `/home/desenvolvedor/.local/bin/nexus`.

## Revisão atual — 2026-09-04

Revisão técnica registrada em [`DEV/validation/CURRENT_CODE_REVIEW.md`](validation/CURRENT_CODE_REVIEW.md).
O estado é **aprovado para uso local individual em loopback**. O frontend evita
polling concorrente e polling em background; o handler de terminal encerra
recursos de forma idempotente e limita resize concorrente. Build final instalado:
`nexus v0.5.0-beta.23`; Web ativo em `http://127.0.0.1:3000`.

Riscos que continuam explícitos: `--remote` usa HTTP sem TLS (risco aceito para
LAN privada), verificações de missão executam shell confiado sem sandbox hostil,
e macOS/Windows/Safari só têm evidência de compilação/configuração, não execução
local nesta máquina. Não há app nativo iOS.

Os controles dos terminais agora exibem estados de aplicação/reinício/pronto/erro,
feedback de lease e confirmação explícita para fechar mantendo ou parando o
runtime. Project Shells confirmam o encerramento do processo. O badge superior
resume trabalho, espera, desconexão e degradação; clique em runtime vivo foca o
terminal e clique em desconectado abre Agentes para recuperação.

Última validação: `make web-verify`, `go test ./...` e `make build` passaram em 2026-09-04 com a aplicação da identidade visual oficial e geração do pacote de assets de marca.

O Codex agora aparece como recurso compatível no modo Coding CLI quando autenticado,
usando `codex exec` para prompts não interativos. O servidor foi reiniciado após o
build e está ativo em `http://127.0.0.1:3000`.

A confirmação de modo ocorre antes da persistência/reinício; o fechamento de abas,
Project Shells e runtimes usa confirmação Nexus centralizada. Bootstrap atual:
`[REDACTED — ephemeral local secret; do not persist]`.

AGY agora mantém o modelo por perfil e recebe o diretório privado do keyring
explicitamente. O binário instalado é `nexus v0.5.0-beta.17`; reinicie sessões
AGY antigas para usar o wrapper corrigido.

CLIs interativos não recebem mais `TERM=dumb` quando o Nexus possui PTY real;
isso evita a confirmação bloqueada “Continue anyway?” do Codex e cobre também
agentes supervisionados.

O atalho `Sessão IA` agora abre o Composer e o launcher de sessão depois da
montagem da superfície, evitando perder o evento no primeiro clique. Na shell,
`Controle` indica quem pode digitar; `Liberar` devolve essa permissão e
`Assumir controle` solicita a permissão quando a sessão está em somente leitura.

Terminais de Agentes desconectados usam imediatamente o `runtime_id` retornado
por Recover/Start, sem esperar o polling global. O botão `Fechar terminal` remove
a aba visual; o Agente persistente continua separado até ser parado em Agentes.

Quando um projeto não tem contexto durável, o Composer informa que criará um
`AGENTS.md` base. A criação só ocorre ao clicar no botão, é atômica e não
substitui `AGENTS.md`, `DEV/INDEX.md` ou `DEV/CONTEXT.md` existentes.

Para AGY, a disponibilidade da conta exige quota nos dois grupos de modelos
(Claude/GPT e Gemini). Uma conta com somente um grupo disponível aparece como
indisponível e não deve ser escolhida pelo scheduler; os badges dos grupos
continuam exibindo o detalhe de cada pool.

O erro de runtime `Cannot read properties of null (reading 'forEach')` foi
corrigido no Flow: payloads com `phases` ou `packages` nulos são tratados como
vazios, e o bundle embutido foi recompilado. O teste de regressão e `yarn build`
passaram; o typecheck também passou na validação final.

A proteção foi centralizada em `web/src/nexus/workPlan.ts`; os endpoints de
planos normalizam as respostas na entrada, evitando que novas telas repitam a
mesma vulnerabilidade.

Terminais de Agentes também foram ajustados para não fixarem uma geração
antiga: após Recover/Start, a superfície resolve novamente pelo `agent_id` e o
bundle embutido foi recompilado.

`nexus usage` foi recompilado para mostrar a capacidade completa por modelo e
os resets de cada janela (5h/semanal), sem reduzir a saída ao bottleneck.

Sessões Web rotacionam automaticamente antes do vencimento. Se o servidor for
reiniciado ou a sessão ficar ociosa além do limite, a interface explica como
abrir o novo Bootstrap exibido por `nexus web`.

O uso de recursos é apresentado em tabela no CLI e em grid responsivo no
dashboard, com capacidade/reset por grupo e status sem sobreposição visual.
Para localizar contas e grupos no terminal, use `nexus usage` e
pressione `/` para filtrar.

O banner de atenção do agente também foi corrigido: ele não ocupa mais a área
central do dashboard e a mensagem padrão `? for shortcuts` não abre aviso
falso. O binário instalado contém esses ajustes.

Conclusões de tarefa e erros usam agora um componente de notificações
transitórias próprio; perguntas e aprovações continuam visíveis no banner por
exigirem resposta.

O motor de Flags Canônicas e Merged Help (`internal/control/flags`) foi incorporado
com sucesso: comandos universais (`--yolo`, `-y`, `--continue`, `-c`, `--resume`, `-r`,
`--print`, `-p`, `--effort`, `--plan`, `--accept-edits`) são traduzidos em tempo de
execução para cada CLI nativo e exibidos com destaque em `nexus <provider> --help`.

## Finalization status — 2026-09-06

The current dirty worktree preserves the user changes and the validation
artifacts. Local Linux/frontend/browser/security/release-static gates are green.
Public support claims now distinguish verified Linux evidence from pending
native Windows/macOS evidence; public community files and the MIT license
metadata are aligned. Installer documentation explicitly records the remaining
checksum-only trust-chain limitation.

The last remote run (`34012236345`) belongs to the committed tree at the same
base SHA, not to these uncommitted fixes. It failed in Frontend Format, Windows
full tests, and macOS race; downstream jobs were skipped. No native PASS is
claimed from that run, cross-compilation, or Wine.

## Community Preview publication preparation — 2026-09-06

The public launch documentation foundation is now in
[`docs/README.md`](../docs/README.md), with product, technical and release
playbooks under [`docs/community-preview`](../docs/community-preview). The
planned destination is `IAPro-Community/nexus`, but that public repository does
not exist yet and the local GitHub token is invalid. Do not rewrite installer or
update URLs until the destination is created or a transfer is explicitly
authorized. Next external action: re-authenticate `gh`, confirm repository
ownership/slug, then push the preserved candidate and inspect the exact-SHA CI.

Required next external gate: publish the preserved candidate through the
authorized branch workflow, then inspect the same-SHA native Windows/macOS,
Desktop, Browser, Axe, Visual and GoReleaser results. Do not declare GO before
those checks and the installer trust-root decision are complete.

## Next action

O cartão de diagnóstico em Configurações já usa o relatório read-only compartilhado
com `nexus doctor`; o próximo gate é evidência nativa Windows para ConPTY, startup,
Named Pipe e Job Objects, além do race/runtime nativo macOS. O build/embed Web
local já passou; o bloqueio restante é obter o mesmo resultado no CI do candidato
publicado.

Execute `nexus agy --help`, `nexus codex --help` ou `nexus claude --help` para conferir
a ajuda fusionada com aliases canônicos; execute `nexus usage` para quotas por janela
e `nexus web` para o Web Workspace OS local.

Correção mais recente: o Dialog de fechamento não aparece mais ao abrir a tela;
ele só é renderizado após uma ação explícita de fechar terminal ou Project Shell.
O servidor foi reiniciado com o bundle corrigido em `http://127.0.0.1:3000`.

O modo Plan foi removido dos controles de terminal e do cadastro de Agentes;
planejamento deve ocorrer no Composer/Flow ou ser enviado como instrução ao
Agente. O fluxo Safe/YOLO não encaminha mais `--plan` ao Codex. Os controles
“Assumir digitação”/“Liberar digitação” continuam ativos porque correspondem
ao lease de escrita do WebSocket do PTY compartilhado.

Build atual instalado em `/home/desenvolvedor/.local/bin/nexus` e servidor ativo
em `http://127.0.0.1:3000` (Bootstrap gerado no processo atual).

Os botões manuais de lease foram removidos do terminal. O WebSocket solicita o
lease automaticamente ao conectar; “Somente leitura” aparece apenas quando
outro acesso já possui a escrita. A barra lateral agora sempre oferece
“Visão geral”, além de Composer, Flow Runs, Uso, Sessões, Projetos e
Configurações, mesmo quando o layout salvo abre em terminais.

## Project picker and 1366×768 handoff — 2026-09-06

O cadastro de projetos usa o seletor nativo do sistema quando executado no Desktop
Wails. No navegador, ou quando a capacidade não é confirmada pelo backend, o modal
HTML continua disponível. A listagem HTML agora é leve por padrão e reutiliza
diretórios visitados durante a sessão; erros podem ser repetidos sem perder o contexto.

O modal foi ampliado e recebeu layout responsivo; o shell ganhou compactação para
1366×768 sem reduzir controles abaixo de 32px. O próximo passo recomendado é validar
visualmente no notebook alvo e, se ainda houver pressão vertical, experimentar o preset
de densidade `compact` antes de reduzir mais a barra lateral.

## Auth and installer follow-up — 2026-09-06

O problema de tela “Session Expired or Unauthorized” no `nexus-desktop` foi rastreado
à chamada de API cross-origin criada pelo uso da URL loopback a partir do WebView Wails.
As chamadas do bootstrap agora permanecem same-origin e usam o token Bearer no handler
embutido; há teste de regressão cobrindo esse contrato.

O instalador agora instala o `nexus-desktop` e cria lançador no Desktop quando a release
publica o artefato nativo com checksum. Para releases antigas sem esse artefato, o CLI
é instalado normalmente e o aviso explica a ausência; publique uma nova release para
ativar a instalação nativa automática.

## AGY model routing — 2026-09-06

O launcher agora resolve o modelo AGY antes de iniciar o processo: Gemini é a primeira
família quando sua cota local confiável está disponível; Claude Sonnet é fallback quando
Gemini está esgotado. A seleção é explícita (`--model`), evitando que o default interno
do AGY escolha Claude/GPT antes da hora. Configurações com modelo explícito continuam
com prioridade. Binários recompilados e instalados em `/home/desenvolvedor/.local/bin`.

## CI and Windows installer — 2026-09-06

O CI remoto está sendo acionado, mas a execução mais recente falhou antes dos jobs
dependentes: formato frontend, testes Go Windows e race test macOS. O workflow agora
permite execução manual e preserva logs dos testes de plataforma em artefatos. As
alterações locais de workflow só serão exercitadas no GitHub depois de commit/push ou
atualização da PR; este agente não cria commits automaticamente.

No Windows, o caminho explícito `-BuildFromSource` tenta instalar Go via WinGet quando
o compilador não existe. O caminho normal de release continua usando apenas o binário
assinado/checksum e não instala Go sem necessidade.

## Workspace tabs and xterm — 2026-09-06

O clique nas abas de produto agora navega pela URL do projeto e ativa a superfície no
mesmo evento. Isso impede o retorno de Terminal para Visão geral causado pelo sincronizador
de rota. Os terminais PTY também não escrevem nem fazem `fit` enquanto o painel está oculto;
a saída pendente é aplicada quando a aba fica ativa, evitando `Viewport.dimensions` indefinido.

## Quota fallback genérico — 2026-09-06

O pipeline de uso agora preserva a última observação válida quando a atualização de
qualquer CLI falha. O snapshot é marcado como `ESTIMATED`, recebe uma mensagem de
diagnóstico e não é tratado como cota atual. A seleção automática só considera
`LIVE`/`CACHED` como evidência fresca e prefere essas leituras às estimadas.
O launcher instalado em `/home/desenvolvedor/.local/bin/nexus` foi recompilado e
validado com as duas contas AGY.
## Autopilot orchestration checkpoint — 2026-09-06

The existing Nexus Composer/Flow/Mission Runner foundation was preserved. The concrete
false-success gap found in audit was that all package reviews could complete a Run
without a persisted global Definition of Done gate. `GlobalVerificationCommands`,
`VERIFYING`, persisted global results and fail→reopen→repair→global-pass behavior are
now implemented and tested. Read `docs/nexus-autopilot-architecture.md`,
`docs/nexus-maestro-orchestration-gap-analysis.md` and
`docs/nexus-autopilot-validation.md`.

Do not claim total product completion yet: provider-authenticated overnight execution
and native Windows/macOS process recovery require their respective environments. Next
action is to execute those scenarios and attach evidence; no commit/push was made.

## CI hardening — 2026-09-07

O checkout contém a correção local para os failures do SHA `1899ca6`: identidade
de filesystem, recência determinística, Codex shared-host, SessionHost/ConPTY
Windows, environment UTF-16, `processAlive`, JSON/PATH portáveis e frontend.

Go normal/race/vet/lint, frontend `bun run verify`, compilação cruzada de todos os
pacotes para Windows amd64/macOS arm64, GoReleaser snapshot e Wails Linux passaram.
Após publicar as alterações, reexecute o workflow e confirme a matriz nativa; não
há commit ou push automático nesta sessão.

## Quota alert deduplication — 2026-09-07

O monitor não usa mais o texto variável de contagem regressiva do reset na
identidade da janela. Isso elimina alertas repetidos para a mesma quota Gemini
abaixo do limiar quando apenas o countdown muda. Regressão normal e race passaram.
## Composer best-prompt handoff — 2026-09-07

Composer-only slice completed without touching Flow internals. New domain
contracts and migration are in `internal/nexus/composer*.go`,
`internal/nexus/store/composer.go`, `internal/nexus/store/migrations/0014_composer_v2.sql`
and `internal/nexus/prompt_compiler.go`. Contract details and known gaps are
in `DEV/SPECS/COMPOSER_FLOW_IMPLEMENTATION.md` and
`docs/nexus-composer-validation.md`.

Do not claim the full approved plan complete yet: remaining work includes
provider/local federated skill adapters, all-mutation revision enforcement,
context redaction/budgets, Composer UI cleanup and dedicated E2E scenarios.
## Sessão de decisões Luna

As decisões de quota sem fabricação, agrupamento visual com credenciais
separadas, prioridade configurável, supervisão opt-in e limites de continuidade
estão registradas em
[`DEV/DECISIONS/NEXUS_TERMINAL_CONTINUITY.md`](DECISIONS/NEXUS_TERMINAL_CONTINUITY.md).
O status atual continua parcial: gates locais passaram, mas a promoção oficial
aguarda matriz nativa macOS/Windows e CI remoto no mesmo SHA.

## Maestro catalog follow-up — 2026-09-07

The Maestro surface now reports the merged global + active-profile catalog.
Local runtime smoke confirmed 53/53 skills (the previous profile-only view was
48). Cards expose and copy the complete `SKILL.md` usage context, with a
generated fallback only when the source file is unavailable. Preserve merge by
stable skill ID and prompt provenance when changing discovery.

## Desktop/Web terminal continuity — 2026-09-07

O Desktop agora tenta anexar ao Core Web loopback já em execução antes de criar
um Core próprio. O fluxo reutiliza o bootstrap autenticado persistido pela Web,
cria uma sessão Desktop válida e usa um reverse proxy no AssetServer Wails para
REST e WebSocket. Isso mantém o mesmo registry, runtime e host PTY usados pelo
navegador. Se a Web não estiver ativa ou o estado não puder ser validado, o
fallback continua sendo o Core embutido do Desktop.

Evidência local: testes de anexação/proxy passaram; `go test ./...`, `go vet
./...`, race focado e `npm run verify` passaram; `make build-desktop-wails`
gerou `cmd/nexus-desktop/build/bin/nexus-desktop`. Para validação operacional,
reinicie a Web, abra primeiro um terminal no navegador, abra o binário Desktop e
confirme que o mesmo terminal aparece e continua emitindo saída. Windows/macOS
seguem sem smoke nativo neste ambiente.

## Desktop navigation continuity — 2026-09-07

O handler do Desktop foi refinado: rotas SPA como `/p/<project>/terminals`
agora recebem o `index.html` embutido localmente; o proxy para o Core Web fica
restrito às rotas `/api/*` e WebSocket. Isso evita que uma troca de tela faça a
WebView carregar um documento remoto e perca o estado do workspace.

O novo binário foi gerado em `cmd/nexus-desktop/build/bin/nexus-desktop`, a
instância antiga foi encerrada e a nova está ativa junto da Web em `127.0.0.1:3000`.
O teste E2E confirmou a troca Overview → Terminais; a execução inicial ainda
registrava uma falha responsiva no menu de criação em viewport 320px, corrigida
no fechamento local abaixo.

## Review closure — 2026-09-07

O bloqueio responsivo foi corrigido: o status do topbar não invade mais o
contexto em 320px e o botão Criar permanece fisicamente acionável. Também foram
corrigidos o deadlock de captura stdout, a comparação cross-platform de paths,
o readiness determinístico das fixtures de terminal, o diagnóstico do Browser
E2E, o bug de loop de loading por dependência de objetos em
`NexusWorkspaceApp`, o gate do bundle macOS e os gates documentais.

Estado local comprovado: frontend 10/10, 310 testes, Browser E2E/Axe PASS,
Go test/vet PASS, docs-verify PASS, Wails Linux build PASS e VIS-011 capturado
com Wails/Core reais. O relatório publicado permanece deliberadamente NO-GO
até o CI nativo Windows/macOS, GoReleaser snapshot e a integração final com
`origin/main` ocorrerem no mesmo SHA. Não fazer commit/push automático.

## Project Shell terminal overlay — 2026-09-07

O terminal do Project Shell apresentava uma linha de `W` porque o textarea
auxiliar do xterm estava visível: o build final não continha o stylesheet
oficial de `xterm`. O PTY e o WebSocket estavam corretos. `web/scripts/build.mjs`
agora incorpora o CSS após o esbuild, e o relatório de verificação exige o
marcador de isolamento do input.

Após recompilar e reiniciar a Web, o smoke real confirmou helper invisível,
terminal sem `W` espúrio e sem erros de console. A Web ativa continua em
`127.0.0.1:3000`; depois de mudanças no frontend, use `make build` e reinicie
somente a instância Nexus dessa porta para validar a tela de Terminais.

Na validação seguinte, a porta estava servindo o executável global antigo em
`/home/desenvolvedor/.local/bin/nexus`, apesar de o build correto existir na
worktree. Essa instância foi encerrada e substituída por `./nexus web --no-open`.
O smoke novo contra `127.0.0.1:3000` confirmou o helper invisível (`0x0`,
`opacity: 0`) e um terminal montado sem o overlay de `W`.

## Espaçamento semântico — 2026-09-07

A escala `--nx-spacing-*` foi restaurada em `workspace-os.css`, com aliases
`--nx-space-*` para compatibilidade. O binário ativo em `127.0.0.1:3000` foi
regenerado após `make build`; para visualizar a alteração no navegador, faça
hard refresh (`Ctrl+Shift+R`).

Nota operacional: neste ambiente `HOME` pode apontar para um perfil do Codex.
Para atualizar a instância usada pelo serviço, instalar explicitamente com
`LOCAL_BIN=/home/desenvolvedor/.local/bin make install-local`; `make build`
sozinho não substitui o executável global.
