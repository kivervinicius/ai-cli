export type TerminalRole = 'CONTROL' | 'VIEW_ONLY';
export function agentTerminalWebSocketURL(protocol: string, host: string, agentId: string): string {
  return `${protocol === 'https:' ? 'wss:' : 'ws:'}//${host}/api/v1/agents/${encodeURIComponent(agentId)}/terminal`;
}
export function normalizeTerminalRole(role: unknown): TerminalRole { return role === 'CONTROL' ? 'CONTROL' : 'VIEW_ONLY'; }
