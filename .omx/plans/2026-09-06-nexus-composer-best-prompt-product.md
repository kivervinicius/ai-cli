# Plano de Produto — Nexus Composer: do motivo ao melhor prompt

**Status:** proposta para aprovação  
**Escopo:** reescrita evolutiva do Composer e integração com o Flow existente  
**Fora do escopo:** reescrever Flow, WorkPlan, Mission Runner, Scheduler ou Maestro

## 1. Visão

O Nexus Composer será o ambiente no qual uma intenção incompleta se torna um
contrato de trabalho claro, contextualizado, verificável e pronto para um agente.

Ele deve descobrir não apenas **o que** o usuário pediu, mas por que importa,
para quem, qual resultado concreto é esperado, quais limites não podem ser
violados, como provar o sucesso e quais skills instaladas melhoram a execução.

O usuário nunca é obrigado a usar Flow. Em qualquer revisão útil, ele pode:

1. continuar aprofundando;
2. copiar o prompt;
3. enviar a um Agent existente;
4. criar uma nova sessão de Agent;
5. transformar a revisão aprovada em Flow;
6. salvar e retomar depois.

O Flow deve parecer valioso quando houver dependências, paralelismo, múltiplos
agentes, gates, longa duração ou necessidade de recovery. Essa recomendação é
explicável e nunca coercitiva.

## 2. Princípios

1. Motivação e resultado antes da solução.
2. Completude útil: aprofundar enquanto a resposta puder mudar escopo,
   arquitetura, risco, aceitação ou verificação.
3. Uma pergunta principal de maior impacto por rodada.
4. Fato, inferência, premissa e desconhecido são estados diferentes.
5. Recomendar somente skills comprovadamente instaladas/disponíveis.
6. Copy, Send to Agent e Turn into Flow são destinos de primeira classe.
7. Flow é recomendado por valor demonstrável, nunca por bloqueio artificial.
8. Maestro permanece autoridade de método, processo, risco, gates e skills
   canônicas; Nexus não duplica sua inteligência durável.
9. Contexto é limitado, redigido, versionado e rastreável.
10. Toda saída referencia uma revisão imutável do PromptArtifact.

## 3. Responsabilidades

| Capacidade | Autoridade |
|---|---|
| Conversa, edição, histórico, UX e destinos | Nexus Composer |
| Project, branch, HEAD, arquivos e DEV | Nexus |
| Interpretação pelo modelo configurado | Nexus Intelligence, capacidade interna |
| Método, processo, risco, gates e skills canônicas | Maestro |
| Skills adicionais instaladas | Adapter da ferramenta/provider, com origem explícita |
| PromptArtifact e variantes | Nexus Composer |
| DAG, decomposição, preflight e execução | Flow/Mission Runner existente |
| Agents, perfis, quotas e alocação | Nexus Scheduler |

### Proibido

- criar segundo Flow, planner de Flow ou runner;
- copiar catálogo/memória do Maestro para autoridade paralela;
- declarar skill disponível sem prova;
- instalar skills automaticamente durante o Composer;
- enviar secrets, `.env`, credenciais ou transcripts brutos;
- transformar prompt em Flow sem ação explícita;
- esconder Copy/Agent para forçar Flow;
- usar score de readiness como única prova de qualidade;
- finalizar silenciosamente com gaps bloqueantes.

## 4. Experiência do Composer

### Entrada

Oferecer três inícios simples:

- Descrever uma ideia;
- Melhorar um prompt existente;
- Começar a partir do projeto atual.

Classificar internamente o arquétipo: feature, bug, refactor, arquitetura,
pesquisa, UI, backend, segurança, migração, operação ou genérico.

### Ciclo de descoberta

Cada rodada executa:

```text
entrada
  → Context Snapshot
  → Motivation Map
  → Living Brief
  → conselho e skills aplicáveis
  → gaps priorizados
  → resumo das mudanças
  → próxima pergunta de maior impacto
```

A resposta separa:

- Entendi;
- Descobri no projeto;
- Inferi — confirme;
- Decisões tomadas;
- Ainda falta;
- Próxima decisão e por que importa.

O usuário pode responder, aceitar/editar uma sugestão, dizer “não sei”, delegar
uma decisão segura, dispensar a pergunta ou seguir para um destino.

