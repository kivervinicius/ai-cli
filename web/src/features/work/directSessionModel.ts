export interface DirectResourceLike {
  id: string;
  provider: string;
  profile: string;
  authenticated: boolean;
  available: boolean;
}

const providerLabel = (provider: string) => provider.trim().replace(/(^|[-_\s])\w/g, (part) => part.toUpperCase());

export function buildDirectAgentName(prompt: string, provider: string): string {
  const label = providerLabel(provider || 'AI');
  const cleaned = (prompt || '').trim().replace(/\s+/g, ' ');
  if (!cleaned) return `${label} · Direct Session`;
  const words = cleaned.split(' ');
  const short = words.slice(0, 3).join(' ');
  return `${label} · ${short}`;
}

export function directAccountTitle(resource: { display_name?: string; provider: string; profile: string }): string {
  const name = (resource.display_name || '').trim();
  if (name) return name;
  const profile = (resource.profile || '').trim();
  if (!profile || profile === 'default') return resource.provider;
  return `${resource.provider} (${profile})`;
}

export function directQuotaPercent(resource: {
  quota_remaining?: number;
  avail_reasons?: { unknown_quota?: boolean };
  quota_view?: { status?: string };
}): number | null {
  const known = !resource.avail_reasons?.unknown_quota && resource.quota_view?.status !== 'UNKNOWN';
  if (!known || !Number.isFinite(resource.quota_remaining)) return null;
  const raw = resource.quota_remaining ?? 0;
  return Math.round(raw * (raw <= 1 ? 100 : 1));
}

export function eligibleDirectResources<T extends DirectResourceLike>(accounts: T[], preferProvider = ''): T[] {
  const preferred = preferProvider.trim().toLowerCase();
  return accounts
    .filter((account) => account.authenticated && account.available)
    .sort((a, b) => {
      const ap = a.provider.toLowerCase() === preferred ? 1 : 0;
      const bp = b.provider.toLowerCase() === preferred ? 1 : 0;
      if (ap !== bp) return bp - ap;
      return `${a.provider}:${a.profile}`.localeCompare(`${b.provider}:${b.profile}`);
    });
}
