import { describe, expect, it } from 'vitest';
import {
  TERMINAL_MAX_RECONNECT_ATTEMPTS,
  agentTerminalWebSocketURL,
  isFatalTerminalAttachError,
  normalizeInitialPrompt,
  normalizeTerminalRole,
  isRecoverAlreadyAlive,
  shouldFallbackRecoverToStart,
  terminalAttachFailureMessage,
  terminalReconnectDelay,
} from './agentTerminalModel';

describe('Agent terminal model', () => {
  it('builds agent-scoped websocket URLs', () =>
    expect(agentTerminalWebSocketURL('https:', 'nexus.local', 'agt_1')).toBe(
      'wss://nexus.local/api/v1/agents/agt_1/terminal'
    ));
  it('appends runtime_id when focusing a specific radar runtime', () =>
    expect(agentTerminalWebSocketURL('http:', 'localhost:3000', 'agt_1', 'rt_9')).toBe(
      'ws://localhost:3000/api/v1/agents/agt_1/terminal?runtime_id=rt_9'
    ));
  it('supports agent-scoped reattachment after a runtime generation changes', () =>
    expect(agentTerminalWebSocketURL('http:', 'localhost:3000', 'agt_1')).toBe(
      'ws://localhost:3000/api/v1/agents/agt_1/terminal'
    ));
  it('uses ws on http and encodes AgentID', () =>
    expect(agentTerminalWebSocketURL('http:', 'localhost:8080', 'a b')).toBe(
      'ws://localhost:8080/api/v1/agents/a%20b/terminal'
    ));
  it('only grants control for explicit CONTROL lease', () => {
    expect(normalizeTerminalRole('CONTROL')).toBe('CONTROL');
    expect(normalizeTerminalRole('owner')).toBe('VIEW_ONLY');
  });
  it('backs off reconnect attempts with a bounded delay', () => {
    expect(terminalReconnectDelay(0)).toBe(250);
    expect(terminalReconnectDelay(1)).toBe(500);
    expect(terminalReconnectDelay(8)).toBe(3000);
    expect(TERMINAL_MAX_RECONNECT_ATTEMPTS).toBeGreaterThan(0);
  });
  it('normalizes a direct-session kickoff prompt exactly once-ready for terminal input', () => {
    expect(normalizeInitialPrompt('  fix the failing tests  ')).toBe('fix the failing tests\n');
    expect(normalizeInitialPrompt('   ')).toBe('');
  });
  it('detects fatal attach errors that must stop reconnect loops', () => {
    expect(isFatalTerminalAttachError('agent has no active runtime: not found')).toBe(true);
    expect(isFatalTerminalAttachError('Runtime host is not running (dial timeout)')).toBe(true);
    expect(isFatalTerminalAttachError('temporary blip')).toBe(false);
    expect(terminalAttachFailureMessage('runtime not found: rt_x')).toContain('Recover/Start');
  });
  it('falls back recover→start for recoverable host failures', () => {
    expect(shouldFallbackRecoverToStart('agent is STOPPED (use StartAgent to restart)')).toBe(true);
    expect(shouldFallbackRecoverToStart('REQUIRED_RESOURCE_SELECTION')).toBe(false);
  });

  it('treats already-alive recover as soft success, not stop+start', () => {
    expect(isRecoverAlreadyAlive('agent runtime is already alive (no recovery needed)')).toBe(true);
    expect(shouldFallbackRecoverToStart('agent runtime is already alive (no recovery needed)')).toBe(false);
  });
});
