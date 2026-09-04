# Composer deliberativo e Flow reutilizável

## Objetivo

Converter o Composer em uma elaboração conversacional durável que produz um
prompt final versionado. Um Flow é criado apenas por escolha explícita e é uma
definição reutilizável por Project, com DAG, Agent líder sugerido e execuções
independentes.

## Regras fixadas

- Contexto, decisões, perguntas e prompts pertencem a uma sessão durável.
- Maestro só fornece catálogo e recomendações reais; skills exigem confirmação.
- Intelligence do Project planeja e revisa; Agents executam nos seus providers.
- O líder é sugerido por Flow, pode ser alterado, nunca é criado silenciosamente
  e não altera o DAG aprovado.
- Execuções usam somente a última revisão aprovada; clone entre Projects limpa
  vínculos locais e move equivale a clone com arquivamento da origem.

## Entregas

1. Store e API de ComposerSession, ComposerTurn, SkillProposal e PromptArtifact.
2. Superfície conversacional com briefing, histórico e materialização explícita.
3. Biblioteca de Flows reutilizáveis, líder sugerido, clone/move e DAG aprovado.
4. Runner com recibos por onda e síntese final do líder, sem retries estruturais.
