# Plano — Nexus Composer como complemento do Orquestrador Maestro

> **SUPERSEDIDO:** este plano foi substituído por
> `2026-09-06-nexus-composer-best-prompt-product.md`, que corrige o escopo:
> reescrever o Composer, preservar o Flow existente e manter Copy/Agent/Flow
> como destinos independentes.

## Objetivo

Fazer o Composer produzir um plano de trabalho realmente bom para o Flow,
usando o Orquestrador Maestro como autoridade de método, skills, risco,
processo e verificação, sem copiar sua memória semântica, planner,
entrevistador, aprovação ou runner.

Este plano não redesenha o Flow. O Flow está sendo repensado por outro agente;
o trabalho aqui deve preservar uma fronteira de integração estável para receber
essa nova versão.

## Decisão arquitetural

O produto deve ter esta divisão:

| Responsabilidade | Dono |
|---|---|
| Projeto, agentes persistentes, perfis, recursos, quotas e execução local | Nexus |
| Sessão visual, contexto do projeto, edição do briefing e apresentação | Nexus Composer |
| Método, entrevista semântica, skills, processo, risco, gates e verificação | Orquestrador Maestro |
| Plano semântico, revisão/aprovação metodológica e requisitos de verificação | Maestro, via contrato versionado |
| Conversão do plano aprovado em Flow executável | Nexus, adaptador fino |
| DAG, assignment, preflight final, admission, execução, receipts e recovery | Flow/Mission Runner do Nexus |

O Composer é uma superfície de trabalho e revisão. Ele não é um segundo
Maestro, não é um segundo planner e não é um segundo runner.

## O que não deve ser refeito no Nexus

Não portar nem duplicar para Go/React os conceitos que já pertencem ao Maestro:

- SemanticPlanner e fallback semântico;
- AI/Dynamic Interviewer e ranking de perguntas;
- MissionBriefBuilder;
- PlanSemanticDiff, PlanRevision e PlanReview;
- GraphValidator e utilitários de plano semântico;
- PlanApprovalGate;
- SemanticRanker e ModelRouter;
- TaskLifecycleMonitor, LaneExecutor e ReadinessEvaluator;
- catálogo, semântica e definição de skills.

O inventário existente já classifica esses componentes como
`KEEP_IN_MAESTRO`: [NEXUS_V1_MAESTRO_WIP_SALVAGE.md](../../DEV/NEXUS_V1_MAESTRO_WIP_SALVAGE.md).

## O que o Nexus deve fazer

- manter a sessão, histórico de interação e estado visual do Composer;
- preparar um envelope de contexto seguro e limitado;
- mostrar ao usuário o briefing/plano recebido e suas alterações;
- permitir aceitar, editar, rejeitar e pedir revisão de decisões;
- armazenar versões e lineage sem se tornar dono da semântica do Maestro;
- traduzir o plano aprovado para o contrato do Flow;
- preservar objetivos, critérios, restrições, skills, riscos, dependências e
  verificações durante a tradução;
- executar somente pelo Flow/Mission Runner existente ou pelo novo contrato que
  o outro agente entregar;
- continuar funcional em `MAESTRO_DEGRADED`, sem inventar recomendações.

## Contrato de integração obrigatório

Antes de ampliar a UI do Composer, congelar um contrato versionado entre os
projetos:

### Request do Nexus para o Maestro

O request deve conter somente contexto seguro:

- `contract_version`;
- projeto, branch, HEAD e dirty fingerprint;
- linguagem/framework e paths relevantes;
- objetivo e prompt original;
- briefing atual do Composer;
- decisões confirmadas e perguntas abertas;
- sinais de risco;
- processo solicitado e modo Maestro;
- resumo de alterações, sem secrets, `.env`, OAuth, credenciais ou transcript
  bruto de terminal.

### Response do Maestro

O response deve ser estruturado e explicável:

- status e versão do Maestro;
- classificação do trabalho;
- risco e justificativa;
- brief/plano semântico proposto;
- perguntas priorizadas;
- skills `REQUIRED`, `RECOMMENDED` e `OPTIONAL`, sempre com motivo;
- gates e requisitos de verificação;
- dependências e ordem proposta;
- diff semântico em relação à revisão anterior;
- versão/hash do plano e origem das decisões;
- estado de degradação quando não houver Maestro.

O Nexus não deve reconstituir esses dados a partir de texto livre quando o
Maestro estiver disponível.

## Plano de implementação

### Fase 0 — Congelar a fronteira com o outro agente

Arquivos/contratos envolvidos:

- `DEV/NEXUS_V1_MAESTRO_INTEGRATION.md`;
- `docs/superpowers/specs/2026-09-01-nexus-canonical-product-contract.md`;
- contratos públicos do Flow em `internal/nexus/flow.go` e no modelo web.

Ações:

1. Registrar que o Composer termina em `ApprovedPlanCandidate`, não em um Flow
   proprietário.
2. Definir o payload mínimo que o Flow repensado aceitará.
3. Definir compatibilidade de versões e rejeição explícita de campos
   desconhecidos obrigatórios.
