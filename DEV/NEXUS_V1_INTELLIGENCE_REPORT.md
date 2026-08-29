# IAPro Nexus V1 — Nexus Intelligence Layer Report

## 1. Overview
The Nexus Intelligence Layer (`internal/nexus/intelligence`) provides provider-neutral intent analysis, structured ambiguity classification, deterministic prompt compilation, and plan generation without hallucinating facts.

## 2. Architecture & Components
- **Provider Interfaces**: `IntelligenceProvider` contract implemented by `OpenAIIntelligenceProvider` (supports OpenAI, DeepSeek, Ollama, local models) and deterministic heuristic fallbacks.
- **Ambiguity Classification**: Classifies unknowns into:
  - `BLOCKING`: Material requirements that must be resolved before proceeding.
  - `IMPORTANT`: Design choices and architectural preferences.
  - `LOW_IMPACT`: Safe defaults chosen autonomously without human friction.
- **Context Assembler & Fact Resolver**: Converts resolved questions into persistent facts in the SQLite store (`structured_facts`), avoiding ephemeral prompt drift.
- **Prompt Compiler**: Compiles scoped, token-bounded system prompts combining project governance, Maestro rules, confirmed facts, and acceptance criteria.

## 3. Verification Evidence
- Unit tests: `internal/nexus/intelligence/engine_test.go` (100% pass under `-race`).
- REST APIs: `/api/v1/plans/:id/compile` verified.
