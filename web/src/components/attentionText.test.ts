import { describe, expect, it } from 'vitest';
import { sanitizeAttentionText } from './attentionText';

describe('sanitizeAttentionText', () => {
  it('removes invalid replacement characters and preserves readable context', () => {
    expect(sanitizeAttentionText('�� Qual é o próximo passo?', 'fallback')).toBe(
      'Qual é o próximo passo?',
    );
  });

  it('uses the fallback when the provider only emits transport noise', () => {
    expect(sanitizeAttentionText('\u0000\uFFFD\u0007', 'Atenção necessária')).toBe(
      'Atenção necessária',
    );
  });
});