4. Não alterar o algoritmo, canvas ou runner do Flow neste plano.

Saída: documento de seam e fixtures JSON compartilhadas entre os dois agentes.

### Fase 1 — Auditoria e remoção de duplicação

Arquivos principais:

- `internal/nexus/composer.go`;
- `internal/nexus/composer_brief.go`;
- `internal/nexus/maestro.go`;
- `internal/nexus/maestrogates/`;
- `web/src/features/work/ComposerSurface.tsx`;
- `web/src/features/work/PlanBuilderSurface.tsx`.

Ações:

1. Mapear cada campo atual do `LivingBrief` para fonte: usuário, contexto,
   Maestro, inferência local ou desconhecido.
2. Marcar parsers heurísticos locais que devem permanecer apenas como fallback
   de degradação.
3. Remover qualquer recomendação, skill, processo ou gate sintético que possa
   parecer Maestro real.
4. Definir uma projeção de compatibilidade para sessões antigas, sem migração
   destrutiva.

Saída: matriz `Nexus field → Maestro field → Flow field → owner`.

### Fase 2 — Bridge Maestro v1 real

Arquivos principais:

- `internal/nexus/maestro.go`;
- novos tipos em `internal/nexus/maestrocontract/` ou pacote equivalente;
- handlers `/api/v1/maestro` e `/api/v1/maestro/advice`;
- testes de contrato.

Ações:

1. Implementar transporte local `stdin JSON → Maestro → stdout JSON`, conforme
   o contrato existente, com timeout e stderr separado.
2. Validar versão, capabilities e schema antes de consumir uma resposta.
3. Redigir contexto proibido antes do processo filho.
4. Preservar `MAESTRO_DEGRADED` com erro explícito e listas vazias.
5. Não bloquear trabalho direto, shell, Agent ou Composer básico quando o
   Maestro estiver ausente.
6. Registrar request/response redigidos e hash de contrato, nunca segredos.

Critério: o Nexus só exibe plano, skill ou gate como Maestro quando recebeu e
validou esse dado do adapter real.

### Fase 3 — Composer como editor de plano, não gerador concorrente

Arquivos principais:

- `internal/nexus/composer.go`;
- `internal/nexus/composer_brief.go`;
- store de Composer;
- `web/src/features/work/ComposerSurface.tsx`;
- `web/src/features/work/composerModel.ts` e testes.

Ações:

1. Manter `ComposerSession` como sessão e histórico de edição.
2. Adicionar revisão explícita do plano recebido do Maestro.
3. Mostrar por campo: valor, origem, confiança, status e revisão.
4. Exibir diff semântico entre revisão atual e anterior.
5. Tornar perguntas priorizadas pelo Maestro; usar fallback local apenas em
   degradação declarada.
6. No modo `EXISTING_PROMPT`, mostrar extração, lacunas, contradições e riscos
   antes de pedir aprovação.
7. Permitir `aceitar`, `editar`, `rejeitar` e `pedir revisão` sem alterar
   silenciosamente a revisão aprovada.
8. Separar claramente três ações:
   - salvar revisão;
   - aprovar plano para Flow;
   - executar Flow.

Não adicionar um novo planner no frontend. A UI deve editar e apresentar
estruturas recebidas, não inferir uma segunda verdade.

### Fase 4 — Adaptador Composer → Flow

Arquivos principais:

- `internal/nexus/plan.go`;
- `internal/nexus/flow.go`;
- modelo de request do Flow que o outro agente entregar;
- testes de materialização.

Ações:

1. Substituir a materialização genérica de “uma fase/um pacote” por uma
   tradução baseada no plano semântico aprovado.
2. Preservar ID/hash da revisão Maestro e do artefato Composer.
3. Traduzir somente conceitos suportados pelo Flow atual.
4. Recusar com diagnóstico claro recursos sem representação segura, em vez de
   reduzir silenciosamente um plano complexo a dois passos genéricos.
5. Manter o `WorkPlan`/`FlowDefinition` como façade compatível, sem segunda
   persistência nem segundo runner.
6. Entregar ao Flow critérios de aceite e verificação como dados executáveis,
   não apenas texto de prompt.

Esta fase só deve começar quando o contrato do outro agente estiver definido.

### Fase 5 — Interface e experiência do Composer

Arquivos principais:

- `web/src/features/work/ComposerSurface.tsx`;
- componentes novos apenas após pesquisa em `web/src/components/` e
  `web/src/design-system/primitives/`;
- SCSS Modules e `web/src/i18n/resources.ts`.

Ações:

1. Organizar a tela em quatro áreas: intenção, contexto, plano e aprovação.
2. Mostrar o status Maestro sem transformá-lo em gateway obrigatório.
3. Usar cartões de decisão para perguntas e riscos.
4. Mostrar a origem de cada decisão e a idade do contexto.
5. Remover strings hardcoded e estilos inline estáticos.
6. Garantir teclado, foco, leitura por screen reader e estados de erro.
7. Mostrar “Maestro indisponível” honestamente, sem plano falso.

