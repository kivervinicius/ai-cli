# Changelog

O projeto ainda está em Community Preview. Entradas abaixo descrevem mudanças
verificáveis do trabalho atual; não representam uma release publicada.

## Unreleased — Community Preview

- adicionado cache de catálogo de skills com TTL 30s e invalidação em sync;
- adicionado allowlist de diretório para execução de shell via API;
- adicionada captura de stderr e documentação de bounded output;
- reforçado hash de preview com mtime do script para detecção de stale;
- reforçada asserção de timestamp em testes de runner;
- migrados ~20 inline styles estáticos para SCSS Modules no Composer;
- corrigida a seleção por `Enter` na tela TUI `nexus usage`;
- reforçada a identidade de filesystem e a recência de workspaces;
- corrigidos readiness de SessionHost, fixtures Windows e ciclo de vida
  ConPTY, com evidência nativa ainda pendente;
- separado o update do Nexus da manutenção explícita do Maestro;
- reforçados testes de segurança, atualização, browser E2E, Axe e sincronização
  do frontend embutido;
- corrigida a metadata pública de licença para MIT.

Consulte [`DEV/validation/FINAL_PLATFORM_RELEASE_DESKTOP_REPORT.md`](DEV/validation/FINAL_PLATFORM_RELEASE_DESKTOP_REPORT.md)
para a matriz de evidências e limitações atuais.
