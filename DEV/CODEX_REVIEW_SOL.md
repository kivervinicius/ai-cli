# Revisão de engenharia — IAPro AI Control

**Commit auditado:** `1342069`  
**Branch:** `feat/ai-control-web-remote`  
**Data:** 28 de agosto de 2026  
**Recomendação:** **REQUEST CHANGES — não liberar para produção pública**

## Sumário executivo

O projeto possui uma arquitetura promissora e vários controles corretos: `SessionHost` desacoplado, PTY Unix real, ring buffer limitado, fanout não bloqueante, IPC local com permissões restritas, persistência atômica e redação abrangente de segredos.

Entretanto, o Web Control Center introduz riscos incompatíveis com um lançamento público:

- Terminal WebSocket sem autenticação.
- Possibilidade de controle arbitrário de uma sessão ativa.
- Ambiente completo do processo, potencialmente contendo segredos, persistido em `runtimes.json` e retornado pela API.
- Registro persistente sujeito a perda de atualizações entre processos.
- Lease do navegador divergente do lease efetivo no `SessionHost`.
- Contratos TypeScript incompatíveis com as estruturas Go.
- Implementação Windows anunciada como ConPTY, mas operando apenas com pipes.

Classificação geral:

| Dimensão | Avaliação |
|---|---|
| Backend Go local | Boa base, ainda com falhas multiprocesso e de lifecycle |
| IPC/terminal Unix | Funcional, mas framing e lease precisam de reforço |
| Windows/ConPTY | Incompleto |
| Web Control Center | Alto risco |
| Isolamento de credenciais | Parcial; não constitui sandbox real |
| Testes backend | Razoáveis, mas insuficientes nos cenários críticos |
| Testes frontend | Ausentes |
| Maturidade | Pré-produção avançada / beta interna |

---

# P0 — Críticos

## P0.1 — Terminal WebSocket ignora autenticação e permite tomada de controle

A rota WebSocket é desviada diretamente para `HandleWebSocket`, sem passar pelo middleware de autenticação: [server.go:129](/projetos/tools/ai-manager/internal/control/web/server.go:129) e [server.go:135](/projetos/tools/ai-manager/internal/control/web/server.go:135).

O handler apenas verifica a existência do runtime e faz o upgrade: [handler_terminal.go:46](/projetos/tools/ai-manager/internal/control/web/handler_terminal.go:46). Não valida:

- Cookie de sessão.
- Bootstrap previamente consumido.
- CSRF ou token equivalente para o WebSocket.
- Autorização do usuário para aquele runtime.

Além disso, qualquer conexão pode sobrescrever o lease em [handler_terminal.go:174](/projetos/tools/ai-manager/internal/control/web/handler_terminal.go:174).

A validação de origem também é vulnerável por usar prefixos textuais:

- [auth.go:64](/projetos/tools/ai-manager/internal/control/web/auth.go:64)
- [handler_terminal.go:21](/projetos/tools/ai-manager/internal/control/web/handler_terminal.go:21)

Origens como `http://localhost.evil.example` e `http://127.0.0.1.attacker.example` passam no teste. Origem vazia também é aceita.

**Impacto:** leitura do terminal, envio de comandos ao agente e possível execução arbitrária com os privilégios do usuário que iniciou o Control Center.

**Gravidade:** crítica no modo `--listen` ou quando a porta é previsível/exposta; ainda relevante em loopback por permitir cross-site WebSocket hijacking.

**Correção obrigatória:**

1. Exigir sessão autenticada antes do upgrade.
2. Comparar `scheme`, `hostname` e porta após `url.Parse`, nunca por prefixo.
3. Rejeitar `Origin` vazio no fluxo Web.
4. Desabilitar bind não-loopback por padrão; exigir flag explícita acompanhada de autenticação forte.
5. Tratar aquisição de lease como operação autorizada e transacional.

## P0.2 — Segredos do ambiente são persistidos e expostos por GET não autenticado

`RuntimeSession` afirma que segredos nunca são persistidos, mas contém `Env` serializável: [models.go:29](/projetos/tools/ai-manager/internal/control/registry/models.go:29) e [models.go:45](/projetos/tools/ai-manager/internal/control/registry/models.go:45).

Os drivers constroem esse campo a partir de `os.Environ()`:

- [codex_driver.go:143](/projetos/tools/ai-manager/internal/control/driver/codex_driver.go:143)
- [claude_driver.go:143](/projetos/tools/ai-manager/internal/control/driver/claude_driver.go:143)
- [gemini_driver.go:143](/projetos/tools/ai-manager/internal/control/driver/gemini_driver.go:143)

O launcher o coloca na sessão persistente em [launcher.go:89](/projetos/tools/ai-manager/internal/control/launcher/launcher.go:89), e o registry serializa todo o objeto em [registry.go:91](/projetos/tools/ai-manager/internal/control/registry/registry.go:91).

A API devolve as sessões completas:

- Lista: [handlers_api.go:123](/projetos/tools/ai-manager/internal/control/web/handlers_api.go:123)
- Detalhe: [handlers_api.go:205](/projetos/tools/ai-manager/internal/control/web/handlers_api.go:205)

Para GET, o middleware não exige autenticação; ela é aplicada somente a métodos de escrita em [server.go:152](/projetos/tools/ai-manager/internal/control/web/server.go:152). Clientes sem `Origin` também são aceitos em [auth.go:50](/projetos/tools/ai-manager/internal/control/web/auth.go:50).

**Dados potencialmente expostos:**

- API keys herdadas.
- Tokens de registries e proxies.
- Credenciais cloud.
- Variáveis de CI.
- Tokens GitHub, OpenAI, Anthropic ou Google.
- Argumentos contendo prompts ou conteúdo operacional sensível.

**Correção obrigatória:**

- Remover `Env` do modelo persistente.
- Não retornar `Args`, `Binary`, `Env` ou endpoint interno diretamente na API pública.
- Criar DTOs específicos para persistência e API.
- Construir um ambiente mínimo por allowlist, em vez de herdar `os.Environ()`.
- Exigir autenticação para todos os endpoints, inclusive GET.

---

# P1 — Altos

## P1.1 — Registry perde atualizações entre processos

O file lock evita gravações simultâneas no arquivo, mas não torna o read-modify-write transacional.

Cada processo carrega o arquivo apenas ao criar o registry em [registry.go:40](/projetos/tools/ai-manager/internal/control/registry/registry.go:40). Posteriormente, `saveLocked()` bloqueia o arquivo e grava exclusivamente o mapa local, sem recarregar o estado mais recente: [registry.go:78](/projetos/tools/ai-manager/internal/control/registry/registry.go:78).

Cenário:

1. Processo A e processo B carregam o mesmo snapshot.
2. A registra runtime A e salva.
3. B registra runtime B em seu mapa antigo.
4. B salva e apaga logicamente o runtime A.

Isso afeta justamente o modelo de vários hosts destacados.

**Correção:** sob o lock de arquivo, recarregar o estado atual, aplicar a mutação e gravar a nova versão; alternativamente usar SQLite com transações.

## P1.2 — Lease Web e lease IPC possuem fontes de verdade diferentes

O navegador controla `TerminalHub.leases` em [handler_terminal.go:95](/projetos/tools/ai-manager/internal/control/web/handler_terminal.go:95). O host controla outro lease em `SessionHost.activeWriter`: [host.go:263](/projetos/tools/ai-manager/internal/control/host/host.go:263) e [host.go:325](/projetos/tools/ai-manager/internal/control/host/host.go:325).

Ao fazer `lease_release`, a conexão IPC continua sendo o `activeWriter`. Outro navegador pode receber visualmente `CONTROL`, mas seu input será descartado silenciosamente pelo host.

O teste existente de handover verifica apenas se `Write` retorna sem erro, não se os bytes chegaram ao processo: [qa_test.go:77](/projetos/tools/ai-manager/internal/control/host/qa_test.go:77) e [qa_test.go:134](/projetos/tools/ai-manager/internal/control/host/qa_test.go:134).

Além disso, o goroutine de saída e o loop principal fazem `WriteJSON` concorrente na mesma conexão:

- [handler_terminal.go:123](/projetos/tools/ai-manager/internal/control/web/handler_terminal.go:123)
- [handler_terminal.go:178](/projetos/tools/ai-manager/internal/control/web/handler_terminal.go:178)
- [handler_terminal.go:189](/projetos/tools/ai-manager/internal/control/web/handler_terminal.go:189)

Isso viola o modelo de um único writer do Gorilla WebSocket.