### Motivation Map

Representar a cadeia:

```text
Motivação → Resultado → Requisito → Critério de aceite → Evidência
```

Campos: problema atual, afetados, motivo/impacto, resultado observável, urgência,
trade-offs e sinais de sucesso/fracasso.

Não repetir “por quê?” indefinidamente. A motivação converge quando permite
decidir prioridade, escopo e sucesso, ou quando o usuário escolhe parar.

### Living Brief

Cobrir conforme o arquétipo:

- intenção, motivação e resultado;
- público/atores e contexto atual;
- evidências e paths relevantes;
- entregáveis;
- requisitos funcionais e não funcionais;
- escopo e fora de escopo;
- stack, ambientes e compatibilidade;
- restrições técnicas, produto, segurança, compliance, prazo e orçamento;
- decisões, alternativas rejeitadas e dependências;
- riscos, premissas e unknowns;
- autonomia e ações proibidas;
- aceite, testes, revisão e verificação;
- observabilidade, rollback, documentação e definição de pronto.

Cada item contém `id`, `value`, `source_type`, `source_ref`, `confidence`,
`status`, `impact`, `revision` e `updated_at`.

Status: `PROPOSED`, `CONFIRMED`, `INFERRED`, `ASSUMED` ou `DISMISSED`.

### Convergência

Readiness usa gates separados: objetivo, motivação/resultado, escopo,
restrições críticas, aceite, verificação, risco, contexto fresco e skills.

Estados: `EXPLORING`, `ACTIONABLE_WITH_GAPS`, `READY`, `FINALIZED`, `STALE`.

`READY` exige ausência de gaps bloqueantes ou recomendados capazes de alterar
materialmente o resultado. Gaps opcionais permanecem visíveis. O usuário pode
sair antes confirmando uma única vez os gaps/riscos aceitos.

## 5. Skills do Maestro e outras instaladas

### Catálogo federado

Criar um `SkillCatalog` normalizado com adapters somente de leitura:

1. `MaestroSkillSource`: capabilities/manifest canônico;
2. `ProviderSkillSource`: capacidades comprovadas por Codex, Claude, Gemini,
   OpenCode, Cursor ou AGY;
3. `LocalCompatibleSkillSource`: manifests instalados em raízes suportadas.

Identidade: `(source, skill_id, version)`. Nomes iguais de fontes diferentes não
são fundidos silenciosamente.

Exibir nome, descrição, origem, versão, compatibilidade com o destino, motivo,
aplicabilidade, risco/permissões, prioridade e estado.

Prioridade: `REQUIRED`, `RECOMMENDED`, `OPTIONAL`.  
Estado: `SUGGESTED`, `ACCEPTED`, `REJECTED`, `UNAVAILABLE`, `APPLIED`.

### Recomendação e portabilidade

Combinar arquétipo, risco, stack, artefatos, destino e conselho real do Maestro.
Carregar corpo completo de skill apenas quando necessário e permitido.

- Send to Agent valida skills no provider/profile escolhido.
- Copy gera manifesto e alerta de portabilidade, sem presumir instalação externa.
- Flow recebe skills estruturadas e deixa o preflight existente validá-las.
- Maestro/provider indisponível nunca produz skill sintética.

## 6. Compilador do melhor prompt

Compilar da revisão estruturada, nunca da concatenação do chat.

Estrutura canônica:

1. papel/postura;
2. objetivo;
3. motivação e resultado;
4. contexto confirmado e fontes;
5. requisitos;
6. escopo e fora de escopo;
7. restrições e ações proibidas;
8. decisões e premissas;
9. abordagem ou liberdade de decisão;
10. skills e procedência;
11. entregáveis;
12. critérios de aceite;
13. testes e verificação;
14. observabilidade, rollback e documentação;
15. comportamento em bloqueio;
16. formato final e evidências exigidas.

Omitir seções inaplicáveis, deduplicar e nunca promover inferência a fato.

### Variantes

- `GENERIC_PORTABLE`: copiar para qualquer Agent, com manifesto de skills;
- `NEXUS_AGENT`: adaptada às capacidades do Agent escolhido;
- `FLOW_HANDOFF`: estrutura, prompt e lineage, sem criar DAG concorrente.

