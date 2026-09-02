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
