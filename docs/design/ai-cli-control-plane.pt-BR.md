# Arquitetura do plano de controle

[🇬🇧 English](ai-cli-control-plane.md) | **🇧🇷 Português** | [🇪🇸 Español](ai-cli-control-plane.es.md)

O IAPro Nexus separa apresentação, agendamento e execução. O CLI e as TUIs chamam a camada de aplicação; o servidor web expõe APIs locais autenticadas; registries mantêm providers e runtimes; scheduler, quota, cooldown e fallback decidem qual perfil pode executar; adapters traduzem a decisão para cada CLI.

Projetos, Agentes e Missões são identidades duráveis. Runtimes e sessões de provider são gerações substituíveis sob essas identidades. O Workspace OS persiste layout sem misturar estado visual com o processo do terminal.

Internacionalização também respeita essa fronteira: React usa catálogos `i18next`; CLI/TUIs usam catálogos `go-i18n`; contratos JSON, IDs e enums não são localizados. Novos idiomas exigem registro do locale, catálogo completo e teste de paridade.

Limites de segurança: loopback por padrão, autenticação e CSRF na web, isolamento de homes, argumentos sem shell intermediário e redação em handoffs. Falhas degradam recursos específicos sem fabricar estado ou capacidade.