### Fase 6 — Verificação e integração

Testes unitários:

- request redigido não contém secrets ou transcript bruto;
- resposta Maestro inválida é rejeitada;
- skill inexistente nunca aparece como disponível;
- modo degradado não inventa plano, skill ou gate;
- revisão aceita permanece imutável;
- diff semântico preserva origem e autoria;
- plano complexo não é reduzido silenciosamente;
- campos e critérios chegam ao Flow sem perda.

Testes de integração:

- Nexus chama Maestro real via fixture CLI;
- Composer retoma sessão após reinício;
- aprovação do plano cria somente um draft side-effect-free;
- nenhum Agent/runtime é criado antes de `Approve & Run`;
- Flow recebe a revisão exata aprovada;
- alteração no Composer cria nova revisão, sem mutar execução anterior;
- Maestro ausente mantém Direct Agent e Project Shell funcionando.

Testes de interface:

- prompt existente mostra análise antes da aprovação;
- usuário consegue revisar cada decisão;
- status degradado é acessível e localizado;
- plano aprovado e Flow são distinguíveis visualmente;
- a aba Composer não precisa ser aberta para manter execução já aprovada.

Gates:

- `go test ./...`;
- `go vet ./...`;
- `go test -race ./...`;
- `make web-verify`;
- browser E2E dos caminhos Composer → approve → Flow;
- fixture/contract test do Maestro;
- validação same-SHA nas plataformas suportadas quando a mudança atingir
  runtime ou empacotamento.

## Critérios de aceitação do produto

1. O Nexus nunca apresenta uma skill, gate, processo ou plano sem origem real
   no Maestro ou fallback explicitamente marcado.
2. O Maestro permanece opcional: sua indisponibilidade não quebra Agent,
   Shell, sessão direta ou execução já aprovada.
3. O Composer consegue transformar ideia ou prompt existente em uma revisão
   estruturada, explicável e aprovada.
4. Cada decisão tem origem, confiança, revisão e possibilidade de correção.
5. O plano aprovado chega ao Flow com critérios, restrições, dependências,
   skills e verificações preservados.
6. Nenhuma execução começa apenas por abrir, editar ou finalizar o Composer.
7. O Nexus não cria um segundo planner, segundo sistema de skills, segunda
   memória semântica ou segundo runner.
8. O Flow repensado pelo outro agente pode substituir sua implementação interna
   sem obrigar o Composer a ser reescrito, desde que preserve o contrato.

## Riscos e mitigação

| Risco | Mitigação |
|---|---|
| Composer e Maestro divergirem em semântica | Maestro é autoridade; Nexus usa adapter e schema versionado |
| Outro agente alterar o Flow durante a integração | Congelar seam, fixtures e contrato antes da Fase 4 |
| Degradação parecer sucesso | estados explícitos, origem obrigatória e fail-closed |
| Perda de dados na tradução | matriz de campos, hash, fixtures e testes de round-trip |
| Interface virar outro planner | UI só edita projeção; sem heurística paralela como fonte primária |
| Maestro bloquear trabalho direto | integração opcional e timeout; Direct/Agent/Shell independentes |
| Plano complexo virar prompt genérico | rejeitar campos sem mapeamento seguro, nunca inventar steps |

## ADR

### Decisão

O Composer do Nexus será uma camada de intake, contexto, edição, revisão e
aprovação de plano. O Maestro continuará sendo a autoridade semântica de
método, processo, skills, risco e verificação. O Nexus converterá o plano
aprovado em Flow por um adapter fino.

### Drivers

- não duplicar o Orquestrador Maestro;
- preservar o Flow que está sendo repensado por outro agente;
- manter execução, agentes e recursos sob autoridade do Nexus;
- garantir rastreabilidade e degradação honesta;
- permitir evolução independente dos dois projetos.

### Alternativas consideradas

1. Reimplementar planner e entrevista dentro do Composer: rejeitada por
   duplicação e divergência semântica.
2. Fazer o Maestro executar diretamente dentro do Nexus: rejeitada porque
   agentes, recursos, runtime e runner pertencem ao Nexus.
3. Manter o Composer como prompt builder atual: rejeitada porque perde plano,
   origem, revisão e integração real com o Maestro.
4. Contrato Maestro → Composer → Flow: escolhida.

### Consequências

- exige contrato versionado e fixtures compartilhadas;
- o Composer fica menor semanticamente, mas mais forte em UX e rastreabilidade;
- a integração precisa tolerar Maestro ausente ou em versão incompatível;
- o Flow poderá evoluir sem duplicar a tela de planejamento.

### Follow-ups

- obter a versão do contrato CLI v1 do repositório Maestro;
- alinhar com o agente que está repensando o Flow o payload de entrada;
- decidir se `ApprovedPlanCandidate` viverá no store do Composer ou como revisão
  do WorkPlan, sem criar nova autoridade concorrente;
- executar a matriz de contract tests antes da implementação visual ampla.
