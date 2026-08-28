<p align="center">
  <a href="https://github.com/IAPro-Community">
    <img src="logo.png" alt="IAPro Community Logo" width="200">
  </a>
</p>

<p align="center">
  <img src="assets/banner.svg" alt="IAPro AI Control Banner" width="100%">
</p>

<p align="center">
  <a href="https://github.com/IAPro-Community"><img src="https://img.shields.io/badge/Organization-IAPro--Community-blueviolet?style=for-the-badge&logo=github" alt="IAPro Community"></a>
  <a href="https://golang.org"><img src="https://img.shields.io/badge/Go-1.25%2B-00ADD8?style=for-the-badge&logo=go" alt="Versão Go"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-Apache--2.0-green.svg?style=for-the-badge" alt="Licença"></a>
  <a href="https://kernel.org"><img src="https://img.shields.io/badge/Plataforma-Linux%20%7C%20macOS%20%7C%20Windows-FCC624?style=for-the-badge&logo=linux&logoColor=black" alt="Plataforma"></a>
  <img src="https://img.shields.io/badge/Providers-Codex%20%7C%20AGY%20%7C%20Claude%20%7C%20OpenCode%20%7C%20Gemini%20%7C%20Cursor-7C3AED?style=for-the-badge" alt="Provedores Suportados">
</p>

<p align="center">
  <strong>🇧🇷 Português (Brasil)</strong> &nbsp;|&nbsp; <a href="README.en.md">🇬🇧 English</a>
</p>

<h3 align="center">
  ⚡ IAPro AI Control — Open-Source Local Control Plane &amp; Web Cockpit para Coding Agents
</h3>

<p align="center">
  <i>Um projeto do ecossistema <strong><a href="https://github.com/IAPro-Community">IAPro Community</a></strong> para Engenharia de Software Agêntica</i>
</p>

---

O **IAPro AI Control (`ai`)** é o **Control Plane local e visual oficial da <a href="https://github.com/IAPro-Community">IAPro Community</a>**, projetado para gerenciar, supervisionar e orquestrar múltiplos coding agents no terminal e no navegador (como **Google AGY / Antigravity**, **Anthropic Claude Code**, **Cursor Agent**, **OpenAI Codex**, **OpenCode** e **Google Gemini CLI**).

Ele gerencia de forma inteligente e segura múltiplos projetos, isola autenticações e credenciais em perfis dedicados por provedor (isolamento de credenciais, não sandbox de processo), fornece terminal supervisionado em tempo real com governança de escrita e checkpoints de handoff de contexto de trabalho entre provedores.

---

## 📸 Interface de Terminal Interativa (TUI)

Inicie o painel interativo a qualquer momento executando `ai`:

