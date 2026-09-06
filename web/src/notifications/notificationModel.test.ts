import { describe, expect, it } from 'vitest';
import { notificationTitle } from './notificationModel';

describe('notificationTitle', () => {
  it('uses the provider/runtime dynamic title when available', () => {
    expect(notificationTitle('proxy-nginx', 'QUESTION', 'Deploy requires approval')).toBe(
      '[Nexus | proxy-nginx] Deploy requires approval',
    );
  });

  it('keeps semantic fallback titles for attention reasons', () => {
    expect(notificationTitle(undefined, 'APPROVAL')).toContain('Aprovação Necessária');
  });
});
