# Final Provider Matrix

> Historical snapshot. For the current closure decision use the canonical
> `ValidationEvidenceStream` projections and `./nexus doctor --json` recorded
> at HEAD `a58cca4d73bdfd55678649b30c0ca64f73b7d410`. Binary installation is
> not authenticated Mission evidence; current release verdict is `NO-GO`.

| Provider | Binary probe | Authenticated real E2E |
|---|---|---|
| Codex | INSTALLED (`0.151.0`); direct host marker PASS | Nexus isolated runtime UNVERIFIED (`model: loading`) |
| Claude | INSTALLED (`2.1.250`) | NOT_AUTHENTICATED/NOT_TESTED |
| Gemini | INSTALLED (`0.28.2`) | NOT_AUTHENTICATED/NOT_TESTED |
| AGY | INSTALLED (`1.1.22`); registered/authenticated status | Nexus runtime UNVERIFIED (provider reported not signed in) |
| OpenCode | INSTALLED (`1.18.25`); pending auth | UNVERIFIED |
| Cursor | INSTALLED; non-interactive version probe hung | UNSUPPORTED FOR THIS RUN |

No provider is represented as functional based on binary presence alone.
The direct Codex host marker only proves host provider availability; it is not
proof of Nexus profile isolation, Mission completion, failover or evidence
ledger emission.
