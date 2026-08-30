import { describe, expect, it } from 'vitest';
import { normalizeOpenRuntimeIds, visibleTerminalRuntimes } from './terminalViewModel';
import type { RuntimeSession } from '../types';

const runtime = (runtime_id: string): RuntimeSession => ({ runtime_id } as RuntimeSession);

describe('terminalViewModel', () => {
  it('drops stale and duplicate views while preserving the explicit order', () => {
    expect(normalizeOpenRuntimeIds(['a', 'a', 'gone'], [runtime('a'), runtime('b')])).toEqual(['a']);
  });

  it('opens the selected session without duplicating an existing view', () => {
    expect(normalizeOpenRuntimeIds(['a'], [runtime('a'), runtime('b')], 'b')).toEqual(['a', 'b']);
  });

  it('never renders every runtime as an implicit split view', () => {
    const runtimes = ['a', 'b', 'c'].map(runtime);
    expect(visibleTerminalRuntimes(runtimes, ['b'], 'b', 'single').map((item) => item.runtime_id)).toEqual(['b']);
    expect(visibleTerminalRuntimes(runtimes, ['b'], 'b', 'split-h').map((item) => item.runtime_id)).toEqual(['b']);
  });
});
