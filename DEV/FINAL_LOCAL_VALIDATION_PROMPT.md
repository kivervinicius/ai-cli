# Local validation prompt

Run from the repository root. Do not print credentials or provider tokens.

1. Verify `go version` is supported by `go.mod`, then run `go test ./...`.
2. Run `make web-verify` and confirm `DEV/validation/FRONTEND_LATEST.md` is PASS.
3. Run `make build`, start `nexus web`, and open the printed loopback URL.
4. Create a Project and open Composer. Test: vague idea, an imported prompt,
   answering an open question, confirming an assumption, selecting/rejecting a
   real Maestro skill, previewing and finalizing a PromptArtifact.
5. Confirm reopening the Project restores the Composer session and recent turns.
6. Transform a finalized prompt into a Flow Draft; verify no runtime starts
   until the explicit Run/Approve action.
7. Inspect Agent assignment, leader suggestion, preflight errors and recovery.
8. For custom OpenCode Agents, verify the configured command template receives
   the Project workspace and arguments. Authenticate only inside the intended
   runtime with `opencode auth login <provider>`.
9. Record provider availability, OS, browser, test commands and any PARTIAL
   behavior in a local report without including secrets.