```text
 ┌────────────────────────────────────────────────────────────────────────────────────────┐
 │  AI CLI Control Plane v0.4.0                             Workspace: ~/tools/ai-manager │
 ├─────────────────────────┬──────────────────────────────────────────────────────────────┤
 │ Providers               │ Accounts (Codex)                                             │
 │                         │                                                              │
 │ ▸ ● Codex             2 │ ▸ ● [1] openai-work          ChatGPT Plus  [███████░░░] 70% ★│
 │   ● AGY               2 │   ● [2] openai-personal      ChatGPT Plus  [██████████] 100% │
 │   ○ Claude            0 │                                                              │
 │   ○ OpenCode          0 │                                                              │
 │   ○ Gemini            0 │                                                              │
 ├─────────────────────────┴──────────────────────────────────────────────────────────────┤
 │ Recent Sessions (Universal Index)                                                      │
 │                                                                                        │
 │ ▸ [1] há 12 min  # REFACTOR CONTROL PLANE                 [AGY   ]  ~/tools/ai-manager │
 │   [2] ontem      Verificar tipagem e lint                [CODEX ]  ~/tools/ai-manager   │
 │   [3] há 2 dias  Corrigir problemas de concorrência      [CODEX ]  ~/tools/ai-manager   │
 │   [4] há 3 dias  Auditoria de segurança                  [AGY   ]  ~/projetos           │
 ├────────────────────────────────────────────────────────────────────────────────────────┤
 │ Selected: Codex / openai-work  [Authenticated]                                         │
 │ Actions:  [Enter/1-9] Run  [c] Continue Latest  [r] Resume Modal  [s] Quotas  [l] Login│
 │ AUTO: openai-personal (100% capacity) is optimal for new sessions                     │
 ├────────────────────────────────────────────────────────────────────────────────────────┤
 │ [↑/↓] Navegar  [←/→/Tab] Alternar Caixa  [1-9] Disparo Rápido  [/] Buscar  [q/Esc] Sair│
 └────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## 🌟 Principais Recursos

### 1. 🛡️ Isolamento de Credenciais Multi-Conta
- **Estado Isolado:** Cada perfil possui seu próprio diretório `HOME`, `XDG_DATA_HOME`, `XDG_CONFIG_HOME`, sessão privada D-Bus e cofre `gnome-keyring-daemon` dedicado.
- **Zero Colisão de Tokens:** Tokens OAuth do Google, credenciais OpenAI, Anthropic e chaves de API ficam estritamente isolados em sua pasta de perfil com permissões restritas `0600`.
- **Presets de Isolamento Configuráveis:** Escolha entre `developer` (compartilha dotfiles e contexto de git), `strict` (isolamento completo de credenciais e HOME) e `compat`.
- **Preservação de Contexto de Projeto:** Mantém seu diretório de trabalho (`CWD`), usuário UID/GID, configurações locais (`.gitconfig`, `.ssh`) e repositórios intactos.

### 2. 🧠 Seleção Inteligente de Contas (Smart Account Selector)
O motor de agendamento avalia múltiplos fatores em tempo real para escolher a conta ideal:
- **Pontuação Multi-Fator:** Avalia capacidade restante de quota, bindings de workspace, perfil padrão, prioridade manual configurada e planos Pro.
- **Transparência Total (`ai explain <provider>`):** Entenda exatamente por que uma conta foi escolhida:
  ```bash
  $ ai explain agy
  === Smart Account Selection Explanation: AGY ===
  Evaluation of all candidate profiles:

  Optimal Choice: google-work (Reason: authenticated, 92% capacity (+73.9), default profile (+25.0), pro tier (+15.0))

  PROFILE            ELIGIBLE   SCORE    BREAKDOWN / REJECTION
  google-work        YES        213.9    authenticated, 92% capacity (+73.9), default profile (+25.0), pro tier (+15.0)
  google-personal    YES        195.0    authenticated, 100% capacity (+80.0), pro tier (+15.0)
  ```

### 3. ⚡ Fallback Automático & Cooldown Anti-Rate-Limit
- **Recuperação de Ciclo Seguro:** Quando uma conta atinge HTTP 429 ou esgota a quota durante uma execução, o AI CLI intercepta a falha, registra o cooldown e realiza fallback transparente para a próxima conta disponível com saldo.
- **Prevenção de Loops:** O sistema garante que cada conta só seja testada uma vez por ciclo de fallback.

### 4. 📊 Monitor de Quotas Reais e Honesto (`ai usage`)
- **Zero Quotas Fictícias:** Elimina suposições de 100%. Exibe com precisão o estado real da quota: `LIVE` (obtida via API/CLI), `CACHED` (estado local recente), `RATE_LIMITED` ou `UNKNOWN`.
- **Limites de 5 Horas e Semanais:** Visualização clara das janelas de reset de cada modelo.

```bash
$ ai usage
```
```text
PROVIDER   PROFILE          ACCOUNT                  PLAN             CAPACITY / 5H                STATUS
agy        google-work      work@company.com         Google AI Pro    [█████████░] 92%             CACHED
agy        google-personal  dev@gmail.com            Google AI Pro    [██████████] 100%            CACHED
codex      openai-work      work@company.com         ChatGPT Plus     [███████░░░] 70%             CACHED
codex      openai-personal  dev@gmail.com            ChatGPT Plus     [██████████] 100%            CACHED
```

### 5. 🔄 Retomada Universal de Sessões (Session Handoff)
- **Índice Unificado:** Descubra e pesquise instantaneamente conversas de todos os provedores (`ai sessions` ou `/` na TUI).
- **Handoff Entre Contas:** Continue uma conversa existente com outra conta que possua limite disponível:
  ```bash
  ai resume <session-id> [provider:profile]
  ```
- **Modal Interativo na TUI:** Pressione `[r]` ou `[Enter]` numa sessão recente para escolher a conta com sugestão automática do Smart Selector.

### 6. 📁 Bindings por Workspace / Projeto
Associe pastas de projetos a perfis específicos para uso automático de contas de trabalho ou pessoais:
```bash
# Vincular projeto atual à conta corporativa
ai bind codex:openai-work

