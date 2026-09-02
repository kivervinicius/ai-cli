# Handoff

## Current state

Nexus was rebuilt from the current source and installed at `/home/desenvolvedor/.local/bin/nexus`; the `ai` alias points to the same binary. Both report version `0.5.0-beta.9`.

## Verification

Fresh `zsh` verification and focused Nexus Go tests passed. No `nexus web` process is currently running.

`nexus usage` foi recompilado para mostrar a capacidade completa por modelo e
os resets de cada janela (5h/semanal), sem reduzir a saída ao bottleneck.

## Next action

Run `nexus usage` para conferir as janelas por modelo; depois `nexus web` para
iniciar o local Web Workspace OS e abrir a URL Bootstrap de uso único.
