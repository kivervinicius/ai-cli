import { describe, expect, it } from 'vitest';
import { sanitizeAttentionText } from './attentionText';

describe('Attention Notification Card sanitization & logic', () => {
  it('correctly handles raw context with UTF-8 noise', () => {
    const raw = ' Deseja executar a migração? [y/N]';
    expect(sanitizeAttentionText(raw, 'fallback')).toBe('Deseja executar a migração? [y/N]');
  });

  it('provides safe fallback when context is empty or corrupted', () => {
    expect(sanitizeAttentionText(null, 'O agente requer atenção.')).toBe('O agente requer atenção.');
    expect(sanitizeAttentionText('', 'O agente requer atenção.')).toBe('O agente requer atenção.');
  });
});
