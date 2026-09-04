export interface ComposerSessionSummary {
  id: string;
  state: string;
  updated_at: string;
}

export function selectResumableComposerSession(sessions: ComposerSessionSummary[]): string | undefined {
  return sessions.find((session) => session.state !== 'FINALIZED')?.id;
}
