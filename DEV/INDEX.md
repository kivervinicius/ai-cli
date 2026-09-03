# DEV Master Index

Guia e índice de navegação estruturada da documentação de engenharia e governança do IAPro Nexus.

O código canônico utiliza o binário `nexus`, mantendo `ai` como alias transparente de compatibilidade.

---

## 1. Governança e Estado Ativo

- [`HANDOFF.md`](HANDOFF.md): Estado atual de execução e próxima ação recomendada.
- [`CONTEXT.md`](CONTEXT.md): Arquitetura corrente, runtime, drivers e restrições operacionais.
- [`SPECS/ACTIVE.md`](SPECS/ACTIVE.md): Contrato ativo de trabalho e critérios de aceitação vigentes.
- [`VERIFY.md`](VERIFY.md): Evidências e histórico consolidado de validações e gates.
- [`WORKLOG.md`](WORKLOG.md): Diário de bordo detalhado de todas as implementações e mudanças.

---

## 2. Arquitetura, Interoperabilidade e Especificações Canônicas

- [`NEXUS_V1_ARCHITECTURE.md`](NEXUS_V1_ARCHITECTURE.md): Arquitetura mestre do Workspace OS, runtimes e store.
- [`NEXUS_CANONICAL_ALIGNMENT.md`](NEXUS_CANONICAL_ALIGNMENT.md): Alinhamento canônico entre CLI, Web e drivers de provedores.
- [`NEXUS_CAPABILITY_PRESERVATION.md`](NEXUS_CAPABILITY_PRESERVATION.md): Matriz de preservação honesta de capacidades.
- [`NEXUS_V1_AGENT_MODEL.md`](NEXUS_V1_AGENT_MODEL.md): Modelo de agentes duráveis e persistentes.
- [`NEXUS_V1_MAESTRO_INTEGRATION.md`](NEXUS_V1_MAESTRO_INTEGRATION.md): Integração profunda e governança com o Orquestrador Maestro.

---

## 3. Matrizes de Plataforma, Provedores e Segurança

- [`FINAL_PROVIDER_MATRIX.md`](FINAL_PROVIDER_MATRIX.md): Tabela de suporte e comportamento de Codex, AGY, Claude, OpenCode, Gemini e Cursor.
- [`FINAL_PLATFORM_MATRIX.md`](FINAL_PLATFORM_MATRIX.md): Compatibilidade entre Linux, macOS e Windows.
- [`FINAL_SECURITY_REPORT.md`](FINAL_SECURITY_REPORT.md) & [`NEXUS_MAXIMUM_DELIVERY_SECURITY.md`](NEXUS_MAXIMUM_DELIVERY_SECURITY.md): Isolamento de processos, sandbox, redação de credenciais e integridade de D-Bus/keyrings.

---

## 4. Frontend e Validação Canônica

- [`validation/FRONTEND_LATEST.md`](validation/FRONTEND_LATEST.md): Último gate do frontend (`make web-verify`).
- [`validation/FRONTEND_HISTORY.md`](validation/FRONTEND_HISTORY.md): Histórico compacto dos gates de frontend.
- Após alterações em `web/`, execute `make web-verify` (ou `npm --prefix web run verify`).
- Em caso de inconsistência visual na Web, faça o rebuild completo do binário com `make build` e reinicie o processo.

---

## 5. Histórico e Relatórios de Release

- [`FINAL_RELEASE_REPORT.md`](FINAL_RELEASE_REPORT.md): Relatório de fechamento de releases estáveis.
- [`NEXUS_V0_FINAL_AUDIT.md`](NEXUS_V0_FINAL_AUDIT.md) & [`NEXUS_V1_FINAL_ENGINEERING_REPORT.md`](NEXUS_V1_FINAL_ENGINEERING_REPORT.md): Auditorias técnicas de consolidação.
- [`DEV/validation/`](validation/): Logs brutos, capturas de tela e artefatos de testes automatizados e manuais.
