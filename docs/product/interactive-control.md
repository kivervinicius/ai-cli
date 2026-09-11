# Interactive control

Interactive TTY provider launches default to supervised mode. `--direct`,
`--supervised`, `--print`, and `NEXUS_LAUNCH_MODE` are resolved centrally.

`/nexus` and `:nexus` are the canonical control prefixes. `//nexus` and
`::nexus` escape one delimiter and forward the provider command literally.
The old `ai` aliases are not part of the supervised control contract.

The byte router is covered by focused tests. Provider PTY E2E and contextual
completion remain pending evidence.
