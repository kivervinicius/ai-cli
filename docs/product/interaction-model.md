# Product interaction model

Nexus has one Core with three surfaces. Users should think in three modes even
though the internal model remains richer:

| Mode | User intent | Core path |
| --- | --- | --- |
| Direct | Work directly with an agent | Project → Session → Provider → Runtime → Workspace/Terminal |
| Automated | Give Nexus an objective | Composer → WorkPlan → Flow → Run |
| Orchestrated | Execute toward a Definition of Done | Mission → Run/tasks → verification, optionally Maestro |

This is a product vocabulary layer, not a request to expose every internal
object in the UI. The current Web/Desktop layout remains unchanged in this
refactor. CLI and API preserve their existing names for backward compatibility.

Maestro contributes method, skills, risk, review, and quality-gate policy. It
does not become the Nexus runtime or persistence layer.
