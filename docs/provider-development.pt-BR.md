# Desenvolvimento de providers

[🇬🇧 English](provider-development.md) | **🇧🇷 Português** | [🇪🇸 Español](provider-development.es.md)

Um adapter de provider implementa detecção, autenticação, execução, retomada e leitura de quota conforme as capacidades reais da CLI. Registre o adapter no registry, mantenha argumentos como lista segura e nunca invente suporte quando a ferramenta não oferece uma operação.

Dados contratuais usam IDs e enums estáveis em inglês. Textos humanos criados pelo Nexus devem usar IDs do pacote `internal/localization`; saída livre do provider permanece intacta. Adicione testes de detecção, comando, isolamento, classificação de falha e fallback antes de registrar um novo provider.