# Listar workspaces e seus vínculos
ai workspaces
```

### 7. 🔌 5 Provedores Suportados Nativamente

O AI CLI Control Plane integra nativamente os principais assistentes de IA para terminal:

#### 🟢 OpenAI Codex (`codex`)
- **Execução:** `ai codex` (seleção inteligente) ou `ai codex:openai-work --model gpt-5.6-sol`
- **Isolamento:** Variável de ambiente `CODEX_HOME` dedicada por perfil, com `auth.json` e `config.toml` em modo restrito (`0600`).
- **Quotas:** Limites de 5 Horas e Semanais compatíveis com os painéis oficiais do Codex (`/status`, `/usage`).
- **Retomada:** Sintaxe nativa do Codex (`codex resume <session-id>`).

#### 🔵 Google AGY / Antigravity (`agy`)
- **Execução:** `ai agy` (seleção inteligente) ou `ai agy:google-work -c`
- **Isolamento:** Sessão D-Bus privada isolada (`dbus-run-session`) com cofre `gnome-keyring-daemon` dedicado e `antigravity-oauth-token` isolado.
- **Quotas:** Limites de 5 Horas e Semanais do Google AI Pro (Gemini 2.5 Pro / Claude 3.7 Sonnet).
- **Retomada:** Despacho nativo de conversas (`agy --conversation=<session-id>`).

#### 🟣 Anthropic Claude Code (`claude`)
- **Execução:** `ai claude` (seleção inteligente) ou `ai claude:claude-work -p "refatorar auth"`
- **Isolamento:** `CLAUDE_CONFIG_DIR` isolado por perfil com gerenciamento independente de OAuth e tokens de API.
- **Retomada:** Despacho oficial (`claude --resume <session-id>`).

#### 🟠 OpenCode (`opencode`)
- **Execução:** `ai opencode` ou `ai opencode:local --model ollama/deepseek-r1`
- **Isolamento:** `OPENCODE_HOME` e `XDG_DATA_HOME` independentes por perfil.
- **Capacidades:** Suporte multi-modelo para LLMs locais (Ollama, vLLM) e provedores em nuvem.
- **Retomada:** Retomada de sessões do OpenCode (`opencode session <id>`).

#### 💎 Google Gemini CLI (`gemini`)
- **Execução:** `ai gemini` (seleção inteligente) ou `ai gemini:personal`
- **Isolamento:** `GEMINI_CLI_HOME` com `google_accounts.json` dedicado por perfil.
- **Capacidades:** Autenticação Google OAuth isolada sem sobrescrita de contas.

---

### 8. ⚡ AI Control — Runtimes Supervisionados & Canal Universal `/ai`

O AI CLI oferece dois modos de operação complementares:
- **Modo Classic (`ai <provider>`)**: Execução rápida e direta do assistente no terminal com seleção inteligente de perfis e fallback anti-rate-limit.
- **Modo Supervised (`ai control start <provider>`)**: Execução supervisionada pelo `SessionHost`, permitindo reconexão (Attach/Detach), observabilidade de eventos em tempo real e canal universal de comandos em sessão.

#### 🎮 Comandos do Canal Universal `/ai` (Dentro do Terminal Supervisionado)
Ao executar em modo supervisionado, digite comandos especiais iniciados por `/ai` diretamente no prompt do assistente. O AI Control intercepta o comando localmente sem repassar ao modelo:

| Comando | Descrição |
| :--- | :--- |
| `/ai status` | Exibe o status do runtime ativo, PID, sessão e capacidade de quota. |
| `/ai accounts` | Lista todas as contas configuradas do provedor e seus limites restantes. |
| `/ai usage` | Mostra as quotas de uso e janelas de reset em tempo real. |
| `/ai handoff <perfil>` | **Account Handoff:** Migra a sessão ativa para outra conta com quota do mesmo provedor. |
| `/ai continue <provedor>` | **Context Handoff:** Cria uma nova sessão em outro provedor com o envelope de tarefas e arquivos modificados. |
| `/ai detach` | Desconecta do terminal interativo mantendo o processo do assistente em execução. |
| `/ai stop` | Envia sinal de encerramento controlado para o assistente. |
| `//ai <texto>` | **Prefixo de Escape:** Envia o texto literal iniciado por `/ai` para o assistente. |

