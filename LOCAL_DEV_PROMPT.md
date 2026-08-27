# Prompt para continuar localmente

Cole este prompt no Codex/AGY depois de descompactar o projeto:

---

Você está trabalhando no projeto `ai-manager`, um CLI Linux em Go que orquestra múltiplos perfis de autenticação de CLIs externas sem API keys e sem trocar de usuário Linux.

Leia primeiro `README.md`, `ARCHITECTURE.md` e todo o código em `cmd/` e `internal/`. Não reimplemente Codex ou Antigravity e não leia/parsing/copie tokens OAuth. O limite arquitetural é: preparar um ambiente isolado e executar o binário oficial do provider.

Objetivos desta sessão:

1. Rode `go test ./...`, `go vet ./...` e `go build ./cmd/ai`; corrija qualquer falha encontrada.
2. Revise o fluxo `ai add codex <nome>`:
   - criar perfil em `~/.local/share/ai-manager/profiles/codex/<nome>/home`;
   - garantir `cli_auth_credentials_store = "file"` sem destruir config existente;
   - executar `codex login` com `CODEX_HOME` do perfil;
   - manter cwd, UID e GID do usuário.
3. Revise o fluxo `ai add agy <nome>`:
   - criar HOME/XDG/keyring privados por perfil;
   - usar `dbus-run-session` e `gnome-keyring-daemon` apenas para `secrets`;
   - manter cwd e mesmo UID/GID;
   - permitir que o OAuth oficial do AGY abra o navegador do desktop através do helper `ai-browser`/`xdg-open` que restaura `AI_HOST_DBUS_SESSION_BUS_ADDRESS`;
   - nunca acessar ou exibir os tokens do AGY.
4. Faça testes de integração não destrutivos usando fake binaries `codex` e `agy` em um PATH temporário. Os testes devem confirmar:
   - perfis não compartilham HOME/CODEX_HOME;
   - cwd é preservado;
   - UID/GID não mudam;
   - argumentos após `--` são repassados sem alteração;
   - exclusão de perfil não afeta outros perfis.
5. Melhore `ai doctor` para detectar distro e dar instruções precisas sem executar `sudo` automaticamente.
6. Adicione um comando `ai inspect <provider> <profile>` que mostre somente metadados não secretos: caminhos, binário externo, cwd, UID/GID e variáveis de isolamento. Não mostre conteúdo de `auth.json`, keyrings, cookies, URLs OAuth, authorization codes ou tokens.
7. Adicione shell completion para bash/zsh se isso puder ser feito de forma pequena e confiável.
8. Mantenha compatibilidade com Linux e Go >= 1.22; prefira stdlib e poucas dependências.
9. Não implemente rotação automática baseada em quota/limite. A escolha de perfil deve continuar explícita pelo usuário (`ai agy:a`, `ai codex:b`, `ai use ...`).
10. Ao terminar, atualize README e explique exatamente quais comandos devo executar para testar com:
    - 3 perfis AGY: `google-a`, `google-b`, `google-c`;
    - 2 perfis Codex: `openai-a`, `openai-b`.

Antes de alterar a arquitetura, valide qualquer suposição dependente das versões atuais dos CLIs usando `codex --version`, `codex --help`, `agy --version` e `agy --help` disponíveis na máquina. Faça mudanças pequenas e verificáveis.

---
