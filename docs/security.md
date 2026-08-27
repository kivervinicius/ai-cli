# Security & Isolation Model

`ai-cli` is a local control plane designed to coordinate multiple identities and accounts across AI coding CLIs while preserving strong credential isolation.

---

## 1. What Is Isolated vs What Is Not a Sandbox

### What IS Isolated:
- **Authentication Tokens & Keyrings**: Each profile has its own dedicated credentials store (`CODEX_HOME` for Codex, isolated D-Bus Secret Service keyring for AGY, isolated `CLAUDE_CONFIG_DIR` for Claude Code, isolated `XDG_DATA_HOME` for OpenCode, isolated `GEMINI_CLI_HOME` for Gemini CLI).
- **OAuth Session Files**: Stored exclusively within the profile directory tree.
- **Provider Settings & Overrides**: Configured per profile without cross-contamination.
- **Environment Variables**: Overridden on process invocation (`HOME`, `CODEX_HOME`, `XDG_*`, `SSH_AUTH_SOCK`).

### What Is NOT a Sandbox:
- `ai-cli` runs processes under the **same Linux UID/GID** as the calling user.
- It is **not a virtual machine or container jail**.
- If an agent is authorized to edit files in your working directory, it can write to files within the permissions of your Linux user account.

---

## 2. Isolation Presets

You can configure the global isolation level in your config:

```json
{
  "isolation_preset": "developer"
}
```

### Presets Comparison:

| Preset | Shared Features | Blocked / Isolated | Recommendation |
|---|---|---|---|
| `strict` | Git user identity (read-only), `SSH_AUTH_SOCK` | `~/.ssh/*`, `~/.git-credentials`, `~/.gnupg`, `~/.kube`, `~/.docker`, `~/.npmrc` | Hardened environments, untrusted external scripts |
| `developer` *(default)* | `.gitconfig`, `SSH_AUTH_SOCK`, `.npmrc` (read-only), developer tool configurations | `~/.ssh` private keys, `~/.git-credentials`, `~/.kube`, `~/.docker`, `~/.gnupg` | **Recommended default** for active development |
| `compat` | Host dotfiles (`.gitconfig`, `.ssh`, `.npmrc`, `.bashrc`) | Isolated profile keyrings & OAuth tokens | Legacy backward compatibility |

---

## 3. SSH Key Security

`ai-cli` avoids linking or copying raw private key files from `~/.ssh` into profile directories by default. Instead:
- It forwards the active `SSH_AUTH_SOCK` (SSH Agent).
- The AI agent can perform git operations via the agent without ever reading your private key bytes from disk.

---

## 4. Secret Redaction

The control plane implements an automatic redaction filter that masks sensitive tokens before logging, diagnostic exports, or issue reports:
- Bearer tokens & Authorization headers
- JSON Web Tokens (JWT)
- OpenAI API keys (`sk-...`)
- Anthropic API keys (`sk-ant-...`)
- Google OAuth tokens (`ya29...`)
- RSA/EC/OpenSSH private key blocks
- Plain-text passwords and cookies

Run `ai security` to inspect the exact permission boundaries and file links for each configured profile.
