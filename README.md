<p align="center">
  <a href="https://github.com/IAPro-Community">
    <picture>
      <source media="(prefers-color-scheme: dark)" srcset="nexus-logo-dark.png">
      <img src="nexus-logo.png" alt="IAPro Nexus Logo" width="380">
    </picture>
  </a>
</p>

<p align="center">
  <a href="https://github.com/IAPro-Community"><img src="https://img.shields.io/badge/Organization-IAPro--Community-blueviolet?style=for-the-badge&logo=github" alt="IAPro Community"></a>
  <a href="https://golang.org"><img src="https://img.shields.io/badge/Go-1.25%2B-00ADD8?style=for-the-badge&logo=go" alt="Versão Go"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-Apache--2.0-green.svg?style=for-the-badge" alt="Licença"></a>
  <a href="https://kernel.org"><img src="https://img.shields.io/badge/Plataforma-Linux%20%7C%20macOS%20%7C%20Windows-FCC624?style=for-the-badge&logo=linux&logoColor=black" alt="Plataforma"></a>
  <img src="https://img.shields.io/badge/Providers-Codex%20%7C%20AGY%20%7C%20Claude%20%7C%20OpenCode%20%7C%20Gemini%20%7C%20Cursor-7C3AED?style=for-the-badge" alt="Provedores Suportados">
</p>

<p align="center">
  <strong>🇧🇷 Português (Brasil)</strong> &nbsp;|&nbsp; <a href="README.en.md">🇬🇧 English</a> &nbsp;|&nbsp; <a href="README.es.md">🇪🇸 Español</a>
</p>

<h3 align="center">
  ⚡ IAPro Nexus — Workspace Web-first para Coding Agents · Powered by Orquestrador Maestro
</h3>

<p align="center">
  <i>Um projeto do ecossistema <strong><a href="https://github.com/IAPro-Community">IAPro Community</a></strong> para Engenharia de Software Agêntica</i>
</p>

---

# Manual do IAPro Nexus

Este manual cobre **do básico ao avançado**, em linguagem clara para quem está
começando e com profundidade técnica para quem já vive de terminal.

Se preferir, a versão em inglês está em [`README.en.md`](README.en.md).

---

## 1. O que é o IAPro Nexus?

O **IAPro Nexus** (binário canônico `nexus`, com alias `ai` para compatibilidade total) é um
**workspace de controle local para coding agents** — os assistentes de IA que
trabalham dentro do seu terminal (Codex, Claude Code, Gemini CLI, OpenCode, AGY,
Cursor Agent).

Em resumo, o Nexus faz três coisas:

1. **Organiza por Projetos** — cada projeto de código vira a raiz do seu
   trabalho, com seus próprios agentes, sessões, terminais e configurações.
2. **Mantém Agentes persistentes** — um "Agente" (ex.: *Backend Developer*)
   sobrevive a reinícios, trocas de conta e até trocas de provedor. O que muda é
   a "geração de runtime" (a execução concreta), não a identidade do agente.
3. **Web-first** — a interface oficial é um painel web local (`nexus web`),
   com terminais reais (xterm.js) dentro do navegador. Sem terminal gigante
   obrigatório, sem instalar nada além de um binário.

