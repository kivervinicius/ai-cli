import { describe, expect, it } from 'vitest';
import { NexusAPIError } from '../../nexus/api';
import { clarificationFromError, unresolvedBlocking } from './clarificationModel';

const checkpoint = {
  id: 'clr-1',
  project_id: 'proj-1',
  goal: 'Build product',
  status: 'PENDING' as const,
  intent: { intent: 'Build product', scope: 'project', risk_level: 'medium', identified_goals: [], constraints: [], assumptions: [], created_at: '' },
  unknowns: [
    { key: 'platform', level: 'BLOCKING' as const, question: 'Platform?', rationale: 'Changes architecture', suggested_options: ['web', 'mobile'], is_resolved: false },
    { key: 'theme', level: 'LOW_IMPACT' as const, question: 'Theme?', rationale: '', default_choice: 'system', answer: 'system', is_resolved: true },
  ],
  facts: {},
};

describe('clarification model', () => {
  it('extracts a structured checkpoint only from clarification_required errors', () => {
    const err = new NexusAPIError(409, { error: 'clarification_required', clarification: checkpoint }, 'clarification_required');
    expect(clarificationFromError(err)?.id).toBe('clr-1');
    expect(clarificationFromError(new Error('boom'))).toBeNull();
  });

  it('returns only unresolved blocking items', () => {
    expect(unresolvedBlocking(checkpoint)).toHaveLength(1);
    expect(unresolvedBlocking(checkpoint)[0].key).toBe('platform');
  });
});
