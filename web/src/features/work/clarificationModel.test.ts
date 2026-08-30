import { describe, expect, it } from 'vitest';
import { NexusAPIError } from '../../nexus/api';
import { clarificationFromError, seedProjectPathAnswers, unresolvedBlocking } from './clarificationModel';

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

  it('seeds only path-shaped blocking questions from the project path', () => {
    const pathCheckpoint = {
      ...checkpoint,
      unknowns: [
        { key: 'repository_path', level: 'BLOCKING' as const, question: 'What is the absolute path to the repository?', rationale: '', is_resolved: false },
        { key: 'architecture', level: 'BLOCKING' as const, question: 'Which architecture?', rationale: '', is_resolved: false },
      ],
    };
    expect(seedProjectPathAnswers(pathCheckpoint, '/tmp/project')).toEqual({ repository_path: '/tmp/project' });
  });

  it('preserves an answer already supplied by the provider', () => {
    const answered = { ...checkpoint, unknowns: [{ ...checkpoint.unknowns[0], key: 'workspace_path', answer: '/already/selected', is_resolved: false }] };
    expect(seedProjectPathAnswers(answered, '/tmp/project')).toEqual({ workspace_path: '/already/selected' });
  });
});
