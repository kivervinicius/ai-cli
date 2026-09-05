# Guia de Contribuição — IAPro Nexus

Obrigado por contribuir com o **IAPro Nexus**! Este documento orienta desenvolvedores humanos e mantenedores sobre as convenções arquiteturais, padrões de código, fluxos de trabalho e critérios de aceitação do projeto.

---

## 1. Princípios de Engenharia

No IAPro Nexus, priorizamos **qualidade, manutenibilidade, previsibilidade e acessibilidade** a longo prazo:

```text
Corretude → Segurança → Acessibilidade (WCAG 2.2 AA) → Manutenibilidade → Legibilidade → Testabilidade → Consistência → Performance → Estética
```

Antes de iniciar qualquer implementação, consulte o documento oficial:
* [Web Engineering Standards](docs/engineering/WEB_ENGINEERING_STANDARDS.md)

---

## 2. Estrutura do Projeto

```text
.
├── cmd/nexus/              # Entry point do executável Go
├── internal/               # Lógica de domínio do backend Go
│   ├── app/                # Comandos CLI (Charm / Bubble Tea)
│   ├── control/            # Servidor web, PTY drivers e protocolo
│   └── nexus/              # Runtimes, profiles e orquestração
├── web/                    # Aplicação Web (React 19 + TypeScript)
│   ├── src/
│   │   ├── app/            # Shell OS, bootstrapping e layout principal
│   │   ├── components/     # Componentes compartilhados
│   │   ├── design-system/  # Primitivas de UI e temas
│   │   ├── features/       # Módulos de domínio (agents, work, settings, etc.)
│   │   ├── i18n/           # Catálogo de internacionalização
│   │   ├── nexus/          # API client e integração com PTY
│   │   └── workspace/      # Gerenciador de janelas e renderizador
│   └── scripts/            # Scripts de build e gates de validação
└── docs/                   # Documentação técnica e arquitetural
```

---

## 3. Padrões Frontend

### 3.1 Hierarquia de Componentes
Sempre pesquise antes de criar um novo componente:
1. **Componente Existente**: Verifique em `web/src/design-system/primitives/` ou `web/src/components/`.
2. **Primitiva Radix**: Use `@radix-ui/react-*` para modais, tooltips, dialogs e dropdowns.
3. **Composição**: Combine primitivas existentes antes de criar novos controles.

### 3.2 HTML Semântico & Acessibilidade
* Use controles nativos: `<button type="button">`, `<a>`, `<form>`, `<label>`, `<select>`.
* **Proibido** `<div onClick>` ou `<span onClick>` para elementos interativos.
* Garanta operação completa por teclado (`Tab`, `Shift+Tab`, `Enter`, `Space`, `Escape`).
* Mantenha contraste de cores de acordo com WCAG 2.2 AA em todos os temas.

### 3.3 Estilos & CSS Custom Properties
* Use variáveis semânticas (`var(--nx-surface)`, `var(--nx-text)`, `var(--nx-accent)`, etc.).
* **Proibido** CSS inline estático (`style={{ ... }}`). Estilos inline são reservados estritamente para propriedades dinâmicas de runtime (dimensões de janelas/posicionamento flutuante).
* **Proibido** o uso de `!important`.
* Modularize estilos com CSS/SCSS e limite o aninhamento a no máximo 3 níveis.

### 3.4 Internacionalização (i18n)
* Nenhum texto visível ao usuário deve ser fixado diretamente no JSX.
* Todos os textos devem passar pelo `react-i18next` (`t('chave.semantica')`).
* Atualize `web/src/i18n/resources.ts` com as novas chaves.

### 3.5 TypeScript & Resiliência
* `strict: true` em todo o código.
* Evite `any`, `@ts-ignore` e casts forçados.
* Trate arrays vindos da API Go que podem ser `null` utilizando `asArray()` (`web/src/lib/safeArray.ts`) ou `(val || [])`.

---

## 4. Comandos de Desenvolvimento e Qualidade

| Comando | Descrição |
| :--- | :--- |
| `make quality` | Executa todos os gates locais rápidos (formatação, linters, types e testes) |
| `make web-verify` | Executa a suíte de verificação frontend com relatório detalhado |
| `make build` | Compila os assets frontend e o binário Go |
| `npm --prefix web run quality` | Validação completa do frontend (Prettier + ESLint + Stylelint + TypeScript + Vitest) |
| `npm --prefix web test` | Executa todos os testes unitários do frontend |

---

## 5. Checklist de Code Review

Antes de submeter um Pull Request, certifique-se de que todos os itens abaixo foram atendidos:

- [ ] Componente ou primitiva existente foi reutilizado sempre que possível.
- [ ] Nenhuma primitiva de UI foi duplicada desnecessariamente.
- [ ] HTML é estritamente semântico (sem divs clicáveis como botões).
- [ ] Todas as interações funcionam por teclado.
- [ ] Textos da interface utilizam `i18n`.
- [ ] Nenhum CSS inline estático ou `!important` foi introduzido.
- [ ] Tokens de design e CSS variables foram respeitados.
- [ ] Tamanho e responsabilidade dos arquivos foram revisados (1 componente por arquivo).
- [ ] Testes unitários/componentes foram adicionados ou atualizados.
- [ ] `make quality` executa com 100% de sucesso.

---

## 6. Padrão de Commits

Utilizamos **Conventional Commits**:

```text
type(scope): description

Exemplos:
feat(ui): add keyboard navigation for workspace tabs
fix(api): ensure null-safe fallback for agent dependencies
refactor(settings): streamline theme preset selection
```

* **Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`
* **Scopes**: `terminal`, `chat`, `flow`, `agents`, `providers`, `quota`, `workspace`, `settings`, `web`, `go`, `infra`, `security`, `ui`, `deps`, `release`
