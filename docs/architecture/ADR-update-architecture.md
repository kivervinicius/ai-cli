# ADR: Arquitetura Unificada do Serviço de Atualizações (Update Service)

## Status
Aceito e Implementado

## Contexto
Diferentes superfícies (CLI, Web UI e Desktop Shell) precisavam verificar e aplicar atualizações de forma consistente, evitando que lógicas dispersas causassem divergências de versão, verificações inseguras ou corrupção de binários em ambientes gerenciados por gerenciadores de pacotes do SO.

## Decisão
Implementar um **Update Service Único** centralizado em `internal/update/`:
1. **Contrato Único**: `internal/update/service.go` orquestra verificação de versões, compatibilidade, canais (`stable`, `beta`, `nightly`) e download de artefatos.
2. **Validação Criptográfica Rigorosa**: Manifestos de release assinados com Ed25519 e checksums SHA256 verificados antes de qualquer modificação no disco. Rejeição estrita de downgrades e manifestos expirados.
3. **Respeito aos Gerenciadores de Pacote (`InstallationMethod`)**: Instalações sob DEB, RPM, Homebrew, Winget ou NSIS não realizam substituição direta do binário (`AllowsSelfUpdate() == false`). Em vez disso, fornecem instruções e comandos nativos do gerenciador de pacotes para atualização segura.
4. **Proteção de Trabalho Ativo**: Nenhuma atualização é aplicada silenciosamente se houver agentes ou missões ativas em execução.

## Consequências
- A CLI (`nexus update`) e o painel de configurações na UI (Web/Desktop) invocam a mesma lógica de atualização.
- Elimina falhas de concorrência ou substituições parciais de código.
