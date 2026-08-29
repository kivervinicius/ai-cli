import { describe, expect, it } from 'vitest';
import { agentTerminalWebSocketURL, normalizeTerminalRole } from './agentTerminalModel';

describe('Agent terminal model', () => {
  it('builds agent-scoped websocket URLs', () => expect(agentTerminalWebSocketURL('https:', 'nexus.local', 'agt_1')).toBe('wss://nexus.local/api/v1/agents/agt_1/terminal'));
  it('uses ws on http and encodes AgentID', () => expect(agentTerminalWebSocketURL('http:', 'localhost:8080', 'a b')).toBe('ws://localhost:8080/api/v1/agents/a%20b/terminal'));
  it('only grants control for explicit CONTROL lease', () => { expect(normalizeTerminalRole('CONTROL')).toBe('CONTROL'); expect(normalizeTerminalRole('owner')).toBe('VIEW_ONLY'); });
});
