# Lane 02 — Qualidade

Objetivo: fechar as lacunas de acessibilidade e validação visual antes de uma declaração formal de conformidade.

## P1 — Em aberto

- [ ] Rodar auditoria automatizada com `@axe-core/playwright` contra o servidor ativo.
- [ ] Corrigir todas as violações encontradas e registrar evidência.
- [ ] Adicionar `role="tablist"`, `role="tab"` e `role="tabpanel"` às abas do Workspace.
- [ ] Validar navegação e layout com zoom de navegador em 200%.

## Critério de aceite

Não declarar WCAG 2.1 AA formal sem resultado do axe-core, correções verificadas e evidência do teste de zoom.