**Correção:** lease único no `SessionHost`, comandos IPC explícitos `acquire/release`, geração/epoch do lease e um writer goroutine por WebSocket.

## P1.3 — Falhas parciais deixam processos e recursos órfãos

`SessionHost.Start()` inicia o processo em [host.go:95](/projetos/tools/ai-manager/internal/control/host/host.go:95), mas só depois abre o listener IPC em [host.go:123](/projetos/tools/ai-manager/internal/control/host/host.go:123). Se `Listen` falhar, retorna sem matar ou aguardar o processo.

Outros problemas:

- Falha de handshake apenas marca `FAILED`, sem terminar o host criado: [launcher.go:139](/projetos/tools/ai-manager/internal/control/launcher/launcher.go:139).
- O processo destacado retornado por `cmd.Start()` não recebe `Wait` ou `Release`: [spawn_unix.go:40](/projetos/tools/ai-manager/internal/control/launcher/spawn_unix.go:40).
- `Stop` sinaliza apenas o PID principal, não a árvore/grupo do processo: [host.go:440](/projetos/tools/ai-manager/internal/control/host/host.go:440).
- `CleanupStale` verifica o PID do provider, não o `HostPID`: [cleanup.go:22](/projetos/tools/ai-manager/internal/control/registry/cleanup.go:22).

**Correção:** lifecycle transacional com rollback, process groups/job objects, cleanup idempotente e estado explícito `STARTING → RUNNING` somente após listener e handshake.

## P1.4 — Protocolo IPC possui framing ambíguo e limites tardios

O servidor usa `ReadBytes('\n')` e só testa o tamanho depois da alocação: [host.go:175](/projetos/tools/ai-manager/internal/control/host/host.go:175) e [host.go:182](/projetos/tools/ai-manager/internal/control/host/host.go:182). Um frame sem newline pode crescer arbitrariamente antes da rejeição.

No modo attached, bytes iniciados por `{"version":` ou `{"command":` são reinterpretados como RPC: [host.go:216](/projetos/tools/ai-manager/internal/control/host/host.go:216). Conteúdo legítimo digitado no terminal pode ser interceptado como controle.

Também há duas rotas de escrita concorrentes na mesma conexão IPC:

- Respostas RPC em [host.go:316](/projetos/tools/ai-manager/internal/control/host/host.go:316).
- Output do fanout em [fanout.go:97](/projetos/tools/ai-manager/internal/control/host/fanout.go:97).

Durante `attach`, o output pode chegar antes da resposta JSON e quebrar `Client.Send`.

O protocolo declara versão e IDs, mas não valida `req.Version` nem correlaciona `Request.ID` com `Response.ID`.

**Correção:** framing binário ou header `{type,length}`, limite antes da alocação, canais separados para controle/dados e writer serializado.

## P1.5 — “Isolamento” não é sandbox e o preset padrão expõe configurações do host

O preset padrão é `developer`: [config.go:47](/projetos/tools/ai-manager/internal/core/config/config.go:47).

Nesse preset, `ApplyIsolation` percorre praticamente todo `~/.config` e cria links ou cópias no perfil: [isolation.go:186](/projetos/tools/ai-manager/internal/core/security/isolation.go:186) e [isolation.go:190](/projetos/tools/ai-manager/internal/core/security/isolation.go:190).

Isso pode incluir credenciais de ferramentas não previstas. Adicionalmente, todos os drivers herdam o ambiente do host.

Mesmo com `HOME` separado, o processo:

- Executa com o mesmo UID.
- Mantém acesso ao filesystem permitido ao usuário.
- Não possui namespaces, seccomp, container, chroot ou política MAC.
- Pode ler outros diretórios do usuário.

Portanto, o mecanismo atual é **separação de home/configuração**, não isolamento de sandbox.

**Correção:** documentar corretamente o threat model; tornar `strict` o padrão; usar allowlist de variáveis e diretórios; considerar sandbox de SO opcional.

## P1.6 — Contratos Go/TypeScript quebram funcionalidades reais

### Runtime

Go envia `provider_id` e `profile_id`, mas os modais usam `runtime.provider` e `runtime.profile`:

- [HandoffModal.tsx:20](/projetos/tools/ai-manager/web/src/components/HandoffModal.tsx:20)
- [HandoffModal.tsx:33](/projetos/tools/ai-manager/web/src/components/HandoffModal.tsx:33)
- [ContinueModal.tsx:21](/projetos/tools/ai-manager/web/src/components/ContinueModal.tsx:21)