#### 🖥️ Subcomandos do Control Center (`ai control` / `ai ui`)
```bash
ai control                                      # Abre a interface interativa Bubble Tea
ai control start codex [--profile work]         # Inicia um runtime supervisionado
ai control running [--json]                     # Lista runtimes supervisionados ativos
ai control status <runtime-id> [--json]         # Inspeciona detalhes de um runtime
ai control attach <runtime-id>                  # Reconecta o terminal a uma sessão ativa
ai control stop <runtime-id>                    # Para um runtime supervisionado
ai control handoff <id> codex:personal          # Executa troca de conta com preservação de sessão
ai control continue <id> --with claude:work     # Transfere contexto para outro provedor
ai control cleanup                              # Remove registros órfãos e sockets antigos
ai control doctor [--json]                      # Audita drivers, sockets e compatibilidade
```

---

## 🚀 Instalação e Início Rápido

### Instalação em 1 Linha (Zero-Clone / Recomendado)

**Linux e macOS (via `curl | bash`):**
```bash
curl -fsSL https://raw.githubusercontent.com/kivervinicius/ai-cli/main/install.sh | bash
```

**Windows e PowerShell Core (via `irm | iex`):**
```powershell
irm https://raw.githubusercontent.com/kivervinicius/ai-cli/main/install.ps1 | iex
```

**Via `go install` (qualquer sistema com Go):**
```bash
go install github.com/kivervinicius/ai-cli/cmd/ai@latest
```

---

### Instalação a partir do Código Fonte (Opcional)

**Linux / macOS:**
```bash
git clone https://github.com/kivervinicius/ai-cli.git
cd ai-cli
make install
```

**Windows / PowerShell:**
```powershell
git clone https://github.com/kivervinicius/ai-cli.git
cd ai-cli
.\install.ps1
```

---

## 🐚 Autocompletar no Shell (Bash, Zsh, Fish e PowerShell)

Ative o autocompletar completo de provedores, perfis, conversas e flags:

### Bash
```bash
source <(ai completion bash)
# Para persistir no ~/.bashrc:
echo 'source <(ai completion bash)' >> ~/.bashrc
```

### Zsh
```zsh
source <(ai completion zsh)
# Para persistir no ~/.zshrc:
echo 'source <(ai completion zsh)' >> ~/.zshrc
```

### Fish
```fish
ai completion fish | source
```

