import { describe, expect, it } from 'vitest';
import { selectResumableComposerSession } from './composerSessionModel';

describe('selectResumableComposerSession', () => {
  it('prefers the most recently updated non-finalized session', () => {
    expect(selectResumableComposerSession([
      { id: 'done', state: 'FINALIZED', updated_at: '2026-09-01T00:00:00Z' },
      { id: 'older', state: 'EXPLORING', updated_at: '2026-09-02T00:00:00Z' },
      { id: 'active', state: 'EXPLORING', updated_at: '2026-09-03T00:00:00Z' },
    ])).toBe('active');
  });
});