Resultado provável: lista vazia, `undefined:perfil` e seleção do próprio provider como destino.

### Perfis

`/profiles` retorna `[]model.Profile`: [handlers_api.go:316](/projetos/tools/ai-manager/internal/control/web/handlers_api.go:316). Esse modelo contém apenas provider, nome, criação, prioridade e labels: [types.go:106](/projetos/tools/ai-manager/internal/core/model/types.go:106).

O frontend espera `authenticated`, `is_default`, `account_email` e `plan`: [types.ts:65](/projetos/tools/ai-manager/web/src/types.ts:65).

### Eventos

Go envia `provider` e `profile`: [events.go:35](/projetos/tools/ai-manager/internal/control/events/events.go:35). O frontend espera `provider_id` e `profile_id`: [types.ts:77](/projetos/tools/ai-manager/web/src/types.ts:77).

### Estados

Go define `WAITING`, `APPROVAL`, `DETACHED` e `STOPPING`: [models.go:9](/projetos/tools/ai-manager/internal/control/registry/models.go:9). O union TypeScript os omite: [types.ts:23](/projetos/tools/ai-manager/web/src/types.ts:23).

**Correção:** gerar tipos a partir de OpenAPI/JSON Schema ou manter DTOs contratuais com testes de golden JSON.

## P1.7 — Validação insuficiente de nomes usados em caminhos e sockets

`ProfileRoot` concatena provider e nome diretamente: [config.go:244](/projetos/tools/ai-manager/internal/core/config/config.go:244). O Web launcher não chama `ValidateProfileName` antes de repassar `profile`.

A validação existente permite pontos e não rejeita explicitamente `.` ou `..`: [config.go:430](/projetos/tools/ai-manager/internal/core/config/config.go:430).

Runtime IDs também são usados diretamente no nome do socket: [endpoint_common.go:22](/projetos/tools/ai-manager/internal/control/protocol/endpoint_common.go:22). Antes de ouvir, o código remove incondicionalmente o caminho calculado: [endpoint_unix.go:23](/projetos/tools/ai-manager/internal/control/protocol/endpoint_unix.go:23).

Somado aos IDs baseados em módulo de tempo ou `len(reg.List())+1`:

- [launcher.go:52](/projetos/tools/ai-manager/internal/control/launcher/launcher.go:52)
- [account.go:142](/projetos/tools/ai-manager/internal/control/handoff/account.go:142)
- [context.go:112](/projetos/tools/ai-manager/internal/control/handoff/context.go:112)

há risco de colisão e remoção do socket de um runtime ainda ativo.

**Correção:** IDs UUID/ULID criptograficamente aleatórios, allowlist estrita e verificação de existência antes de registrar ou remover socket.

## P1.8 — Suporte Windows não entrega ConPTY

O arquivo carrega símbolos de ConPTY, mas `Start` usa pipes comuns: [terminal_windows.go:51](/projetos/tools/ai-manager/internal/control/terminal/terminal_windows.go:51), marcando `isConPTY = false` em [terminal_windows.go:68](/projetos/tools/ai-manager/internal/control/terminal/terminal_windows.go:68).

Consequências:

- Sem terminal VT interativo real.
- Resize inoperante.
- Raw mode indisponível.
- CLIs TUI podem alterar comportamento ou falhar.

Isso deve ser tratado como feature não implementada, não como fallback equivalente.

---

# P2 — Melhorias técnicas

## P2.1 — Workspace store não cumpre ordenação e não é multiprocesso

`List()` promete ordenar por `LastUsedAt`, mas apenas itera sobre um map: [workspace.go:128](/projetos/tools/ai-manager/internal/control/workspace/workspace.go:128).

IDs são derivados somente do nome: [workspace.go:216](/projetos/tools/ai-manager/internal/control/workspace/workspace.go:216). Projetos homônimos produzem IDs ambíguos.

O store também não possui file lock, embora Web, CLI e hosts possam acessá-lo em processos diferentes.

## P2.2 — Abas e layout responsivo possuem inconsistências

`openIds` é mantido em estado, mas a barra renderiza todos os runtimes: [TerminalView.tsx:20](/projetos/tools/ai-manager/web/src/components/TerminalView.tsx:20) e [TerminalView.tsx:71](/projetos/tools/ai-manager/web/src/components/TerminalView.tsx:71). Fechar uma aba não a remove visualmente.

