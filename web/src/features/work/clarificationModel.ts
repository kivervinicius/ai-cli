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

/**
 * A project already owns a validated canonical repository path. Use it to
 * answer path-shaped clarification checkpoints without guessing any other
 * requirement. The value remains visible and editable in the confirmation UI.
 */
export function seedProjectPathAnswers(checkpoint: ClarificationCheckpoint, projectPath: string) {
  const answers = Object.fromEntries(
    checkpoint.unknowns
      .filter((item) => item.answer?.trim())
      .map((item) => [item.key, item.answer?.trim() || '']),
  );
  const path = projectPath.trim();
  if (!path) return answers;

  for (const item of unresolvedBlocking(checkpoint)) {
    const key = item.key.toLowerCase();
    const question = item.question.toLowerCase();
    const isPathQuestion = /(path|workspace|repository|repo|diretório|caminho)/.test(key)
      || /(absolute path|caminho absoluto|repository path|workspace path|raiz do projeto)/.test(question);
    if (isPathQuestion && !answers[item.key]) answers[item.key] = path;
  }
  return answers;
}
