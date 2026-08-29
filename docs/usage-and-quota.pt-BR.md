# Uso e quota

[🇬🇧 English](usage-and-quota.md) | **🇧🇷 Português** | [🇪🇸 Español](usage-and-quota.es.md)

`nexus usage` apresenta snapshots de capacidade por provedor e perfil. Cada snapshot informa origem, idade do cache, limite conhecido, consumo, saldo e janela de renovação quando o provedor disponibiliza esses dados.

`UNKNOWN` significa que o provedor não ofereceu um valor confiável; não significa quota ilimitada. Use `--json` para automação: suas chaves e enums não mudam com o idioma. O seletor evita contas em cooldown e pode fazer fallback após rate limit.
