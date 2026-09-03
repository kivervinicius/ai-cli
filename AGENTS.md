# IAPro Nexus — Contrato local para agentes

## Estado do produto

O Nexus ainda não está em produção. São permitidas refatorações estruturais
compatíveis com o objetivo do produto; não preserve implementações duplicadas
somente por compatibilidade interna sem uma necessidade explícita.

## Frontend (web)

Após qualquer alteração em `web/`, rode `make web-verify` antes de declarar
conclusão. O relatório canônico fica em `DEV/validation/FRONTEND_LATEST.md`.
Gate vermelho bloqueia entrega. UI “quebrada” com testes verdes quase sempre
significa binário antigo: `make build` + reiniciar `nexus web`.

Arrays vindos da API Go podem ser JSON `null`. Nunca use `.length`/`.map` direto
em `dependencies`, `phases`, `packages`, `generations`, etc. — normalize com
`asArray` (`web/src/lib/safeArray.ts`) ou `(value || [])`. O gate
`null-arrays` no `web-verify` rejeita acessos inseguros conhecidos.

## Interface de terminal

- Todo fluxo visual/interativo de terminal deve usar o stack Charm já adotado:
  Bubble Tea para ciclo de interface, Bubbles para componentes e Lip Gloss para
  estilos. Não criar tabelas interativas com `fmt.Printf` ou bibliotecas
  concorrentes.
- Dados de integração ou automação devem oferecer `--json`; o modo humano é a
  interface Charm padrão, sem flag visual adicional.
- Para tabelas, incluir navegação por teclado e filtro/pesquisa quando houver
  mais de uma linha ou quando o usuário precise localizar uma identidade.
