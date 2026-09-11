import { describe, expect, it } from 'vitest';
import type { ProviderAccount } from '../../types';
import {
  accountQuotaState,
  capabilityLabelKey,
  displayLabelKey,
  formatResetDescription,
  groupHasCapacity,
  parseResetDescription,
  relativeAge,
  summarizeAccounts,
  windowKindLabelKey,
} from './usageModel';

const base = (partial: Partial<ProviderAccount>): ProviderAccount => ({
  id: 'agy:test',
  provider: 'agy',
  profile: 'test',
  display_name: 'agy test',
  authenticated: true,
  is_default: false,
  available: true,
  quota_remaining: 0.5,
  quota_total: 1,
  rate_limited: false,
  health: 'healthy',
  last_checked: new Date().toISOString(),
  ...partial,
});

describe('usageModel', () => {
  it('does not mark account exhausted when another group still has capacity', () => {
    const account = base({
      available: true,
      avail_reasons: { exhausted_windows: ['gemini:weekly'] },
      quota_view: {
        status: 'LIVE',
        model_groups: [
          {
            name: 'Gemini',
            windows: [{ kind: 'weekly', remaining: 0 }],
          },
          {
            name: 'Claude',
            windows: [{ kind: '5h', remaining: 100 }],
          },
        ],
      },
    });
    expect(accountQuotaState(account)).toBe('confirmed');
    expect(groupHasCapacity(account.quota_view?.model_groups?.[1])).toBe(true);
  });

  it('summarizes fleet counts', () => {
    const summary = summarizeAccounts([
      base({
        available: true,
        quota_view: {
          status: 'LIVE',
          model_groups: [{ windows: [{ kind: '5h', remaining: 40 }] }],
        },
      }),
      base({
        id: 'b',
        available: false,
        avail_reasons: { exhausted_windows: ['all'] },
        quota_view: { status: 'LIVE', model_groups: [{ windows: [{ kind: '5h', remaining: 0 }] }] },
      }),
      base({
        id: 'c',
        available: false,
        avail_reasons: { unknown_quota: true },
        quota_view: { status: 'UNKNOWN' },
      }),
    ]);
    expect(summary.usable).toBe(1);
    expect(summary.exhausted).toBe(1);
    expect(summary.unknown).toBe(1);
  });

  it('returns relative age without mixing language into the status code', () => {
    const now = Date.parse('2026-09-10T20:00:00Z');
    expect(relativeAge('2026-09-10T19:59:52Z', now)).toEqual({ unit: 'seconds', count: 8 });
  });

  it('maps known English API labels and window kinds to i18n keys', () => {
    expect(displayLabelKey('OpenCode Provider')).toBe('usage.plan.opencode');
    expect(displayLabelKey('Gemini Models', 'gemini')).toBe('usage.group.gemini');
    expect(displayLabelKey(undefined, 'claude_gpt')).toBe('usage.group.claudeGpt');
    expect(displayLabelKey('ChatGPT Plus')).toBeNull();
    expect(windowKindLabelKey('weekly')).toBe('usage.window.weekly');
    expect(windowKindLabelKey('5h')).toBe('usage.window.fiveHour');
    expect(windowKindLabelKey('unknown')).toBe('usage.window.unknown');
    expect(capabilityLabelKey('cancel_turn')).toBe('usage.capability.cancel_turn');
  });

  it('parses English Codex reset strings for localized formatting', () => {
    expect(parseResetDescription('resets 03:48')).toEqual({ kind: 'sameDay', time: '03:48' });
    expect(parseResetDescription('resets 05:35 on 15 Sep')).toEqual({
      kind: 'crossDay',
      time: '05:35',
      day: 15,
      monthIndex: 8,
    });
    expect(
      formatResetDescription('resets 05:35 on 15 Sep', 'pt-BR', (key, options) =>
        key === 'usage.resetOn' ? `renova às ${options?.time} em ${options?.date}` : key,
      ),
    ).toMatch(/^renova às 05:35 em /);
  });
});
