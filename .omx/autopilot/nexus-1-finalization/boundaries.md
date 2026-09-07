# Boundaries — Nexus 1.0 finalization

## NEVER

- Não executar `git reset --hard`, `git checkout --`, remoções amplas ou push automático.
- Não expor tokens, credenciais, `.env`, sessões ou transcrições em logs/artifacts.
- Não declarar PASS por documentação, mock isolado, cross-build ou SHA diferente.
- Não criar segundo scheduler, DAG, WorkPlan, runtime, memória ou provider abstraction.

## DANGER

- Provider authentication/quota, worktrees, process killing, installer e native CI.
- Alterações em auth, filesystem policy, cookies, CSRF, Origin, WebSocket e redaction.

## ROLLBACK

- Preservar branch/HEAD/status atual como baseline; cada lane deve usar ownership disjunto.
- Se uma mudança falhar, reverter somente a alteração própria via patch revisável.

## VERIFY

- Testes focados antes dos gates amplos; depois `make quality`, `make security` e `git diff --check`.
- Release só avança com evidência apontando ao mesmo SHA imutável.
