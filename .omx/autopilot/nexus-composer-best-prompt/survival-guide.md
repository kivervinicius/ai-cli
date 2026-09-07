# Composer survival guide

The Composer is persisted under the existing Composer session/store contracts.
Prompt revisions and artifacts must remain immutable once finalized; mutations
use optimistic revision checks. A Flow handoff is a draft-only adapter and must
not start a runtime. On restart, resume the persisted session by ID and reload
the canonical artifact/revision rather than reconstructing it from chat text.

If Maestro is unavailable, expose `MAESTRO_DEGRADED` with concrete availability
evidence and omit synthetic skills/recommendations. Never install skills from the
Composer path.