Todas apontam para o mesmo PromptArtifact base e registram target/capabilities.

### Rubrica de qualidade

Avaliar de 0–2: objetivo, motivação, contexto, escopo, consistência, aceite,
riscos/restrições, bloqueios, skills e concisão. O relatório mostra razões e
melhorias; score não substitui gates.

## 7. Tornar o Flow desejável

Calcular `FlowSuitability` por sinais objetivos:

- duas ou mais frentes paralelizáveis;
- dependências entre entregas;
- múltiplas especialidades/agentes;
- tarefa longa ou retomável;
- gates independentes;
- risco alto;
- worktree/isolamento;
- execução em ondas;
- receipts, checkpoints ou recovery.

Resultados:

- `DIRECT_FIT`;
- `FLOW_BENEFICIAL`;
- `FLOW_STRONGLY_RECOMMENDED`.

Exemplo de UI:

> **Flow recomendado**  
> Há 3 frentes paralelizáveis, 2 dependências e revisão independente. No Flow
> você acompanha cada etapa, retoma falhas e preserva evidências. Você ainda
> pode copiar ou enviar o prompt diretamente.

### Handoff

Usar `PromptArtifact → Flow` existente, evoluindo apenas payload/adapter quando
necessário. Não alterar canvas, runner, scheduler ou lifecycle nesta campanha.

Preservar IDs, revisão, hash, Motivation Map, brief, prompt, skills/origens,
aceite, verificações, restrições, riscos, contexto e hints de dependência.
Criar Flow Draft não inicia Agent, runtime ou MissionRun.

## 8. Interface

- Centro: conversa e pergunta ativa;
- lateral: brief, decisões, gaps e skills;
- cabeçalho: projeto, contexto, Maestro e revisão;
- barra persistente: continuar, copiar, Agent, Flow e salvar;
- revisão final: prompt, diff, qualidade, gaps e destinos.

Interações: quick replies, “não sei”, “sugira”, “decida com segurança”, “não se
aplica”, edição com histórico, confirmação de inferência, diff, fonte, decisão
de skill, preview por destino e escolha de Agent compatível.

Usar componentes existentes, i18n, SCSS Modules, HTML semântico, teclado e
WCAG 2.2 AA. Decompor o atual `ComposerSurface.tsx` por responsabilidade.

## 9. Domínio e persistência

Preservar tabelas atuais com migração aditiva:

- `ComposerSession`;
- `ComposerTurn`;
- `ComposerRevision` imutável;
- `ComposerFact` com provenance;
- `ContextSnapshot`;
- `SkillRecommendation`;
- `PromptArtifact`;
- `PromptVariant`;
- `ComposerDestinationReceipt`;
- `FlowSuitabilityAssessment`.

Não persistir secrets, bodies integrais de skills sem necessidade, transcripts
ou arquivos completos de contexto. Sessões antigas usam adapter de
compatibilidade sem migração destrutiva.

## 10. API

Preservar rotas atuais e evoluir contratos:

- criar/obter/listar ComposerSession;
- adicionar turn;
- decidir campo/inferência;
- decidir skill;
- revisar/finalizar;
- gerar PromptVariant;
- registrar Copy;
- enviar para Agent;
- materializar no Flow existente.

Toda mutação usa `expected_revision`. Erros distinguem validation, stale
context, skill unavailable, provider unavailable, Maestro degraded e revision
conflict.

## 11. Fases de implementação

### Fase 0 — Baseline e contrato

Arquivos: contrato canônico, `DEV/SPECS/COMPOSER_FLOW_IMPLEMENTATION.md`, APIs
atuais e fixtures Flow.

1. Registrar formalmente que o Flow não será reescrito.
2. Inventariar Composer, PromptArtifact, Agent Ask e handoff existentes.
3. Congelar fixtures de compatibilidade.
4. Confirmar payload aceito pelo Flow atual.

Gate: testes atuais verdes e fixture registrada.

### Fase 1 — Domínio versionado

Arquivos: `internal/nexus/composer*.go`, store/migrations, `web/src/types.ts`.