Split e grid escolhem simplesmente os primeiros runtimes, ignorando abas abertas: [TerminalView.tsx:59](/projetos/tools/ai-manager/web/src/components/TerminalView.tsx:59).

A sidebar fixa de 256 px e as linhas de runtime sem overflow horizontal degradam telas pequenas: [Sidebar.tsx:48](/projetos/tools/ai-manager/web/src/components/Sidebar.tsx:48) e [Dashboard.tsx:98](/projetos/tools/ai-manager/web/src/components/Dashboard.tsx:98).

A identidade IAPro está aplicada de forma coerente em marca, cores e links, mas ainda depende fortemente de um visual Tailwind genérico.

## P2.3 — Checkpoint não é realmente limitado

Embora `ChangedFiles` seja limitado a 50, `GitStatus` armazena todo o output antes do corte: [checkpoint.go:71](/projetos/tools/ai-manager/internal/control/handoff/checkpoint.go:71) e [checkpoint.go:73](/projetos/tools/ai-manager/internal/control/handoff/checkpoint.go:73).

`exec.Command(...).Output()` também materializa toda a saída em memória. O diff é truncado somente após a alocação: [checkpoint.go:88](/projetos/tools/ai-manager/internal/control/handoff/checkpoint.go:88).

## P2.4 — Pipeline de release não valida o frontend

A CI executa vet, testes e build apenas do Go: [.github/workflows/ci.yml:36](/projetos/tools/ai-manager/.github/workflows/ci.yml:36).

Não existem:

- Build do frontend.
- `tsc --noEmit`.
- Testes React.
- Playwright.
- Audit de dependências frontend.
- Verificação de sincronismo entre `web/src` e o bundle embutido.

O `go.mod` requer Go 1.25, enquanto a matriz declara 1.22 e 1.24: [go.mod:3](/projetos/tools/ai-manager/go.mod:3) e [.github/workflows/ci.yml:23](/projetos/tools/ai-manager/.github/workflows/ci.yml:23).

