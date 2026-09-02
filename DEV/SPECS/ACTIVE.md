# Especificação ativa

## Objetivo

Preservar a entrega Nexus atual e incorporar melhorias verificáveis de segurança de launcher, continuidade local de conversas e seleção honesta de contas.

## Aceitação

- Argumentos de hosts destacados não ficam persistidos no registry; são consumidos uma vez de envelope privado.
- Perfis suportados reaproveitam somente artefatos de conversa não credenciais.
- Quota sem observação recente e atribuível não decide automaticamente entre contas; fallback usa LRU de perfis saudáveis.
- Build e testes focados passam.
- Interfaces visuais de terminal usam exclusivamente Bubble Tea, Bubbles e
  Lip Gloss; tabelas humanas são pesquisáveis por padrão e `--json` atende
  automações.

## Fora de escopo

- Substituir a arquitetura atual de Nexus/Flow por APIs da branch antiga `feat/nexus-v1`.
- Copiar binários, bundles ou documentos históricos inteiros sem validação.
