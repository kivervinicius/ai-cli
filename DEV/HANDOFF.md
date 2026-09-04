# Handoff

## Current state

Nexus was rebuilt from the current source and installed at `/home/desenvolvedor/.local/bin/nexus`; the `ai` alias points to the same binary. The installed binary is `0.5.0-beta.23`.

## Verification

Fresh `zsh` verification and focused Nexus Go tests passed. The rebuilt
`nexus web` process is currently active on `http://127.0.0.1:3000`.

## Revisão atual — 2026-09-04

Revisão técnica registrada em [`DEV/validation/CURRENT_CODE_REVIEW.md`](validation/CURRENT_CODE_REVIEW.md).
O estado é **aprovado para uso local individual em loopback**. O frontend evita
polling concorrente e polling em background; o handler de terminal encerra
recursos de forma idempotente e limita resize concorrente. Build final instalado:
`nexus v0.5.0-beta.23`; Web ativo em `http://127.0.0.1:3000`.

Riscos que continuam explícitos: `--remote` usa HTTP sem TLS (risco aceito para
LAN privada), verificações de missão executam shell confiado sem sandbox hostil,
e macOS/Windows/Safari só têm evidência de compilação/configuração, não execução
local nesta máquina. Não há app nativo iOS.

Os controles dos terminais agora exibem estados de aplicação/reinício/pronto/erro,
feedback de lease e confirmação explícita para fechar mantendo ou parando o
runtime. Project Shells confirmam o encerramento do processo. O badge superior
resume trabalho, espera, desconexão e degradação; clique em runtime vivo foca o
terminal e clique em desconectado abre Agentes para recuperação.

Última validação: `make web-verify` e `make build` passaram em 2026-09-03.

O Codex agora aparece como recurso compatível no modo Coding CLI quando autenticado,
usando `codex exec` para prompts não interativos. O servidor foi reiniciado após o
build e está ativo em `http://127.0.0.1:3000`.

A confirmação de modo ocorre antes da persistência/reinício; o fechamento de abas,
Project Shells e runtimes usa confirmação Nexus centralizada. Bootstrap atual:
`http://127.0.0.1:3000/?token=ce1e105d4306712f28fa7369c32905d249ab0381f506cce49f46e916842d520c`.

AGY agora mantém o modelo por perfil e recebe o diretório privado do keyring
explicitamente. O binário instalado é `nexus v0.5.0-beta.17`; reinicie sessões
AGY antigas para usar o wrapper corrigido.

CLIs interativos não recebem mais `TERM=dumb` quando o Nexus possui PTY real;
isso evita a confirmação bloqueada “Continue anyway?” do Codex e cobre também
agentes supervisionados.

O atalho `Sessão IA` agora abre o Composer e o launcher de sessão depois da
montagem da superfície, evitando perder o evento no primeiro clique. Na shell,
`Controle` indica quem pode digitar; `Liberar` devolve essa permissão e
`Assumir controle` solicita a permissão quando a sessão está em somente leitura.

Terminais de Agentes desconectados usam imediatamente o `runtime_id` retornado
por Recover/Start, sem esperar o polling global. O botão `Fechar terminal` remove
a aba visual; o Agente persistente continua separado até ser parado em Agentes.

Quando um projeto não tem contexto durável, o Composer informa que criará um
`AGENTS.md` base. A criação só ocorre ao clicar no botão, é atômica e não
substitui `AGENTS.md`, `DEV/INDEX.md` ou `DEV/CONTEXT.md` existentes.

Para AGY, a disponibilidade da conta exige quota nos dois grupos de modelos
(Claude/GPT e Gemini). Uma conta com somente um grupo disponível aparece como
indisponível e não deve ser escolhida pelo scheduler; os badges dos grupos
continuam exibindo o detalhe de cada pool.

O erro de runtime `Cannot read properties of null (reading 'forEach')` foi
corrigido no Flow: payloads com `phases` ou `packages` nulos são tratados como
vazios, e o bundle embutido foi recompilado. O teste de regressão e `yarn build`
passaram; o typecheck também passou na validação final.

A proteção foi centralizada em `web/src/nexus/workPlan.ts`; os endpoints de
planos normalizam as respostas na entrada, evitando que novas telas repitam a
mesma vulnerabilidade.

Terminais de Agentes também foram ajustados para não fixarem uma geração
antiga: após Recover/Start, a superfície resolve novamente pelo `agent_id` e o
bundle embutido foi recompilado.

`nexus usage` foi recompilado para mostrar a capacidade completa por modelo e
os resets de cada janela (5h/semanal), sem reduzir a saída ao bottleneck.

Sessões Web rotacionam automaticamente antes do vencimento. Se o servidor for
reiniciado ou a sessão ficar ociosa além do limite, a interface explica como
abrir o novo Bootstrap exibido por `nexus web`.

O uso de recursos é apresentado em tabela no CLI e em grid responsivo no
dashboard, com capacidade/reset por grupo e status sem sobreposição visual.
Para localizar contas e grupos no terminal, use `nexus usage` e
pressione `/` para filtrar.

O banner de atenção do agente também foi corrigido: ele não ocupa mais a área
central do dashboard e a mensagem padrão `? for shortcuts` não abre aviso
falso. O binário instalado contém esses ajustes.

Conclusões de tarefa e erros usam agora um componente de notificações
transitórias próprio; perguntas e aprovações continuam visíveis no banner por
exigirem resposta.

O motor de Flags Canônicas e Merged Help (`internal/control/flags`) foi incorporado
com sucesso: comandos universais (`--yolo`, `-y`, `--continue`, `-c`, `--resume`, `-r`,
`--print`, `-p`, `--effort`, `--plan`, `--accept-edits`) são traduzidos em tempo de
execução para cada CLI nativo e exibidos com destaque em `nexus <provider> --help`.

## Next action

Execute `nexus agy --help`, `nexus codex --help` ou `nexus claude --help` para conferir
a ajuda fusionada com aliases canônicos; execute `nexus usage` para quotas por janela
e `nexus web` para o Web Workspace OS local.

Correção mais recente: o Dialog de fechamento não aparece mais ao abrir a tela;
ele só é renderizado após uma ação explícita de fechar terminal ou Project Shell.
O servidor foi reiniciado com o bundle corrigido em `http://127.0.0.1:3000`.

O modo Plan foi removido dos controles de terminal e do cadastro de Agentes;
planejamento deve ocorrer no Composer/Flow ou ser enviado como instrução ao
Agente. O fluxo Safe/YOLO não encaminha mais `--plan` ao Codex. Os controles
“Assumir digitação”/“Liberar digitação” continuam ativos porque correspondem
ao lease de escrita do WebSocket do PTY compartilhado.

Build atual instalado em `/home/desenvolvedor/.local/bin/nexus` e servidor ativo
em `http://127.0.0.1:3000` (Bootstrap gerado no processo atual).

Os botões manuais de lease foram removidos do terminal. O WebSocket solicita o
lease automaticamente ao conectar; “Somente leitura” aparece apenas quando
outro acesso já possui a escrita. A barra lateral agora sempre oferece
“Visão geral”, além de Composer, Flow Runs, Uso, Sessões, Projetos e
Configurações, mesmo quando o layout salvo abre em terminais.
