# Seleção de contas

[🇬🇧 English](account-selection.md) | **🇧🇷 Português** | [🇪🇸 Español](account-selection.es.md)

O Nexus escolhe uma conta utilizável dentro do provedor solicitado. A decisão considera autenticação, cooldown, quota conhecida, prioridade, vínculo do projeto e estratégia configurada. Quota desconhecida continua desconhecida e nunca é apresentada como capacidade total.

Ordem prática: perfil explícito, vínculo do workspace, perfil padrão e seleção automática. Contas desabilitadas, sem autenticação ou em rate limit são rejeitadas. Se a conta escolhida falhar por limite, o fallback pode tentar outra conta elegível do mesmo provedor.

Use `nexus explain <provider>` para ver a pontuação, `nexus usage` para capacidade e `nexus bind <provider>:<profile>` para uma preferência por projeto. Os IDs e valores exibidos em `--json` permanecem estáveis.
