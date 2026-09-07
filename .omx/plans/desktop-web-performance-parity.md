# Plano de correção — paridade de desempenho Web/Desktop

## Objetivo

Medir, explicar e eliminar a diferença perceptível de desempenho entre a superfície Web e o Nexus Desktop, preservando a arquitetura de um único Core Go, um único frontend React e os mesmos contratos REST/WebSocket.

O trabalho deve distinguir quatro classes de problema: inicialização, latência de API/transporte, renderização/interação e consumo sustentado de CPU/memória. Nenhuma otimização entra sem uma medição que identifique a fronteira responsável.

## Status de execução — 2026-09-07

Aplicadas as primeiras correções: seleção do Flow separada da reconstrução do
grafo, `applyNodeChanges` para alterações de nodes, coalescing de bootstrap
Desktop e batching de saída do terminal por frame. Frontend 10/10, Go tests,
race/vet relevantes, build Wails Linux de produção e diff check passaram.
Benchmark comparativo, métricas de FPS/GPU/latência e smoke nativo continuam
necessários antes de declarar a paridade completa.

## Evidência atual e limites

- O Desktop inicia um Core próprio, aguarda `core.Ready()`, carrega assets, cria a sessão e somente então inicia o Wails (`cmd/nexus-desktop/main.go:20-110`). Isso pode afetar cold start.
- O frontend Desktop pode aguardar o binding Wails por até 20 tentativas de 50 ms e depois consulta bootstrap/capabilities (`web/src/platform/desktopBridge.ts:70-113`). Isso pode adicionar até 1 s ao startup quando a injeção do binding atrasa.
- A tela inicial permanece no splash até `initSession()` concluir (`web/src/app/NexusWorkspaceApp.tsx:93-126`); no Desktop há bootstrap e uma verificação adicional de `/api/v1/session` (`web/src/api.ts:59-103`).
- Web e Desktop usam o mesmo bundle React e o mesmo Core; a diferença estrutural principal é o motor da WebView nativa (`docs/architecture/ADR-desktop-wails.md:9-21`).
- O Linux Desktop usa WebKitGTK, enquanto a Web normalmente roda em Chromium/Firefox. O Desktop Linux ainda está marcado como `native smoke pending` (`docs/platform/PLATFORM_SUPPORT_MATRIX.md:5-12`).
- O terminal processa cada mensagem WebSocket, atualiza o xterm e agenda `fit`/redraw via RAF/ResizeObserver (`web/src/nexus/AgentTerminal.tsx:520-680`).
- O canvas reconstrói todos os nodes e edges sempre que `flow` ou `selectedId` muda e procura cada alteração com `find`, criando custo potencial O(n²) durante drag (`web/src/features/work/FlowCanvas.tsx:53-97`).
- O bundle atual é minificado e code-split (`web/scripts/build.mjs:64-80`); portanto, “build de desenvolvimento” não deve ser presumido sem conferir o binário executado.

## Critérios de aceitação

As medições devem usar a mesma máquina, mesmo projeto, mesmo banco/estado, mesmo Git SHA, bundle de produção e cinco execuções por cenário. Registrar mediana e p95, descartando apenas a primeira execução quando o cenário for explicitamente warm.

1. Cold start Desktop, do início do processo até a UI interativa: mediana <= 2,5 s e p95 <= 4 s na máquina Linux de referência.
2. Warm start Desktop: mediana <= 1,5 s e p95 <= 2,5 s.
3. Diferença Desktop/Web para navegação entre Overview, Terminal, Work e Settings: p95 Desktop <= 1,20x o p95 Web e nenhum long task > 100 ms causado pela aplicação.
4. API REST local: p95 Desktop <= Web + 20 ms para os mesmos endpoints; erro e retry rate iguais a zero no roteiro nominal.
5. Terminal com replay controlado de 1 MiB e rajadas de 200 mensagens/s: nenhum congelamento > 100 ms, input echo p95 <= 50 ms e frames >= 50 FPS durante a rajada na máquina de referência.
6. Flow com 20, 50 e 100 nodes: seleção p95 <= 50 ms, drag p95 frame <= 20 ms e nenhum recálculo integral do grafo por alteração de posição.
7. Após 30 min ocioso com uma sessão e após 15 min de terminal ativo: CPU Desktop ociosa <= 2% em média, crescimento de RSS <= 15%, zero crescimento contínuo de timers, observers, sockets ou goroutines.
8. Sem regressão funcional, visual, de acessibilidade, autenticação, REST, WebSocket ou terminal; `make web-verify`, testes Go relevantes, `make build-desktop-wails` e smoke Desktop passam.
9. Linux recebe evidência nativa obrigatória; Windows/macOS permanecem `UNVERIFIED` até rodarem o mesmo roteiro em máquinas/CI nativos.

