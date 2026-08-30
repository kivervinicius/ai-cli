export type TerminalFrame =
  | { kind: 'output'; data: string }
  | { kind: 'control'; type: string; payload: Record<string, unknown> }
  | { kind: 'protocol-error'; message: string };

const CONTROL_TYPES = new Set(['lease', 'runtime_changed', 'attention', 'title', 'status', 'error']);

/** Parse the strict control envelope without hiding legitimate provider JSON output. */
export function parseTerminalFrame(raw: unknown): TerminalFrame {
  if (typeof raw !== 'string') return { kind: 'protocol-error', message: 'terminal frame is not text' };
  let value: unknown;
  try { value = JSON.parse(raw); } catch { return { kind: 'output', data: raw }; }
  if (!value || typeof value !== 'object' || Array.isArray(value)) return { kind: 'output', data: raw };
  const payload = value as Record<string, unknown>;
  if (!('type' in payload)) return { kind: 'output', data: raw };
  if (payload.type === 'output') {
    return typeof payload.data === 'string' ? { kind: 'output', data: payload.data } : { kind: 'protocol-error', message: 'output frame data must be text' };
  }
  if (typeof payload.type !== 'string' || !CONTROL_TYPES.has(payload.type)) return { kind: 'protocol-error', message: 'unknown terminal control frame' };
  return { kind: 'control', type: payload.type, payload };
}

export function frameString(payload: Record<string, unknown>, key: string): string | undefined {
  return typeof payload[key] === 'string' ? payload[key] as string : undefined;
}