Embora exista um `web/bun.lock` local, ele está explicitamente ignorado em [.gitignore:48](/projetos/tools/ai-manager/.gitignore:48). A documentação oficial do Bun recomenda versionar o lockfile e usar `bun ci`/`--frozen-lockfile` para builds reproduzíveis. [Bun lockfile](https://bun.sh/docs/pm/lockfile)

## P2.5 — Frontend “embutido” ainda depende de CDN

O HTML embutido carrega Tailwind e CSS do xterm de serviços externos:

- [index.html:7](/projetos/tools/ai-manager/internal/control/web/dist/index.html:7)
- [index.html:8](/projetos/tools/ai-manager/internal/control/web/dist/index.html:8)

Não há CSP nem SRI. Isso:

- Quebra operação offline.
- Amplia a cadeia de suprimentos.
- Permite que código remoto seja executado numa UI capaz de controlar terminais.

Os estilos devem fazer parte do bundle embutido.

## P2.6 — Redação existe, mas não é uma política central

`security.Redact` cobre vários formatos importantes: [redact.go:9](/projetos/tools/ai-manager/internal/core/security/redact.go:9). Porém, sua aplicação é pontual, principalmente em checkpoint e diagnóstico.

Ela não protege:

- `RuntimeSession.Env`.
- Respostas completas da API.
- Todos os erros retornados pelo launcher/provider.
- Campos futuros de eventos ou logs.

A redação deve ocorrer numa camada central de logging e serialização, acompanhada de testes negativos.

---

# Pontos fortes confirmados

- Ring buffer com limite fixo e cópia segura: [ringbuffer.go:26](/projetos/tools/ai-manager/internal/control/host/ringbuffer.go:26).
- Fanout com filas individuais e drop não bloqueante para observadores lentos: [fanout.go:67](/projetos/tools/ai-manager/internal/control/host/fanout.go:67).
- Socket Unix criado sob `umask 0177` e reforçado para `0600`: [endpoint_unix.go:25](/projetos/tools/ai-manager/internal/control/protocol/endpoint_unix.go:25).
- Named Pipe Windows com ACL owner-only: [endpoint_windows.go:13](/projetos/tools/ai-manager/internal/control/protocol/endpoint_windows.go:13).
- Cliente IPC com deadline de cinco segundos e resposta limitada a 1 MiB: [client.go:43](/projetos/tools/ai-manager/internal/control/protocol/client.go:43) e [client.go:74](/projetos/tools/ai-manager/internal/control/protocol/client.go:74).
- Persistência usa arquivo temporário `0600` e rename: [registry.go:101](/projetos/tools/ai-manager/internal/control/registry/registry.go:101).
- Perfis e checkpoints são normalmente criados com `0700/0600`: [profile.go:70](/projetos/tools/ai-manager/internal/profile/profile.go:70) e [checkpoint.go:114](/projetos/tools/ai-manager/internal/control/handoff/checkpoint.go:114).
- Bootstrap usa aleatoriedade criptográfica, cookie `HttpOnly` e `SameSite=Strict`: [auth.go:30](/projetos/tools/ai-manager/internal/control/web/auth.go:30) e [server.go:87](/projetos/tools/ai-manager/internal/control/web/server.go:87).
- Não encontrei segredo real hardcoded no código versionado; os padrões encontrados estavam em fixtures de teste.

---

# Testes e cobertura

## Situação real

O repositório contém:

- 112 arquivos Go.
- 29 arquivos `*_test.go`.
- **48 funções `Test...`**, não 47.
- 13 arquivos TypeScript/TSX no frontend.
- Nenhum teste frontend.

Não foi possível executar uma validação fresca de `go test -race ./...` neste ambiente. O toolchain falhou antes da compilação:

```text
go: creating work dir: mkdir /tmp/go-build...: read-only file system
```

Portanto, a declaração de “47 testes passando” não foi reproduzida e está desatualizada em relação à contagem estática atual de 48.

## Cobertura existente positiva

A suíte inclui bons testes de:

- Lifecycle básico do host.
- Deadlock previamente identificado em `CmdInput`.
- Observador lento.
- Attach/detach concorrente.
- Lease básico.
- Persistência sequencial do registry.
- Cleanup e purge.
- Bootstrap, CSRF e origem inválida simples.
- Redação e políticas de isolamento.
- Handoff e rollback.

## Lacunas prioritárias

Devem ser adicionados antes do release:

1. WebSocket sem cookie deve retornar `401`.
2. `Origin: http://localhost.evil.example` deve ser rejeitado.
3. GET sem sessão deve retornar `401`.
4. Dois browsers devem transferir lease e comprovar bytes recebidos pelo processo.
5. Output + mudança de lease sob concorrência para detectar múltiplos writers WebSocket.
6. Registry com dois processos fazendo mutações concorrentes.
7. Falha do listener após spawn deve provar que o child foi encerrado.
8. Frame IPC sem newline acima do limite.
9. Output do processo durante resposta de `attach`.
10. Colisão de RuntimeID não pode remover socket ativo.
11. Perfil `..`, `.`, caminho absoluto e separadores devem ser rejeitados.
12. Windows ConPTY com TUI real, resize e Unicode.
13. Testes de contrato JSON Go ↔ TypeScript.
14. React tests para handoff, continue e fechamento de abas.
15. Playwright para terminal, resize, reconexão e responsividade.
16. Teste garantindo que `Env` e segredos nunca aparecem em arquivo ou API.

---

# Veredito final

O projeto ainda **não deve ser publicado como release de produção na IAPro-Community**.

A base Go é tecnicamente interessante e demonstra bom trabalho de arquitetura, mas o Control Center controla processos com os privilégios do usuário. Nesse domínio, autenticação, lease, IPC e persistência de segredos são fronteiras críticas, não melhorias opcionais.

Critérios mínimos para uma nova avaliação:

1. Autenticação obrigatória em REST e WebSocket.
2. Remoção de `Env`/`Args` brutos da persistência e da API.
3. Registry transacional multiprocesso.
4. Lease único governado pelo `SessionHost`.
5. Writer único em IPC e WebSocket.
6. Cleanup transacional de processos e sockets.
7. Tipos Go/TypeScript alinhados.
8. Build, typecheck e testes frontend na CI.
9. Posicionamento honesto do Windows como “pipes/incompleto” até existir ConPTY real.
10. Testes negativos cobrindo todos os bloqueadores.

**Maturidade atual:** beta interna / pré-produção avançada.  
**Postura de segurança:** alto risco.  
**Decisão de release:** **bloqueado até correção dos P0 e P1**.