### PowerShell (Windows & pwsh)
```powershell
ai completion powershell | Out-String | Invoke-Expression

# Para persistir no seu $PROFILE:
Add-Content $PROFILE "`nai completion powershell | Out-String | Invoke-Expression"
```

---

## 💻 Guia Completo de Comandos CLI

### Interface Interativa & Execução
| Comando | Descrição |
| :--- | :--- |
| `ai` | Abre a TUI interativa completa em Bubble Tea (Modo Clássico). |
| `ai control` / `ai ui` | Abre a Central de Controle TUI para runtimes supervisionados. |
| `ai control start <provider>` | Inicia um processo supervisionado em background com canal universal `/ai`. |
| `ai control running` | Lista os runtimes ativos gerenciados pelo Control Plane. |
| `ai control attach <id>` | Conecta o terminal interativo a um runtime em background. |
| `ai control stop <id>` | Encerra graciosamente um runtime supervisionado. |
| `ai control handoff <id> <perfil>` | Executa troca transacional de conta no mesmo provedor com continuidade de sessão. |
| `ai control continue <id> --with <prov>` | Executa migração de contexto (Context Handoff) com redação de segredos. |
| `ai control doctor` | Diagnóstico de integridade e capacidades reais dos drivers do Control Plane. |
| `ai <provider> [args...]` | Executa o provedor com seleção inteligente da melhor conta (ex: `ai codex -m gpt-5`). |
| `ai <provider>:<perfil> [args...]` | Executa diretamente com o perfil especificado (ex: `ai agy:work -c`). |
| `ai explain <provider>` | Exibe a pontuação e justificativa da seleção de conta do Smart Selector. |

### Comandos Slash em Sessão Supervisionada (`/ai`)
| Comando Slash | Descrição |
| :--- | :--- |
| `/ai status` | Exibe estado do runtime, identificadores e métricas de cota ativas. |
| `/ai accounts` | Lista contas configuradas e capacidade restante do provedor ativo. |
| `/ai usage` | Exibe snapshot detalhado de cotas e janelas de reset. |
| `/ai handoff <perfil>` | Transição transacional de conta no mesmo provedor. |
| `/ai continue <provider>` | Migração de contexto para outro provedor com checkpoint seguro. |
| `/ai detach` | Desconecta do terminal mantendo o processo do assistente rodando em background. |
| `/ai stop` | Encerra a sessão supervisionada com segurança. |
| `//ai <texto>` | Prefixo de escape para enviar comandos `/ai` literais para o assistente. |

### Gerenciamento de Perfis & Autenticação
| Comando | Descrição |
| :--- | :--- |
| `ai profiles` / `ai list` | Lista todos os perfis configurados, contas, planos e status. |
| `ai add <provider> <nome>` | Cria um novo perfil isolado e inicializa seu diretório de credenciais. |
| `ai login <provider> <nome>` | Executa o fluxo de autenticação oficial do provedor. |
| `ai logout <provider> <nome>` | Remove as credenciais salvas do perfil. |
| `ai use <provider> <nome>` | Define o perfil padrão para o provedor. |
| `ai rename <provider> <antigo> <novo>` | Renomeia um perfil preservando histórico e sessões. |
| `ai remove <provider> <nome>` | Remove com segurança o perfil e suas credenciais. |

### Quotas & Conversas
| Comando | Descrição |
| :--- | :--- |
| `ai usage` | Monitor unificado de cotas 5H e Semanais com barras de progresso reais. |
| `ai usage <provider> <nome>` | Exibe os limites detalhados de uma conta específica. |
| `ai sessions` | Lista o histórico unificado de sessões recentes de todos os provedores. |
| `ai sessions search <termo>` | Pesquisa sessões por título, ID ou workspace. |
| `ai resume <id> [perfil]` | Retoma uma conversa específica com o perfil indicado ou com a melhor conta. |

### Workspaces & Configurações
| Comando | Descrição |
| :--- | :--- |
| `ai bind <provider>:<perfil>` | Vincula o diretório atual ao perfil especificado. |
| `ai unbind <provider>` | Remove o vínculo de workspace para o provedor. |
| `ai workspaces` | Lista todos os workspaces descobertos e seus vínculos ativos. |
| `ai current` | Exibe o perfil ativo e vínculos do workspace atual. |
| `ai config validate` | Valida a integridade do arquivo de configuração. |

