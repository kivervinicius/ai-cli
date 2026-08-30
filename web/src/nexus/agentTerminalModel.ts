export type TerminalRole = 'CONTROL' | 'VIEW_ONLY';
export function agentTerminalWebSocketURL(protocol: string, host: string, agentId: string): string {
  return `${protocol === 'https:' ? 'wss:' : 'ws:'}//${host}/api/v1/agents/${encodeURIComponent(agentId)}/terminal`;
}
export function normalizeTerminalRole(role: unknown): TerminalRole { return role === 'CONTROL' ? 'CONTROL' : 'VIEW_ONLY'; }

export type TerminalLeaseCommand = 'lease_acquire' | 'lease_release';

export function terminalLeaseCommand(previous: TerminalRole, next: TerminalRole): TerminalLeaseCommand | null {
  if (previous === next) return null;
  return next === 'CONTROL' ? 'lease_acquire' : 'lease_release';
}

export function terminalReconnectDelay(attempt: number): number {
  const safeAttempt = Number.isFinite(attempt) ? Math.max(0, Math.floor(attempt)) : 0;
  return Math.min(3000, 250 * (2 ** safeAttempt));
}

export function normalizeInitialPrompt(prompt?: string): string {
  const value = (prompt || '').trim();
  return value ? `${value}\n` : '';
}

// A Persistent Agent may exist without a running provider process. Opening a
// terminal in that state produces a WebSocket reconnect loop, not a terminal.
export function canOpenAgentTerminal(status: unknown): boolean {
  return status === 'WORKING';
}
