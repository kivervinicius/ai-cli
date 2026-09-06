# ADR: Integração com Orquestrador Maestro Independente e 100% Opcional

## Status
Aceito e Implementado

## Contexto
O Orquestrador Maestro é um produto complementar para automação avançada e orquestração de skills. Anteriormente, os scripts de instalação tentavam instalar silenciosamente `@iapro/orquestrador-maestro-cli` via npm global. Isso violava o princípio de isolamento de ferramentas, exigia privilégios de Node/npm não solicitados pelo usuário e acoplava o ciclo de vida do Nexus a ferramentas externas.

## Decisão
1. **Desacoplamento Completo**: O Nexus funciona plenamente sem o Maestro. A ausência do Maestro resulta no estado de degradação graciosa `MAESTRO_DEGRADED` sem falhar o Nexus, mantendo todos os modos Diretos, terminais, sessões de provedores e workspaces totalmente operacionais.
2. **Sem Instalação Silenciosa**: `install.sh` e `install.ps1` instalam estritamente apenas o binário do IAPro Nexus. A instalação do Maestro requer flag explícita de consentimento (`--with-maestro` ou `-WithMaestro`).
3. **Versionamento e Updates Independentes**: O Nexus e o Maestro possuem versões e cadências de release totalmente distintas. O Nexus nunca tenta sobrescrever o Maestro silenciosamente em atualizações regulares.

## Consequências
- A instalação do Nexus é rápida, segura e com zero dependências externas não consentidas.
- A experiência de usuário reporta com total clareza a disponibilidade do Maestro em `nexus doctor` e no painel de configurações.
