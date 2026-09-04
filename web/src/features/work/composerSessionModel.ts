export interface ComposerSessionSummary {
  id: string;
  state: string;
  updated_at: string;
}

export function selectResumableComposerSession(sessions: ComposerSessionSummary[]): string | undefined {
  const active = (sessions || []).filter((session) => session.state !== 'FINALIZED');
  if (active.length === 0) return undefined;
  return active.slice().sort((a, b) => (b.updated_at || '').localeCompare(a.updated_at || ''))[0]?.id;
}
