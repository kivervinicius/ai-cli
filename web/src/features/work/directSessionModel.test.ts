import { describe, expect, it } from 'vitest';
import { buildDirectAgentName, directAccountTitle, directQuotaPercent, eligibleDirectResources } from './directSessionModel';

describe('direct session model', () => {
  it('builds a short useful Agent name from the prompt', () => {
    expect(buildDirectAgentName('Corrigir autenticação Microsoft no backend', 'codex')).toBe('Codex · Corrigir autenticação Microsoft');
  });

  it('falls back to provider session when prompt is empty', () => {
    expect(buildDirectAgentName('', 'claude')).toBe('Claude · Direct Session');
  });

  it('only offers authenticated and available resources and prefers the requested provider', () => {
    const accounts = [
      { id: '1', provider: 'codex', profile: 'work', authenticated: true, available: true },
      { id: '2', provider: 'claude', profile: 'personal', authenticated: true, available: true },
      { id: '3', provider: 'codex', profile: 'blocked', authenticated: true, available: false },
      { id: '4', provider: 'gemini', profile: 'anon', authenticated: false, available: true },
    ];
    expect(eligibleDirectResources(accounts, 'claude').map((item) => item.id)).toEqual(['2', '1']);
  });

  it('prefers the account display name and skips duplicate provider chips', () => {
    expect(directAccountTitle({ display_name: 'codex (kiver.omegasistemas)', provider: 'codex', profile: 'kiver.omegasistemas' })).toBe('codex (kiver.omegasistemas)');
    expect(directAccountTitle({ provider: 'claude', profile: 'default' })).toBe('claude');
  });

  it('formats known quota as a percent and keeps unknown as null', () => {
    expect(directQuotaPercent({ quota_remaining: 0.7, quota_view: { status: 'OK' } })).toBe(70);
    expect(directQuotaPercent({ quota_remaining: 40, quota_view: { status: 'OK' } })).toBe(40);
    expect(directQuotaPercent({ quota_remaining: 0.7, avail_reasons: { unknown_quota: true } })).toBeNull();
  });
});