Os limites podem ser recalibrados uma única vez após o baseline, desde que a mudança seja registrada antes das correções e não relaxe a meta de paridade relativa de 1,20x.

## Estratégia

### Fase 0 — Congelar a comparação

1. Registrar SHA, versão, SO, compositor gráfico, GPU/driver, versões de WebKitGTK, navegador, Wails e resolução/escala.
2. Construir Web e Desktop a partir do mesmo `web/dist` usando `make web`, `make build` e `make build-desktop-wails` (`Makefile:19-25`, `Makefile:92-120`).
3. Confirmar por hash que `web/dist` e o conteúdo embutido são idênticos; usar o gate de embed-sync já existente.
4. Encerrar apenas instâncias Nexus criadas pelo roteiro e iniciar uma superfície por vez, evitando concorrência de dois Cores, monitores ou terminais.
5. Criar fixtures determinísticas: projeto pequeno, projeto real, terminal idle, replay de 1 MiB, burst de 200 mensagens/s e flows de 20/50/100 nodes.

Saída: manifesto do ambiente e roteiro reproduzível. Sem isso, não avançar para conclusões causais.

### Fase 1 — Instrumentar startup e prontidão

1. Criar um pequeno pacote de métricas de startup em `internal/desktop/`, usando relógio monotônico e eventos estruturados sem tokens, paths sensíveis ou payloads.
2. Marcar no Desktop: processo iniciado, `NewCore` concluído, `Core.Start` iniciado, `Core.Ready`, assets disponíveis, sessão criada, `wails.Run`, `OnStartup` e frontend pronto.
3. No frontend, marcar: script avaliado, bridge escolhido, binding Wails disponível, bootstrap recebido, capabilities recebidas, `/api/v1/session` concluído, primeiro React commit e `nexus:app-interactive`.
4. Expor as marcas apenas em log diagnóstico/trace opt-in e no E2E; produção não deve enviar telemetria externa.
5. Adicionar correlation ID local por inicialização para unir Go e frontend.

Arquivos principais: `cmd/nexus-desktop/main.go`, `internal/desktop/app.go`, `web/src/index.tsx`, `web/src/platform/desktopBridge.ts`, `web/src/api.ts`, `web/src/app/NexusWorkspaceApp.tsx`.

Teste: evento final só é emitido depois de sessão pronta e UI interativa; dados sensíveis nunca aparecem no payload.

### Fase 2 — Criar benchmark comparativo automatizado

1. Adicionar um harness Playwright para Web e um smoke/driver Desktop compatível com Wails no Linux. O mesmo arquivo de cenários deve dirigir ambas as superfícies.
2. Coletar Navigation/Performance Timeline, long tasks, RAF/frame time, React Profiler em build instrumentado, tempos REST, contagem/volume WebSocket, CPU/RSS do processo e filhos, WebKit WebProcess e goroutines do Core.
3. Separar cold e warm start; não misturar cache frio e quente.
4. Repetir cinco vezes e gerar JSON + relatório Markdown com mediana, p95, delta absoluto e razão Desktop/Web.
5. Integrar um comando local, por exemplo `make perf-desktop-compare`, sem colocá-lo inicialmente no gate rápido de CI.

Arquivos principais: novo `web/scripts/performance-compare.mjs`, fixtures em `web/scripts/fixtures/`, helper Go em `internal/desktop/perfdiag/` ou pacote equivalente, `Makefile`, relatório em `DEV/validation/`.

Gate: o relatório precisa apontar a primeira fronteira lenta; resultado apenas “Desktop parece lento” é inválido.

### Fase 3 — Árvore de decisão por causa

Executar as trilhas abaixo somente quando a medição correspondente falhar.

#### 3A. Startup/Core

1. Medir quanto do tempo está antes de `Core.Ready()` (`cmd/nexus-desktop/main.go:21-46`).
2. Se o Core for dominante, perfilar inicialização de banco, registry, quota monitor, provider discovery e recuperação de sessões; mover somente trabalho não crítico para depois da prontidão.
3. Definir prontidão mínima: listener, storage essencial, auth e handler disponíveis. Monitor de quota, descoberta pesada e manutenção podem iniciar em background com estado explícito.
4. Não mostrar UI “pronta” antes de as APIs críticas responderem; usar readiness por capacidades, não um sleep.
5. Testar falha parcial e shutdown durante inicialização para evitar goroutines/processos órfãos.

#### 3B. Bootstrap/autenticação Desktop

