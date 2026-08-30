import { describe, expect, it } from 'vitest';
import { parseTerminalFrame } from './terminalProtocol';

describe('terminal protocol', () => {
  it('renders provider JSON as output when it is not a control envelope', () => {
    expect(parseTerminalFrame('{"hello":"world"}')).toEqual({ kind: 'output', data: '{"hello":"world"}' });
  });
  it('isolates valid control frames from terminal output', () => {
    expect(parseTerminalFrame('{"type":"lease","role":"CONTROL"}')).toMatchObject({ kind: 'control', type: 'lease' });
  });
  it('reports malformed/unknown control frames without leaking JSON', () => {
    expect(parseTerminalFrame('{"type":"secret_control"}')).toEqual({ kind: 'protocol-error', message: 'unknown terminal control frame' });
    expect(parseTerminalFrame('{"type":"output"}')).toEqual({ kind: 'protocol-error', message: 'output frame data must be text' });
  });
});