1. Introduzir Motivation Map, provenance, revisions, Context Snapshot e receipts.
2. Preservar leitura legada.
3. Validar enums/status no backend.
4. Usar revisão otimista e imutabilidade após finalização.
5. Marcar STALE quando contexto mudar.

Gate: round-trip SQLite, conflito de revisão e upgrade de fixture legado.

### Fase 2 — Catálogo federado de skills

Arquivos: `internal/nexus/maestro.go`, `maestrogates/`, novo pacote de skills e
adapters de provider.

1. Definir `SkillSource` e catálogo normalizado.
2. Usar Maestro como fonte canônica quando disponível.
3. Integrar manifests/capabilities reais das ferramentas.
4. Validar compatibilidade por destino.
5. Persistir decisão/procedência, não catálogo paralelo.
6. Manter degradação honesta e nenhuma instalação automática.

Gate: zero skill sintética e zero aplicação incompatível.

### Fase 3 — Descoberta e convergência

Arquivos: Composer backend, brief, Intelligence e testes.

1. Separar classificação, extração, motivação, gaps, ranking e resposta.
2. Consultar Maestro para método e Intelligence para interpretação.
3. Produzir pergunta principal com rationale e quick replies.
4. Impedir repetição por fingerprint.
5. Implementar convergência e saída antecipada.
6. Narrar mudanças concretas em cada rodada.
7. Analisar prompt existente com lacunas, contradições e diff.

Gate: corpus de ideia, prompt, bug, feature, pesquisa, segurança e migração sem
loops de perguntas.

### Fase 4 — Contexto rastreável

Arquivos: readiness/contexto, Intelligence provider e store de snapshots.

1. Montar contexto por relevância e orçamento.
2. Registrar fonte, fingerprint e idade.
3. Permitir incluir/excluir fontes.
4. Redigir dados proibidos antes de adapters externos.
5. Invalidar inferências afetadas por contexto novo.

Gate: redaction, budget, STALE e provenance testados.

### Fase 5 — Compilação e qualidade

Arquivos: compiler atual, Composer e testes golden.

1. Compilar estrutura canônica.
2. Gerar variantes Generic, Agent e Flow.
3. Implementar rubrica explicável.
4. Deduplicar, detectar contradição e preservar gaps aceitos.
5. Versionar contexto, skills e capabilities.
6. Criar corpus golden com revisão humana inicial.

Gate: compilação determinística; nenhum critério confirmado perdido; nenhuma
inferência promovida a fato.

### Fase 6 — Destinos e FlowSuitability

Arquivos: Composer/handlers, `internal/nexus/plan.go` somente no adapter,
`web/src/nexus/api.ts` e Agent Ask.

1. Copy com variante portátil e receipt.
2. Send to Agent com alvo explícito e skill validation.
3. New Agent sem efeito antes da confirmação.
4. FlowSuitability com razões objetivas.
5. Handoff ao Flow sem reimplementar sua lógica.
6. Preservar lineage e side-effect-free draft.

Gate: destinos independentes; nenhuma chamada de run antes de Approve & Run.

### Fase 7 — Experiência visual

Arquivos: decompor `ComposerSurface.tsx`, componentes colocalizados, SCSS
Modules, i18n e testes.

1. Implementar conversa, motivation map, brief, gaps, skills e revisão.
2. Criar barra persistente de destinos.
3. Mostrar FlowSuitability sem pressionar o usuário.
4. Remover strings hardcoded e inline styles tocados.
5. Validar responsividade, foco, teclado e leitores de tela.

Gate: 320, 390, 768, 1024, 1280 e 1440 px sem CTA obstruído; fluxo completo por
teclado.

### Fase 8 — Avaliação do produto

1. Criar cenários versionados e prompts de referência.
2. Avaliar completude, precisão, concisão e utilidade das perguntas.
3. Registrar métricas locais sem conteúdo sensível: rodadas, gaps, repetição,
   skills, destino, recomendação de Flow e degradação.
4. Executar testes de usabilidade simples e complexos.

Gate: zero repetição indevida; 100% dos critérios confirmados preservados;
telemetria sem prompt, secret ou path bruto.

### Fase 9 — Compatibilidade e fechamento