O Nexus é **Powered by Orquestrador Maestro**: o [Orquestrador Maestro](https://github.com/IAPro-Community/Orquestrador-Maestro)
é quem define método, skills, risco e portões de qualidade. O Nexus executa o
trabalho no mundo real (processos, terminais, contas, quotas) sem duplicar o
conhecimento do Maestro.

> **Modelo de honestidade:** o Nexus nunca mostra "intenção como fato". Se você
> pediu *stop*, o estado aparece `STOPPING` até ser confirmado. Continuidade de
> sessão só é marcada como `VERIFIED` quando realmente verificada.

---

## 2. Instalação

### Requisitos

- **Sistema:** Linux, macOS ou Windows (runtime, não só compilação).
- **Go 1.25+** apenas se for compilar do código-fonte.
- **Provedores:** os CLIs oficiais (`codex`, `claude`, `gemini`, `opencode`,
  `agy`, `cursor`) são detectados no `PATH`.

### Opção A — Binário pré-compilado (recomendado)

```bash
# Linux / macOS (amd64 ou arm64)
curl -fsSL https://raw.githubusercontent.com/kivervinicius/ai-cli/main/install.sh | bash
```

```powershell
# Windows (PowerShell)
irm https://raw.githubusercontent.com/kivervinicius/ai-cli/main/install.ps1 | iex
```

O instalador baixa o binário do release mais recente e, se não conseguir, tenta
compilar do fonte.

### Opção B — Compilar do código-fonte

```bash
git clone https://github.com/kivervinicius/ai-cli.git
cd ai-cli
make build          # gera ./bin/nexus
make release-local  # escolhe a versão, compila frontend/Go e instala nexus + ai
sudo make install   # opcional: instala em /usr/local/bin
```

Os mesmos comandos também podem ser executados pela camada de conveniência da raiz:

```bash
npm run release
npm run build
npm test
```

### Verificar

```bash
nexus version
# IAPro Nexus 0.4.1 (linux/amd64, go1.25.0) commit <sha> built <data>
nexus doctor
```

---

## 3. Primeiros passos (guia para iniciantes)

### 3.1 Abrir o painel Web

```bash
nexus web
# ou o alias clássico:
nexus control web
```

O comando imprime uma URL como:

```
Bootstrap: http://127.0.0.1:PORT/?token=...segredo-de-uso-único...
```

Abra essa URL no navegador. Ela troca o token por um cookie de sessão seguro e
redireciona para o painel. O painel **só escuta em `127.0.0.1`** (loopback) — nada
de exposição pública por padrão.

> 💡 Quer abrir sem navegador automático? Use `nexus web --no-open` e copie
> a URL manualmente.

### 3.2 Criar um Projeto

No painel, seção **Projects** (ou Overview):

1. Digite o caminho de uma pasta de código existente (ex.: `~/meu-projeto`).
2. Clique em **Add**.
3. O projeto aparece na lista, com nome, slug e modo Maestro (`ASSIST` por padrão).

> O projeto ganha um **ID estável** (ex.: `prj_01J...`). Dois projetos com a mesma
> pasta base (`/home/a/api` e `/home/b/api`) são projetos **diferentes**.

### 3.3 Criar um Agente

Dentro do projeto:

1. Digite um nome (ex.: `Backend Developer`) em **Create Agent**.
2. O agente aparece com estado `STOPPED`.

### 3.4 Iniciar o Agente e abrir o Terminal

1. Clique em **Start**. O Nexus sobe uma *geração de runtime* supervisionada
   (processo persistente) e o estado vai para `WORKING`.
2. Clique em **Terminal** para abrir um terminal real (xterm.js) conectado ao
   agente.
3. Digite no terminal normalmente — é o terminal do provedor, dentro do navegador.

### 3.5 Parar e Recuperar

- **Stop** encerra a geração de runtime atual.
- Se a máquina reiniciar ou o processo morrer, o agente **continua existindo**
   (ele é persistente), mas o runtime não. O estado aparece como `RECOVERABLE`.
   Clique em **Recover** para tentar retomar a sessão (ou iniciar uma nova, de
   forma honesta e clara).

### 3.6 Navegação e atalhos

- **`Ctrl/Cmd + K`**: paleta de comandos (abrir projeto, iniciar agente, etc.).
- Menu lateral: **Overview · Projects · Agents · Resources · Maestro · Sessions ·
  Settings** e a área **Legacy** (Runtimes · Providers · Events).

---

## 4. Conceitos (entenda o modelo mental)

| Conceito | O que é | Por que importa |
|---|---|---|
| **Projeto** | A raiz do domínio. Tudo pertence a um projeto. | Suas sessões, agentes e layouts ficam organizados por projeto. |
| **Agente persistente** | Uma identidade estável (`agt_01J...`). | Sobrevive a reinício, troca de conta/provedor/modelo e reconnect de terminal. |
| **Geração de runtime** | Uma execução concreta do agente (um processo). | O agente é "eterno"; a geração é temporária. |
| **Revisão de configuração** | Versão imutável da config do agente. | Permite rollback seguro de configuração. |
| **Continuidade** | Como a sessão foi retomada (honesto). | `LIVE_SAME_RUNTIME`, `NATIVE_RESUME_VERIFIED/UNVERIFIED`, `CONTEXT_RECOVERED_NEW_SESSION`, etc. |
| **Maestro** | Camada de método/processo (comunidade). | Recomenda skills, processos, risco e verificação — o Nexus não duplica isso. |
| **Estado honesto** | O que é *observado*, não o que se *deseja*. | Nada de "STOPPED" antes da hora, nada de "VERIFIED" sem verificação. |

### Continuidade — lendo os estados

| Estado | Significado |
|---|---|
| `LIVE_SAME_RUNTIME` | Mesmo processo, conectado agora. |
| `REATTACHED_SAME_RUNTIME` | Reconectou ao mesmo processo (browser fechou e voltou). |
| `NATIVE_RESUME_VERIFIED` | Retomou e o provedor **confirmou** a sessão. |
| `NATIVE_RESUME_UNVERIFIED` | Retomou com a sessão do provedor, mas **não** é possível confirmar (honestidade). |
| `CONTEXT_RECOVERED_NEW_SESSION` | Recuperou o contexto em uma **sessão nova** (ex.: troca de provedor). |
| `CONTINUITY_FAILED` | A retomada falhou. |

> A IA nunca é apresentada como verificação: o Nexus não copia um ID de sessão
> para "provar" continuidade. Quando o provedor não permite verificação, o estado
> mostrado é `UNVERIFIED` — de propósito.

---

## 5. CLI — referência

O comando canônico é `nexus`, com o alias `ai` preservado para compatibilidade total.

### Comandos principais

```bash
nexus                  # Abre o Web Workspace OS (experiência padrão)
nexus web              # Abre explicitamente o Web Workspace OS
nexus start codex      # Inicia runtime supervisionado e conecta terminal
nexus stop <id>        # Encerra runtime supervisionado com segurança
nexus ps               # Lista runtimes supervisionados ativos
nexus attach <id>      # Reconecta terminal a um runtime existente
nexus handoff <id> ... # Handoff de conta (mesmo provedor)
nexus continue <id> ...# Handoff de contexto (novo provedor, NOVA sessão)
nexus version          # Versão, commit, build, go, plataforma
nexus doctor           # Diagnóstico completo de runtimes, chaveiros e Maestro
nexus providers        # Provedores detectados + capacidades honestas
nexus usage            # Quota/uso em tempo real (status honesto)
nexus profiles         # Perfis/contas configurados
```

### Modo clássico (lançamento direto de provedor)

```bash
nexus codex:work       # Codex no perfil "work"
nexus codex:auto       # Codex com seleção automática de conta
nexus claude           # Claude Code
nexus gemini           # Gemini CLI
nexus opencode         # OpenCode
nexus agy              # AGY / Antigravity
```

### Aliases Canônicos Universais & Interoperabilidade de Flags

O Nexus normaliza as flags mais comuns da comunidade e as traduz automaticamente para as opções nativas de cada provedor, tanto em `nexus start <provider>` quanto no modo direto (`nexus <provider>`). **As flags nativas de cada ferramenta continuam 100% suportadas sem qualquer conflito.**

| Alias Canônico | Descrição | Tradução no AGY | Tradução no Codex | Tradução no Claude |
|---|---|---|---|---|
| `--yolo` ou `-y` | Auto-aprova permissões e bypassa prompts | `--dangerously-skip-permissions` | `--dangerously-bypass-approvals-and-sandbox` | `--dangerously-skip-permissions` |
| `--continue` ou `-c` | Continua a conversa mais recente | `--continue` | `resume --last` | `--continue` |
| `--resume <id>` ou `-r <id>` | Retoma uma sessão específica | `--conversation=<id>` | `resume <id>` | `--resume <id>` |
| `--print` ou `-p` | Modo não-interativo (print output) | `--print` | `exec` | `--print` |
| `--effort <level>` | Esforço de raciocínio (low, medium, high) | `--effort <level>` | `-c model_reasoning_effort="<level>"` | — |
| `--plan` | Inicia o agente em modo de planejamento | `--mode plan` | — | — |
| `--accept-edits` | Inicia em modo de aceitação de edições | `--mode accept-edits` | — | — |

Exemplos práticos:
```bash
nexus agy --yolo                   # Dispara agy com --dangerously-skip-permissions
nexus start codex --yolo           # Inicia runtime supervisionado com bypass de sandbox/approvals
nexus codex -c                     # Continua a última conversa do Codex de forma imediata
nexus agy --resume 0192a...        # Conecta diretamente à conversa informada
```

#### Ajuda Integrada com Merge de CLI (`Merged Help`)

Para consultar as opções de qualquer provedor sem perder a visão dos aliases suportados pelo Nexus, use:
```bash
nexus agy --help       # ou: nexus help agy
nexus codex --help     # ou: nexus help codex
nexus claude --help    # ou: nexus help claude
```
O Nexus renderiza no topo uma tabela comparativa com os **aliases canônicos aplicáveis àquele CLI** e, logo abaixo, exibe o **help oficial nativo completo** do binário.

#### Aliases Customizados do Usuário

Você pode registrar aliases personalizados adicionais no arquivo `~/.config/nexus/config.json`:
```json
{
  "flag_aliases": {
    "--fast": {
      "agy": ["--effort", "low"],
      "codex": ["-c", "model_reasoning_effort=\"low\""]
    }
  }
}
```

### Slash `/nexus` dentro de um terminal supervisionado

Dentro de um runtime supervisionado, digite:

```text
/nexus status    /nexus usage    /nexus accounts    /nexus handoff codex:work
/nexus continue  /nexus detach   /nexus stop        /nexus help
```

- `/nexus ...` (e o alias `/ai ...`) é **interceptado pelo Nexus** (nunca vaza para o provedor — zero bytes).
- `/help`, `/model`, `/resume` seguem **normalmente para o provedor**.
- Para enviar um `/nexus` literal ao provedor, escreva `//nexus ...`.

---

## 6. Provedores e capacidades (transparência total)

| Provedor | Modo de controle | Resume | Observação honesta |
|---|---|---|---|
| Codex | `TERMINAL` | Sim (não verificado ao vivo) | Structured events/approvals **não** são anunciados como `SUPPORTED` — exigiriam adaptador `app-server` que não existe nesta versão. |
| Claude Code | `TERMINAL` | Sim | — |
| Gemini CLI | `TERMINAL` | Sim | — |
| OpenCode | `TERMINAL` | Sim (não verificado ao vivo) | O mesmo aviso honesto do Codex sobre eventos estruturados. |
| AGY / Antigravity | `TERMINAL` | Sim | Isolamento de credenciais dedicado. |
| Cursor Agent | `TERMINAL` | — | Detecção multi-caminho. |
| `fake` | `TERMINAL` | Sim | Provedor de **teste** (escondido da UI), usado para E2E e para você experimentar sem gastar quota. |

Regra de ouro: **capacidade efetiva = o provedor suporta ∧ o Nexus implementa ∧ a
plataforma suporta ∧ a versão é compatível ∧ o teste de runtime passou.** Nada de
botão falso, nada de claim inflado. "Allow?" no terminal **não** significa
"approvals programático".

---

## 7. Quota e uso — sem mentira

O Nexus exibe quota com **status honesto**:

```text
LIVE · CACHED · ESTIMATED · UNKNOWN · RATE_LIMITED · UNAVAILABLE
```

`UNKNOWN` **nunca** vira 100%. Cada leitura mostra fonte e frescor:

```bash
nexus usage
nexus usage codex --json
```

Dentro de um runtime: `/nexus usage`.

---

## 8. Contas, perfis e isolamento de credenciais

Cada perfil tem seu próprio `HOME`, `XDG_DATA_HOME`, `XDG_CONFIG_HOME` e — no caso
do AGY — sessão D-Bus privada e cofre de chaves dedicado. Tokens OAuth, chaves de
API e credenciais ficam isolados na pasta do perfil com permissões restritas.

> ⚠️ **Terminologia correta:** isso é **isolamento de credenciais / perfil
> isolado**, **não** um "sandbox hermético". O processo roda com as mesmas
> permissões do usuário; o que é isolado é o estado de configuração/credenciais.

Presets: `developer` (compartilha dotfiles/git), `strict` (isolamento completo),
`compat`.

---

## 9. Handoff — trocar de conta ou de provedor com segurança

### Account Handoff (mesmo provedor, outra conta/perfil)

```bash
nexus handoff <runtime-id> codex:work
```

É **transacional**: preflight → checkpoint (barreira) → quiesce do fonte →
**barreira de stop** (não inicia o alvo enquanto o fonte puder escrever na mesma
sessão) → inicia alvo → **verifica continuidade** (o comando de resume precisa
realmente referenciar a sessão) → atualiza lineage. Falhas caem em
`FAILED_SAFE` / `ROLLBACK`.

### Context Handoff (provedor diferente → NOVA sessão)

```bash
nexus continue <runtime-id> --with claude
```

Nunca é chamado de "resume": é uma **sessão nova** alimentada por um checkpoint
seguro (workspace, branch git, status, diff resumido) + prompt inicial. Segredos
são removidos por um pipeline de redação central.

---

## 10. Acesso remoto (privado e seguro)

O painel Web escuta em loopback por padrão. Para acessar de outra máquina:

### Via túnel SSH (recomendado)

```bash
# Na máquina A (onde os agentes rodam):
nexus web --no-open

# Na máquina B:
ssh -N -L 8080:127.0.0.1:<PORT> user@machine-a
# abra http://127.0.0.1:8080 e use o token bootstrap
```

### Via VPN privada (Tailscale/WireGuard/empresa)

```bash
nexus web --listen <ip-privado> --remote
```

`--remote` é um **opt-in explícito**. Endereços públicos (`8.8.8.8`, etc.) são
**recusados** (erro, não aviso). O range CGNAT `100.64.0.0/10` (Tailscale etc.) é
tratado como rede privada.

---

## 11. Segurança

| Controle | Estado |
|---|---|
| Loopback default (`127.0.0.1` / `::1`) | ✅ |
| Bind público | ❌ recusado (não apenas warning) |
| Bind privado | ✅ só com `--remote` explícito |
| Token bootstrap criptográfico de uso único | ✅ |
| Cookie de sessão `HttpOnly` + `SameSite=Strict` | ✅ |
| CSRF em REST de escrita | ✅ |
| Validação de `Origin` em REST e WebSocket | ✅ |
| WebSocket autenticado | ✅ |
| Sem CORS wildcard | ✅ |
| CSP + `nosniff` + `Referrer-Policy` + `frame-ancestors 'none'` | ✅ |
| Terminal/provedor nunca renderizado como HTML cru | ✅ |
| Canonicalização de caminho de projeto (Abs → EvalSymlinks) | ✅ |
| Isolamento IDOR por projeto (agente de A inacessível via B) | ✅ |
| Redação de segredos (keys, tokens, JWT, cookies, `.env`, PEM) | ✅ |
| Framing IPC limitado (sem alocação ilimitada) | ✅ |
| Versão de protocolo verificada | ✅ |

---

## 12. Dados e estado

O Nexus usa **SQLite local** (driver 100% Go, sem CGO, binário único e portátil)
para o estado durável do produto:

```text
projects · agents · agent_revisions · runtime_generations · lineage ·
events_metadata · maestro_advice · verification_evidence · project_layouts
```

O arquivo fica em `<data-dir>/nexus.db`. O estado vivo de runtime (PIDs,
sockets) continua no registry em memória/disco (`runtimes.json`).

**O que o SQLite NUNCA guarda:** chaves de API, tokens OAuth, `auth.json`,
cookies, segredos de provedor, chaves privadas, transcript completo do terminal.

### Caminhos

| Uso | Local padrão |
|---|---|
| Dados (SQLite, registry, logs) | Linux/macOS: `~/.local/share/ai-manager` · Windows: `%LOCALAPPDATA%` |
| Configuração | `~/.config/ai-manager` (respeita `XDG_CONFIG_HOME`) |
| Sobrescrever | env `AI_CLI_DATA_DIR`, `AI_CLI_CONFIG_DIR`, `AI_CLI_STATE_DIR` |

---

## 13. Suporte a plataformas e evidência de release

O código e o pipeline têm como alvo Linux, Windows e macOS em amd64/arm64. A
matriz obrigatória de CI executa Go 1.25, testes, runtime E2E e build nativo em
`ubuntu-latest`, `windows-latest` e `macos-latest`; o snapshot do GoReleaser só
roda depois dos três jobs.

| Plataforma alvo | Runtime coberto pela CI | Artefato de release | Evidência nesta cópia local |
|---|---|---|---|
| Linux amd64/arm64 | PTY, socket, SessionHost e Web | `tar.gz` | Frontend + unidades Go offline disponíveis; suíte Go 1.25 completa requer CI |
| Windows amd64/arm64 | ConPTY, Named Pipe, SessionHost e Web | `.zip` | requer job `windows-latest` |
| macOS amd64/arm64 | PTY, socket, SessionHost e Web | `tar.gz` | requer job `macos-latest` |

**Regra de release:** uma plataforma só deve ser anunciada como validada para a
versão quando o job nativo correspondente estiver verde. Build cruzado, sozinho,
não é evidência de runtime.

---

## 14. Solução de problemas (FAQ)

**O painel não abre.**
```bash
ai doctor
```
Confira que a porta não está ocupada e que o token bootstrap foi usado na URL.

**`could not open a new TTY`**
A TUI opcional de compatibilidade (`nexus --tui` ou `nexus control`) precisa de um terminal real. Use `nexus`/`nexus web` em sessões sem TTY.

**O agente está `RECOVERABLE`.**
O processo morreu (reboot/crash). O agente está intacto — clique em **Recover**.

**`refusing to bind to public address`**
Você tentou `--listen <ip-público>`. Use túnel SSH ou `--remote` com IP privado.

**Quota mostra `UNKNOWN`.**
O provedor não expõe a fonte. Isso é honesto, não bug.

**O terminal do agente desconectou.**
O browser fechou? O runtime continua. Reabra e o terminal reconecta (replay
limitado). Se o runtime morreu, o estado vira `RECOVERABLE`.

**Preciso de ajuda com o Maestro.**
O Maestro é o repositório irmão [Orquestrador-Maestro](https://github.com/IAPro-Community/Orquestrador-Maestro).
No Nexus, o Maestro é opcional. Se a integração estiver indisponível, o estado é `MAESTRO_DEGRADED`; o Nexus não fabrica skills, gates ou recomendações.

---

## 15. Estado funcional da candidata

A candidata atual integra: Workspace OS Web-first, trabalho Direct sem Mission,
Agents persistentes, terminal supervisionado e reconexão, Safe Apply, seleção de
recursos/quotas, Intelligence explícita, integração Maestro degradável, WorkPlan
versionado, clarificações bloqueantes, Mission Runner durável, DAG/paralelismo,
worktrees por pacote, review independente, remediação governada, scheduling e
Take Control/Return to Mission.

Isso **não substitui os gates de release**. Antes de publicar uma versão, execute
a matriz completa descrita em `.github/workflows/ci.yml` e o roteiro em
`DEV/NEXUS_FINAL_LOCAL_VALIDATION_PROMPT.md`.

---

## 16. Desenvolvimento

```bash
# Backend
go vet ./...
go test -race ./...

# Frontend (Bun é o package manager canônico; lockfile em web/bun.lock)
cd web
bun install --frozen-lockfile
bun run typecheck
bun run lint
bun run test
bun run build      # gera o frontend embutido (internal/control/web/embedded)

# Release snapshot
goreleaser release --snapshot --clean
```

> O binário final é **único** — o frontend é embutido no binário Go. Node/Bun não
> são necessários na máquina do usuário final.

---

## 17. Licença

Apache-2.0. Projeto da comunidade **IAPro Community**.

---

*IAPro Nexus · Powered by Orquestrador Maestro · Community-first. Se um recurso estiver
marcado como "chega no Gate N", é porque ainda não foi construído — o Nexus não
promete o que não existe.*
