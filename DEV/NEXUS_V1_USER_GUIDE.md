# IAPro Nexus V1 — User & Operator Guide

## 1. Quick Start
To launch the Nexus Workspace OS:
```bash
nexus
# or
ai
```
This starts the local web control center and opens the authenticated Workspace OS in your default browser.

## 2. Command Line Automation
- **List & create work plans**:
  ```bash
  nexus plan list
  nexus plan create "Implement SQLite transactions and unit tests"
  nexus plan show <plan-id>
  ```
- **Inspect agents**:
  ```bash
  nexus agents list
  ```
- **Inspect & register projects**:
  ```bash
  nexus projects list
  nexus projects add /path/to/project "Project Name"
  ```
- **Diagnostics & updates**:
  ```bash
  nexus doctor
  nexus update
  ```

## 3. Visual Plan Builder
1. Navigate to the **Work** tab in Workspace OS.
2. Switch to **Planejado (Planned)** mode.
3. Enter your high-level goal in the **Nexus Intent Decomposer** bar or build phases and packages manually.
4. Click **Compilar Prompt** on any WorkPackage to preview the exact scoped prompt with verified Maestro rules and constraints.
5. Click **Executar Plano** to trigger autonomous execution with live step-by-step verification evidence.