1. Medir o retry de binding (`desktopBridge.ts:75-81`) e a verificação extra de sessão (`api.ts:75-92`).
2. Se o binding normalmente chega depois do primeiro frame, substituir polling fixo por uma promessa/evento de bridge pronto ou bootstrap injetado pelo shell, mantendo fallback HTTP bounded.
3. Carregar capabilities fora do caminho crítico quando nenhuma capability for necessária para a primeira tela; cachear a mesma promise.
4. Evitar uma segunda consulta de sessão quando a sessão criada pelo Core puder ser verificada localmente com contrato equivalente e seguro; preservar expiração/rotação e fail-closed.
5. Adicionar timeouts explícitos e erro observável; não mascarar lentidão com splash infinito.

#### 3C. Asset loading/bundle

1. Verificar waterfall do `bundle.js`, CSS, chunks e imagens nas duas superfícies.
2. Confirmar cache headers e MIME corretos no `assetserver`/handler Wails.
3. Medir custo de parse/compile/execute, não apenas bytes transferidos.
4. Se o main chunk for dominante, lazy-load das superfícies não iniciais, terminal/xterm e Flow/ReactFlow; preservar uma tela inicial mínima.
5. Otimizar imagens apenas se aparecerem na rota crítica; não trocar formato sem teste de compatibilidade das três WebViews.

#### 3D. WebView/GPU/estilos

1. Registrar se WebKitGTK usa aceleração de hardware e se há fallback para software; capturar warnings do WebProcess/compositor.
2. Comparar o mesmo bundle em Chromium e WebKitGTK fora do Wails. Se WebKitGTK isolado reproduzir o delta, classificar como motor/renderização, não backend.
3. Usar paint flashing/layer borders para localizar repaint de áreas grandes, blur, sombras, transparência, filtros e animações.
4. Reduzir efeitos caros somente nos componentes comprovados; respeitar temas, tokens e `prefers-reduced-motion`.
5. Manter workaround de GPU por plataforma atrás de capability/feature flag e documentar rollback; não desabilitar aceleração globalmente como correção padrão.

#### 3E. React/workspace

1. Perfilar commits de `NexusWorkspaceApp`, providers e `WorkspaceRenderer` durante navegação e terminal ativo.
2. Isolar contextos de alta frequência para que título/atenção de um terminal não renderize todo o workspace. O batching atual de 80 ms em `PtyLiveChromeContext.tsx:54-92` deve ser medido antes de ser alterado.
3. Estabilizar props/callbacks e aplicar memoização apenas onde o profiler mostrar commits evitáveis.
4. Pausar efeitos, observers e polling de superfícies ocultas; preservar processamento necessário de eventos em background sem renderizá-los.
5. Adicionar teste de contagem de renders/commits nos fluxos críticos.

#### 3F. Terminal/xterm/WebSocket

1. Medir tamanho/frequência das mensagens, tempo de JSON parse, `scrubProtocolOutput`, `ingestOutput`, `term.write`, fit e scroll (`AgentTerminal.tsx:549-657`).
2. Introduzir fila por terminal e flush por RAF com limite de bytes/tempo, aplicando backpressure sem perder ordem ou UTF-8.
3. Evitar `fitAndResize` duplicado entre open, redraw pulse, `ResizeObserver`, ativação de painel e visibility change; deduplicar em um scheduler de frame.
4. Não chamar `scrollToBottom` quando o usuário está com scrollback ou quando não houve alteração visual relevante.
5. Manter superfícies ocultas sem `fit`, foco ou escrita visual; limitar buffer pendente e definir política para overflow.
6. Testar ANSI fragmentado, Unicode multibyte, rajadas, reconexão, troca de runtime, lease CONTROL/VIEW_ONLY e aba oculta.

#### 3G. Flow/ReactFlow

1. Separar reconstrução estrutural de nodes/edges da mudança de seleção (`FlowCanvas.tsx:58-86`).
2. Substituir o `changes.find` por mapa/index ou pelo helper oficial `applyNodeChanges`, evitando O(n²) no drag (`FlowCanvas.tsx:88-97`).
3. Estabilizar `nodeTypes`, callbacks e data; não executar `fitView` em toda mudança.
4. Desativar animação de edges, MiniMap ou detalhes apenas por limiar medido e com degradação explícita para grafos grandes.
5. Validar 20/50/100 nodes com snapshot funcional e frame-time.

#### 3H. API/polling/Core sustentado

