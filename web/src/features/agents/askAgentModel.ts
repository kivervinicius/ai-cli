export interface AskAgentAction {
  label: 'Ask Agent' | 'Start & Ask';
  startIfNeeded: boolean;
}

const active = new Set(['WORKING', 'WAITING', 'APPROVAL', 'HANDOFF', 'STARTING', 'RECOVERING']);

export function askActionForStatus(status: string): AskAgentAction {
  return active.has(String(status || '').toUpperCase())
    ? { label: 'Ask Agent', startIfNeeded: false }
    : { label: 'Start & Ask', startIfNeeded: true };
}