1. Migrar fixture de banco anterior.
2. Abrir sessões/artefatos existentes.
3. E2E Idea → Copy, Existing Prompt → Agent e Complex Task → Flow.
4. Testar Maestro disponível, ausente e incompatível.
5. Validar Web e Desktop no mesmo Core.
6. Rebuild/restart apenas após gates verdes.

## 12. Critérios de aceitação

1. Uma frase gera Motivation Map inicial e pergunta relevante sem efeitos.
2. Há no máximo uma pergunta principal por rodada, com impacto explicado.
3. Pergunta respondida/dispensada não reaparece sem evidência nova.
4. Toda inferência tem origem e confirmação/premissa explícita.
5. Usuário copia `ACTIONABLE_WITH_GAPS` após confirmar gaps, sem Flow.
6. Copy é portátil e não presume skills externas.
7. Send to Agent valida skills no alvo.
8. Nenhuma skill aparece sem fonte, ID e disponibilidade verificável.
9. Maestro indisponível não produz conselho falso.
10. Prompt existente mostra lacunas, contradições e diff.
11. PromptArtifact inclui hash, revisão, contexto e skills exatos.
12. Refinar cria nova revisão sem mutar artefato anterior.
13. Tarefa simples recebe `DIRECT_FIT` sem pressão para Flow.
14. Tarefa complexa recebe recomendação de Flow com razões concretas.
15. Mesmo em `FLOW_STRONGLY_RECOMMENDED`, Copy/Agent continuam acessíveis.
16. Handoff cria somente draft e preserva lineage e requisitos.
17. Nenhum runtime, Agent ou MissionRun nasce implicitamente.
18. Flow existente e Mission Runner não são reescritos.
19. Sessões legadas continuam abrindo e finalizando.
20. UI não adiciona texto hardcoded, `any`, `!important` ou inline estático.
21. E2E dos três destinos passa em Web e bundle Desktop aplicável.
22. Gates Go, Web e diff passam antes da conclusão.

## 13. Riscos

| Risco | Mitigação |
|---|---|
| Entrevista cansativa | impacto, quick replies, saída livre e convergência |
| Prompt redundante | compilador por seções, deduplicação e rubrica |
| Duplicar Maestro | adapters e autoridade explícita |
| Colisão de skills | identidade por fonte/ID/versão |
| Flow virar obrigação | CTAs equivalentes e recomendação explicada |
| Handoff perder dados | contrato, hash, lineage e contract tests |
| Contexto velho | fingerprint, idade, STALE e reconfirmação |
| Migração quebrar sessões | schema aditivo e fixture legado |
| Score decorativo | gates separados, razões e corpus humano |
| Modelo gerar loops | schema, validação e fingerprint de perguntas |

## 14. Verificação final

```text
go test ./...
go vet ./...
go test -race ./...
make web-verify
git diff --check
```

Se o bundle final mudar: `make build`, build Desktop aplicável, restart controlado
e smoke Web/Desktop.

## 15. Coordenação

- Fases 0–1 precedem as demais.
- Skills e contexto podem avançar em paralelo após o domínio.
- Descoberta depende do domínio e contrato básico de skills.
- Compilador depende de descoberta/contexto/skills.
- Destinos dependem do compilador e contrato atual do Flow.
- UI pode iniciar pelos modelos, mas fecha após APIs.
- O responsável pelo Flow valida apenas fixtures de handoff; este plano não
  assume ownership do Flow interno.

## 16. ADR

### Decisão

Reescrever evolutivamente o Composer como ambiente de descoberta de motivação,
briefing rastreável, curadoria federada de skills e compilação de prompt. Copy,
Agent e Flow são destinos independentes. Flow é recomendado por sinais objetivos
de complexidade, mas permanece opcional.

### Alternativas rejeitadas

1. Formulário curto: não descobre motivação nem produz prompt excelente.
2. Sempre gerar Flow: adiciona atrito e contradiz o contrato canônico.
3. Planner/skills próprios: duplica Maestro e diverge com o tempo.
4. Apenas melhorar UI: não corrige domínio, qualidade ou integração.

### Consequências

- migração aditiva e novos contratos;
- qualidade exige corpus e E2E, não apenas persistência;
- skills exigem adapters e capability checks reais;
- Flow recebe handoff mais rico sem ser reescrito;
- profundidade precisa permanecer sob controle explícito do usuário.

