# IAPro Nexus — beta Linux

Este pacote é um beta para testes locais em Linux (amd64 ou arm64).

## Instalação

```bash
tar -xzf nexus-linux-*-v*.tar.gz
cd nexus-linux-*
./nexus version --json
./nexus doctor --json
./nexus web --port 3000
```

Abra o endereço de bootstrap exibido pelo comando `nexus web`.

## Escopo conhecido

- Suporte de teste: Linux, uso local/loopback.
- Windows e macOS ainda aguardam execução nativa dos testes de runtime.
- Não use este beta para dados sensíveis ou exposição pública/Internet.
- O terminal remoto (`--remote`) não oferece TLS; use apenas rede privada confiável.

## Fluxo sugerido para o testador

1. Configurações → Diagnóstico do sistema e exportação do relatório.
2. Projetos → abrir um workspace e preparar contexto.
3. Composer → criar uma instrução e executar um Flow.
4. Agentes → iniciar um agente, enviar uma mensagem e recuperar o terminal.
5. Uso → conferir quotas, janelas e estados de conta.
6. Relatar qualquer erro com `nexus doctor --bundle` e o passo de reprodução.
