# Interactive control

Interactive TTY provider launches default to supervised mode. `--direct`,
`--supervised`, `--print`, and `NEXUS_LAUNCH_MODE` are resolved centrally.

`:nexus` and `:ai` are control prefixes; `::nexus` and `::ai` escape one
delimiter and forward the provider command. Legacy `/nexus`, `/ai`, `//nexus`,
and `//ai` remain supported.

The byte router is covered by focused tests. Provider PTY E2E and contextual
completion remain pending evidence.
