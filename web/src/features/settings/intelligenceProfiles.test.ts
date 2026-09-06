import { describe, expect, it } from 'vitest';
import type { ProviderAccount } from '../../types';
import {
  intelligenceCLIProfiles,
  intelligenceModelChoices,
  isIntelligenceCLICapable,
} from './intelligenceProfiles';

function account(
  partial: Partial<ProviderAccount> & Pick<ProviderAccount, 'provider' | 'profile'>,
): ProviderAccount {
  return {
    id: `${partial.provider}:${partial.profile}`,
    display_name: `${partial.provider} (${partial.profile})`,
    authenticated: true,
    is_default: false,
    available: false,
    quota_remaining: 0.42,
    quota_total: 1,
    rate_limited: false,
    health: 'unknown',
    last_checked: '',
    capabilities: { headless: 'SUPPORTED', submit_prompt: 'SUPPORTED' },
    ...partial,
  };
}

describe('intelligenceCLIProfiles', () => {
  it('lists every discovered profile, including quota-unavailable ones', () => {
    const agy = account({ provider: 'agy', profile: 'kiveromegasistemas', available: false });
    const claude = account({
      provider: 'claude',
      profile: 'work',
      available: true,
      capabilities: { headless: 'SUPPORTED', submit_prompt: 'SUPPORTED' },
    });
    const codex = account({
      provider: 'codex',
      profile: 'default',
      available: true,
      capabilities: { headless: 'UNSUPPORTED', submit_prompt: 'UNSUPPORTED' },
    });
    const profiles = intelligenceCLIProfiles([
      agy,
      claude,
      codex,
      account({ provider: 'shell', profile: 'local' }),
    ]);
    expect(profiles.map((item) => `${item.provider}:${item.profile}`)).toEqual([
      'agy:kiveromegasistemas',
      'claude:work',
      'codex:default',
    ]);
    expect(isIntelligenceCLICapable(codex)).toBe(false);
  });

  it('keeps the currently configured profile visible when missing from discovery', () => {
    const profiles = intelligenceCLIProfiles([], {
      provider: 'agy',
      profile: 'kiveromegasistemas',
    });
    expect(profiles.map((item) => `${item.provider}:${item.profile}`)).toEqual([
      'agy:kiveromegasistemas',
    ]);
  });

  it('extracts model families from quota groups so the user can switch models', () => {
    const agy = account({
      provider: 'agy',
      profile: 'kiveromegasistemas',
      quota_view: {
        status: 'CACHED',
        model_groups: [
          { name: 'gemini', windows: [{ kind: '5h', remaining: 80 }] },
          { name: 'claude_gpt', windows: [{ kind: 'claude_5h', remaining: 12 }] },
        ],
      },
    });
    expect(intelligenceModelChoices(agy)).toEqual([
      { id: 'gemini', label: 'gemini', remaining: 80 },
      { id: 'claude_gpt', label: 'claude_gpt', remaining: 12 },
    ]);
  });
});
