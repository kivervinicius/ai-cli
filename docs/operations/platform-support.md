# Suporte de plataformas

Build, runtime e terminal nativo são gates diferentes. Cross-compilation não
prova execução. O estado abaixo é uma política
documental: uma linha só deve ser marcada como verificada quando houver teste
ou CI nativo correspondente.

| Área | Linux | Windows | macOS |
| --- | --- | --- | --- |
| CLI/Core build | evidência local/CI | runner nativo necessário | runner nativo necessário |
| Web frontend | compartilhado | compartilhado | compartilhado |
| PTY/runtime | PTY/UDS | ConPTY/Named Pipe | PTY/UDS |
| Desktop Wails | build observado | build nativo necessário | build nativo necessário |
| Installer/update | testes de contrato | smoke nativo necessário | smoke nativo necessário |

Não trate cross-compilation como prova de runtime. Para diagnóstico, consulte
[troubleshooting](troubleshooting.md). Use `nexus doctor --json` para o estado
do ambiente. Não diga “suportado” apenas porque `GOOS=windows go build` passa.
