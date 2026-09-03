export type TerminalRole = 'CONTROL' | 'VIEW_ONLY';

/** Hard stop after this many failed reconnects so the UI is not stuck on "connecting". */
export const TERMINAL_MAX_RECONNECT_ATTEMPTS = 5;

export function agentTerminalWebSocketURL(
  protocol: string,
  host: string,
  agentId: string,
  runtimeId?: string
): string {
  const base = `${protocol === 'https:' ? 'wss:' : 'ws:'}//${host}/api/v1/agents/${encodeURIComponent(agentId)}/terminal`;
  const trimmed = (runtimeId || '').trim();
  if (!trimmed) return base;
  return `${base}?runtime_id=${encodeURIComponent(trimmed)}`;
}

export function normalizeTerminalRole(role: unknown): TerminalRole {
  return role === 'CONTROL' ? 'CONTROL' : 'VIEW_ONLY';
}

export function terminalReconnectDelay(attempt: number): number {
  const safeAttempt = Number.isFinite(attempt) ? Math.max(0, Math.floor(attempt)) : 0;
  return Math.min(3000, 250 * (2 ** safeAttempt));
}

export function normalizeInitialPrompt(prompt?: string): string {
  const value = (prompt || '').trim();
  return value ? `${value}\n` : '';
}

/** Fatal attach failures should stop the reconnect loop and surface a clear message. */
export function isFatalTerminalAttachError(message: string): boolean {
  const lower = (message || '').toLowerCase();
  return (
    lower.includes('no active runtime') ||
    lower.includes('runtime not found') ||
    lower.includes('runtime host is not running') ||
    lower.includes('no longer responding') ||
    lower.includes('authentication required') ||
    lower.includes('invalid origin')
  );
}

export function terminalAttachFailureMessage(detail?: string): string {
  const trimmed = (detail || '').trim();
  if (trimmed && isFatalTerminalAttachError(trimmed)) {
    return `${trimmed} Abra Agentes → Recover/Start e tente de novo.`;
  }
  if (trimmed) return trimmed;
  return 'Não foi possível anexar ao runtime do Agente. Use Recover/Start em Agentes e reabra o terminal.';
}