### Auditoria, Diagnóstico & Observabilidade
| Comando | Descrição |
| :--- | :--- |
| `ai doctor` | Executa diagnósticos de sistema (D-Bus, keyrings, CLIs instalados, permissões). |
| `ai security` | Audita permissões de arquivos e valida o isolamento de credenciais. |
| `ai stats` | Exibe métricas agregadas de uso, taxas de sucesso e rate limits. |
| `ai history` | Exibe o log de eventos e telemetria de execuções recentes. |
| `ai paths` | Mostra os diretórios XDG de dados, configuração e estado. |

---

## 🏗️ Arquitetura e Modelo de Segurança

```mermaid
graph TD
    User["Desenvolvedor / Terminal"] --> Entrypoint["ai (CLI / Bubble Tea TUI)"]
    
    subgraph Control_Plane["AI CLI Control Plane"]
        Entrypoint --> Scheduler["Smart Account Selector (Multi-Factor Scoring)"]
        Entrypoint --> QuotaEngine["Honest Quota Engine (LIVE · CACHED · UNKNOWN)"]
        Entrypoint --> FallbackExec["Automatic Fallback & Cooldown Tracker"]
        Entrypoint --> SessionIndex["Universal Session Store (Handoff & Resume)"]
        Entrypoint --> SecurityLayer["Security & Isolation Presets (strict/dev/compat)"]
    end
    
    subgraph Provider_Adapters["Adaptadores de Provedores"]
        Scheduler --> CodexAd["Codex Adapter"]
        Scheduler --> AgyAd["AGY Adapter"]
        Scheduler --> ClaudeAd["Claude Adapter"]
        Scheduler --> OpenCodeAd["OpenCode Adapter"]
        Scheduler --> GeminiAd["Gemini Adapter"]
    end
    
    subgraph Sandboxes["Perfis Isolados de Credenciais"]
        CodexAd --> CodexHome["CODEX_HOME (~/.local/share/ai-cli/profiles/codex/*)"]
        AgyAd --> AgyHome["D-Bus + Keyring (~/.local/share/ai-cli/profiles/agy/*)"]
        ClaudeAd --> ClaudeHome["Isolated Runtime (~/.local/share/ai-cli/profiles/claude/*)"]
    end
```

### Garantias de Segurança:
- 🔒 **Permissões Estritas `0600`:** Arquivos de credenciais, tokens OAuth e chaves privadas nunca são acessíveis por outros usuários do sistema operacional.
- 🔒 **Redação Automática:** Logs de telemetria e saídas de diagnóstico passam por filtros de redação que mascaram tokens JWT, chaves OpenAI/Anthropic e dados sensíveis.
- 🔒 **Isolamento de Processos:** Cofres de senhas do `gnome-keyring` rodam em instâncias D-Bus isoladas para cada perfil do AGY.

---

## 🌐 Ecossistema IAPro Community

O **IAPro AI Control** integra o conjunto oficial de ferramentas para desenvolvimento agêntico da [IAPro Community](https://github.com/IAPro-Community):

- **[Orquestrador Maestro](https://github.com/IAPro-Community)**: Metodologia e CLI de orquestração de missões com DEV Gates rigorosos e garantia formal de entrega.
- **IAPro Skill Library**: Catálogo padronizado de capacidades executáveis para coding agents.
- **IAPro AI Control**: Control Plane de alta performance, PTY multiplexer supervisionado e Web Cockpit visual para todos os agentes na sua máquina.

### Contribua com a Comunidade
Participe do desenvolvimento, reporte sugestões ou proponha novos drivers de coding agents:
👉 **[https://github.com/IAPro-Community](https://github.com/IAPro-Community)**

---

## 🤝 Contribuições

Contribuições, sugestões de novos provedores e melhorias são muito bem-vindas!
1. Faça um Fork do projeto em [IAPro-Community/ai-control](https://github.com/IAPro-Community).
2. Crie uma branch para sua feature (`git checkout -b feat/nova-feature`).
3. Envie seus commits (`git commit -m 'feat: adiciona novo driver'`).
4. Abra um Pull Request.

---

## 📄 Licença

Distribuído sob a licença **Apache-2.0**. Consulte o arquivo [`LICENSE`](LICENSE) para obter mais informações.

