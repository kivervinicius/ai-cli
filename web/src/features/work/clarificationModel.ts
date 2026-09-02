import { NexusAPIError } from '../../nexus/api';
import type { ClarificationCheckpoint } from '../../types';

type ClarificationPayload = { error?: string; clarification?: ClarificationCheckpoint };

export function clarificationFromError(error: unknown): ClarificationCheckpoint | null {
  if (!(error instanceof NexusAPIError) || error.status !== 409) return null;
  const payload = error.payload as ClarificationPayload;
  if (payload?.error !== 'clarification_required' || !payload.clarification) return null;
  return payload.clarification;
}

export function unresolvedBlocking(checkpoint: ClarificationCheckpoint) {
  return checkpoint.unknowns.filter((item) => item.level === 'BLOCKING' && !item.is_resolved);
}