1. Inventariar todas as consultas periódicas, seus owners e comportamento com `document.hidden`/superfície inativa.
2. Correlacionar request duplicada com renders e múltiplas instâncias do Core.
3. Consolidar polling por recurso em um único owner, cancelar requests obsoletas e preferir evento/WebSocket onde já existe fonte canônica.
4. Perfilar endpoints lentos no Go e adicionar índices/cache apenas com evidência.
5. Confirmar que quota monitor e provider probes não executam em duplicidade entre Web e Desktop e não bloqueiam readiness.

### Fase 4 — Implementação incremental e regressões

1. Ordenar correções pelo maior impacto medido: P0 startup/interatividade, P1 terminal/navegação, P2 Flow, P3 consumo sustentado.
2. Entregar um patch por causa raiz, cada um contendo baseline, mudança, teste de regressão e resultado comparativo.
3. Manter feature flags somente para mudanças de WebView/GPU ou comportamento com risco multiplataforma; remover flags depois da validação nativa.
4. Não trocar Wails, duplicar frontend, criar backend Desktop paralelo ou migrar framework durante esta campanha.

### Fase 5 — Validação nativa e promoção

1. Linux: executar matriz obrigatória em WebKitGTK com pelo menos GPU Intel/AMD disponível na máquina de referência, escala 100% e escala fracionária quando suportada.
2. Windows: executar em WebView2 nativo, incluindo Terminal/ConPTY; não aceitar cross-build como evidência de desempenho.
3. macOS: executar em WKWebView nativo, Intel ou Apple Silicon conforme matriz suportada.
4. Repetir cold/warm, navegação, terminal, Flow, idle e soak; anexar logs e relatório por SO.
5. Atualizar `docs/platform/PLATFORM_SUPPORT_MATRIX.md` somente após evidência nativa do mesmo SHA.

## Testes e gates

### Unitários

- Conversão e serialização das marcas de performance sem dados sensíveis.
- Scheduler/deduplicação de terminal por RAF.
- Backpressure e preservação de ordem/UTF-8.
- Atualizações de Flow em O(n) e sem reconstrução por seleção/drag.
- Bootstrap/capabilities com binding imediato, tardio, ausente e falho.

### Integração

- Core readiness versus tarefas pós-ready.
- Sessão Desktop criada, validada, rotacionada e expirada.
- REST e WebSocket no AssetServer Wails.
- Terminal ativo/oculto, reconexão e troca de runtime.

### E2E e desempenho

- Mesmo roteiro Web/Desktop com cinco repetições.
- 20/50/100 nodes, replay 1 MiB e 200 mensagens/s.
- 30 min idle e 15 min terminal ativo.
- Screenshot/Axe existentes para garantir que otimizações visuais não degradaram UI/a11y.

### Gates finais

```bash
make web-verify
go test ./internal/desktop ./internal/app ./internal/control/web ./internal/control/terminal/...
go test -race ./internal/desktop ./internal/control/web ./internal/control/terminal/...
make build-desktop-wails
make perf-desktop-compare
git diff --check
```

Antes de release, executar também `make quality` e o smoke nativo de cada plataforma promovida.

## Riscos e mitigação

- **O benchmark medir máquinas diferentes:** fixar ambiente e usar também razão Desktop/Web na mesma máquina.
- **Instrumentação alterar o resultado:** manter marcas leves, comparar build instrumentado e release, retirar React Profiler do artefato final.
- **Otimização esconder falha de autenticação:** bootstrap permanece fail-closed, com timeout e teste de expiração.
- **Batching do terminal aumentar latência:** limitar flush a um frame e manter métrica de input echo; rollback por flag durante validação.
- **Workaround Linux regredir Windows/macOS:** isolar por capability/plataforma e exigir smoke nativo.
- **Metas irreais em hardware lento:** preservar a meta relativa <= 1,20x e registrar baseline absoluto por classe de hardware.
- **Escopo virar reescrita:** manter Wails v2, Core único e frontend único; migração de framework exige ADR separado.

## Ordem recomendada de execução

1. Fase 0 + Fase 1: 1 pacote.
2. Fase 2: harness e baseline reproduzível.
3. Escolher no máximo duas trilhas da Fase 3 com maior contribuição ao p95.
4. Implementar e verificar cada causa separadamente.
5. Repetir o baseline; só então abrir a próxima trilha.
6. Executar soak e matriz nativa.
7. Atualizar documentação e critérios de suporte.

## Definição de concluído

O trabalho termina quando a diferença é reproduzível, cada regressão possui causa e teste, todos os critérios aplicáveis passam no Linux nativo, os gates de qualidade estão verdes e o relatório mostra Desktop dentro de 1,20x da Web nos fluxos críticos. Windows/macOS só podem ser declarados corrigidos após execução nativa; ausência dessa evidência deve permanecer explícita.
