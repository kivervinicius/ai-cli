import { describe, expect, it } from 'vitest';
import { agentTerminalWebSocketURL, normalizeInitialPrompt, normalizeTerminalRole, terminalLeaseCommand, terminalReconnectDelay } from './agentTerminalModel';

describe('Agent terminal model', () => {
  it('builds agent-scoped websocket URLs', () => expect(agentTerminalWebSocketURL('https:', 'nexus.local', 'agt_1')).toBe('wss://nexus.local/api/v1/agents/agt_1/terminal'));
  it('uses ws on http and encodes AgentID', () => expect(agentTerminalWebSocketURL('http:', 'localhost:8080', 'a b')).toBe('ws://localhost:8080/api/v1/agents/a%20b/terminal'));
  it('only grants control for explicit CONTROL lease', () => { expect(normalizeTerminalRole('CONTROL')).toBe('CONTROL'); expect(normalizeTerminalRole('owner')).toBe('VIEW_ONLY'); });
  it('synchronizes SessionHost lease when broker promotes or revokes control', () => {
    expect(terminalLeaseCommand('VIEW_ONLY', 'CONTROL')).toBe('lease_acquire');
    expect(terminalLeaseCommand('CONTROL', 'VIEW_ONLY')).toBe('lease_release');
    expect(terminalLeaseCommand('CONTROL', 'CONTROL')).toBeNull();
  });
  it('backs off reconnect attempts with a bounded delay', () => {
    expect(terminalReconnectDelay(0)).toBe(250);
    expect(terminalReconnectDelay(1)).toBe(500);
    expect(terminalReconnectDelay(8)).toBe(3000);
  });
  it('normalizes a direct-session kickoff prompt exactly once-ready for terminal input', () => {
    expect(normalizeInitialPrompt('  fix the failing tests  ')).toBe('fix the failing tests\n');
    expect(normalizeInitialPrompt('   ')).toBe('');
  });
});
