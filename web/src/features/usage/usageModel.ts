import type { ProviderAccount } from '../../types';
import {
  bestGroupRemainingFromQuotaView,
  quotaTruthState,
  type QuotaTruthState,
} from '../work/directSessionModel';
import { asArray } from '../../lib/safeArray';

export type UsageSummary = {
  total: number;
  usable: number;
  exhausted: number;
  unknown: number;
  blocked: number;
};

export function accountQuotaState(account: ProviderAccount): QuotaTruthState {
  const best = bestGroupRemainingFromQuotaView(account.quota_view);
  const base = quotaTruthState(account);
  // Independent pools: if any group still has capacity and account is available,
  // do not scream "exhausted" at account level.
  if (base === 'exhausted' && account.available && best != null && best > 0) {
    const status = String(account.quota_view?.status || '').toUpperCase();
    if (status === 'LIVE') return 'confirmed';
    if (status === 'CACHED' || status === 'ESTIMATED') return 'stale';
    return 'confirmed';
  }
  return base;
}

export function groupHasCapacity(group?: {
  windows?: Array<{ kind?: string; remaining?: number }>;
}): boolean {
  return asArray<{ kind?: string; remaining?: number }>(group?.windows).some(
    (window) => window && window.kind !== 'unknown' && Number(window.remaining) > 0,
  );
}

export function summarizeAccounts(accounts: ProviderAccount[]): UsageSummary {
  const summary: UsageSummary = {
    total: accounts.length,
    usable: 0,
    exhausted: 0,
    unknown: 0,
    blocked: 0,
  };
  for (const account of accounts) {
    const state = accountQuotaState(account);
    if (state === 'blocked' || state === 'unauthenticated') summary.blocked += 1;
    else if (state === 'exhausted') summary.exhausted += 1;
    else if (state === 'unknown') summary.unknown += 1;
    else if (account.available) summary.usable += 1;
    else summary.unknown += 1;
  }
  return summary;
}

export type RelativeAge = {
  unit: 'seconds' | 'minutes' | 'hours';
  count: number;
};

export function relativeAge(fetchedAt: string | undefined, now = Date.now()): RelativeAge | null {
  if (!fetchedAt) return null;
  const ts = Date.parse(fetchedAt);
  if (!Number.isFinite(ts)) return null;
  const seconds = Math.max(0, Math.round((now - ts) / 1000));
  if (seconds < 60) return { unit: 'seconds', count: seconds };
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) return { unit: 'minutes', count: minutes };
  return { unit: 'hours', count: Math.round(minutes / 60) };
}

const KNOWN_DISPLAY_LABELS: Record<string, string> = {
  'OpenCode Provider': 'usage.plan.opencode',
  'Gemini Models': 'usage.group.gemini',
  'Claude & GPT Models': 'usage.group.claudeGpt',
  Quota: 'usage.window.unknown',
  'Quota available': 'usage.resetAvailable',
};

export function knownDisplayLabelKey(raw?: string): string | null {
  const value = String(raw || '').trim();
  if (!value) return null;
  return KNOWN_DISPLAY_LABELS[value] || null;
}

export function displayLabelKey(raw?: string, groupKey?: string): string | null {
  const fromRaw = knownDisplayLabelKey(raw);
  if (fromRaw) return fromRaw;
  switch (
    String(groupKey || '')
      .trim()
      .toLowerCase()
  ) {
    case 'gemini':
      return 'usage.group.gemini';
    case 'claude_gpt':
      return 'usage.group.claudeGpt';
    default:
      return null;
  }
}

export function windowKindLabelKey(kind?: string): string | null {
  switch (String(kind || '').trim()) {
    case '5h':
    case 'daily':
    case 'claude_5h':
    case 'claude_five_hour':
      return 'usage.window.fiveHour';
    case 'weekly':
    case 'claude_weekly':
      return 'usage.window.weekly';
    case 'unknown':
      return 'usage.window.unknown';
    default:
      return null;
  }
}

export function capabilityLabelKey(key: string): string {
  return `usage.capability.${key}`;
}

export type ParsedReset =
  | { kind: 'available' }
  | { kind: 'sameDay'; time: string }
  | { kind: 'crossDay'; time: string; day: number; monthIndex: number };

const MONTH_INDEX: Record<string, number> = {
  jan: 0,
  feb: 1,
  mar: 2,
  apr: 3,
  may: 4,
  jun: 5,
  jul: 6,
  aug: 7,
  sep: 8,
  oct: 9,
  nov: 10,
  dec: 11,
};

export function parseResetDescription(raw?: string): ParsedReset | null {
  const value = String(raw || '').trim();
  if (!value) return null;
  if (/^quota available$/i.test(value)) return { kind: 'available' };
  const sameDay = /^resets (\d{1,2}:\d{2})$/i.exec(value);
  if (sameDay) return { kind: 'sameDay', time: sameDay[1] };
  const crossDay = /^resets (\d{1,2}:\d{2}) on (\d{1,2}) ([A-Za-z]{3})$/i.exec(value);
  if (!crossDay) return null;
  const monthIndex = MONTH_INDEX[crossDay[3].toLowerCase()];
  if (monthIndex == null) return null;
  return {
    kind: 'crossDay',
    time: crossDay[1],
    day: Number(crossDay[2]),
    monthIndex,
  };
}

export function formatResetDescription(
  raw: string | undefined,
  locale: string,
  translate: (key: string, options?: Record<string, string>) => string,
): string | null {
  const parsed = parseResetDescription(raw);
  if (!parsed) return null;
  if (parsed.kind === 'available') return translate('usage.resetAvailable');
  if (parsed.kind === 'sameDay') return translate('usage.resetToday', { time: parsed.time });
  const date = new Intl.DateTimeFormat(locale, { day: 'numeric', month: 'short' }).format(
    new Date(2000, parsed.monthIndex, parsed.day),
  );
  return translate('usage.resetOn', { time: parsed.time, date });
}

export function groupAccountsByProvider(
  accounts: ProviderAccount[],
): Array<{ provider: string; accounts: ProviderAccount[] }> {
  const map = new Map<string, ProviderAccount[]>();
  for (const account of accounts) {
    const key = account.provider || 'unknown';
    const list = map.get(key) || [];
    list.push(account);
    map.set(key, list);
  }
  return [...map.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([provider, items]) => ({ provider, accounts: items }));
}
