# Handoff

## Current state

Nexus was rebuilt from the current source and installed at `/home/desenvolvedor/.local/bin/nexus`; the `ai` alias points to the same binary. Both report version `0.5.0-beta.10`.

## Verification

Fresh `zsh` verification and focused Nexus Go tests passed. No `nexus web` process is currently running.

O erro de runtime `Cannot read properties of null (reading 'forEach')` foi
corrigido no Flow: payloads com `phases` ou `packages` nulos são tratados como
vazios, e o bundle embutido foi recompilado. O teste de regressão e `yarn build`
passaram; o typecheck também passou na validação final.

A proteção foi centralizada em `web/src/nexus/workPlan.ts`; os endpoints de
planos normalizam as respostas na entrada, evitando que novas telas repitam a
mesma vulnerabilidade.

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
