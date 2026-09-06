# Segurança

[🇬🇧 English](security.md) | **🇧🇷 Português** | [🇪🇸 Español](security.es.md)

O Nexus é local-first. Perfis recebem diretórios isolados, o servidor web usa loopback por padrão e handoffs passam por redação de segredos. Não exponha a interface diretamente à internet; para acesso remoto use túnel SSH ou VPN privada.

Execute `nexus security` para auditar limites de compartilhamento e `nexus doctor` para verificar binários, autenticação e permissões. Nunca inclua tokens, cookies ou arquivos de credenciais em relatórios. Mensagens e logs de provedores permanecem no idioma original para preservar evidência operacional.
