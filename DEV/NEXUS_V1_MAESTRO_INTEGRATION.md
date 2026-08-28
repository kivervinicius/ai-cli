# Nexus V1 — Maestro Integration

Charter §5, §52-58, §103, §134-136. Maestro is a **supported dependency** of Nexus,
never the reverse. Nexus consumes a machine-readable contract; it does not copy
Maestro knowledge.

## Dependency contract

- `Orquestrador-Maestro` = 100% COMMUNITY, owns methodology/skills/process/risk/gates/verification.
- Nexus installer/doctor must: detect Maestro → read version/capabilities → verify
  compatibility → install/update when explicitly safe (§5).
- If Maestro is unavailable at runtime → **`MAESTRO_DEGRADED`**: agents, terminals,
  stop, recovery and runtime observation keep working; Nexus must not invent
  recommendations or simulate Maestro Assist (§5, §136).

## Maestro modes (project-scoped, §52)

| Mode | Behavior |
|---|---|
| OFF | Installed but no active recommendations |
| ASSIST | Default. Recommends skills/process/gates/verification; no mandatory DAG |
| ORCHESTRATE | Optional/BETA; activates Mission concepts |

## Machine-readable contract (planned for Gate 6)

Maestro is the **owner of the schemas** (§54). Versioned contract v1 must expose at
least:

```
MaestroVersion/Capabilities
AdviceRequest { project metadata, language/framework, git summary, changed paths,
                goal, risk signals, process state }   // NO secrets, §55
AdviceResponse { classification, risk, process, requiredSkills, recommendedSkills,
                 optionalSkills, gates, explanations }  // priorities: REQUIRED|RECOMMENDED|OPTIONAL + WHY
SkillRecommendation · ProcessRecommendation · RiskAssessment · VerificationRequirement
```

- Local transport preference: `stdin JSON → orquestrador-maestro → stdout JSON`,
  no mandatory daemon (§53).
- Nexus sends **safe context only**: never `.env`, credentials, OAuth, auth files or
  raw terminal transcripts (§55).
- UI: Project Maestro page + Recommendation Cards with `[Apply]`/`[Explain]`;
  compliance (Design/Plan/TDD/Review/Verification) shown only with real evidence (§57-58).

## Status

- Maestro CLI contract: **not yet implemented** (Maestro repo work, `feat/nexus-contracts-v1`).
- Nexus side: `maestro_mode` column + default `ASSIST` already in the Project model;
  recommendation UI placeholder ready; bridge + doctor detection are Gate 6.
