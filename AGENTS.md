# IAPro Nexus — Contrato local para agentes

## Estado do produto

O Nexus ainda não está em produção. São permitidas refatorações estruturais
compatíveis com o objetivo do produto; não preserve implementações duplicadas
somente por compatibilidade interna sem uma necessidade explícita.

## Interface de terminal

- Todo fluxo visual/interativo de terminal deve usar o stack Charm já adotado:
  Bubble Tea para ciclo de interface, Bubbles para componentes e Lip Gloss para
  estilos. Não criar tabelas interativas com `fmt.Printf` ou bibliotecas
  concorrentes.
- Dados de integração ou automação devem oferecer `--json`; o modo humano é a
  interface Charm padrão, sem flag visual adicional.
- Para tabelas, incluir navegação por teclado e filtro/pesquisa quando houver
  mais de uma linha ou quando o usuário precise localizar uma identidade.
