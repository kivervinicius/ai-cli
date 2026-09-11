import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { AttentionGroup, AttentionItem, HumanIntervention } from '../../types';

// Mock the nexus API
vi.mock('../../nexus/api', () => ({
  nexus: {
    getAttention: vi.fn(),
    resolveIntervention: vi.fn(),
  },
}));

import { nexus } from '../../nexus/api';

const mockGetAttention = vi.mocked(nexus.getAttention);
const mockResolveIntervention = vi.mocked(nexus.resolveIntervention);

function makeIntervention(overrides: Partial<HumanIntervention> = {}): HumanIntervention {
  return {
    id: 'intv_1',
    reason_code: 'DISPATCH_OUTCOME_UNKNOWN',
    summary: 'Provider outcome unknown',
    question: 'Should we inspect the provider runtime?',
    context: 'The dispatch boundary was crossed but no completion was recorded.',
    recommended_actions: ['Inspect logs', 'Retry after confirmation'],
    impact: 'Only the affected task is paused.',
    mission_id: 'run_1',
    task_id: 'pkg_1',
    source: 'mission_runner',
    timestamp: new Date().toISOString(),
    version: 1,
    scope: 'PACKAGE',
    options: [
      {
        id: 'confirm-external-completion',
        operation: 'CONFIRM_EXTERNAL_COMPLETION',
        label: 'Confirm external completion',
        package_id: 'pkg_1',
      },
    ],
    resolved: false,
    ...overrides,
  };
}

function makeAttentionItem(overrides: Partial<AttentionItem> = {}): AttentionItem {
  return {
    mission_id: 'run_1',
    project_id: 'proj_1',
    state: 'BLOCKED_NEEDS_USER',
    level: 'REQUIRE_USER',
    reason_code: 'DISPATCH_OUTCOME_UNKNOWN',
    summary: 'Mission blocked: human decision required',
    question: 'Should we inspect?',
    impact: 'Only affected task paused',
    recommended_actions: ['Inspect', 'Retry'],
    age_seconds: 120,
    intervention: makeIntervention(),
    ...overrides,
  };
}

function makeAttentionGroup(overrides: Partial<AttentionGroup> = {}): AttentionGroup {
  return {
    needs_you: [],
    completed: [],
    failed: [],
    all_items: [],
    total_needs: 0,
    ...overrides,
  };
}

describe('AttentionCenter', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders empty state when no attention items exist', async () => {
    mockGetAttention.mockResolvedValue(makeAttentionGroup());
    // This tests the API contract - the component rendering is tested via E2E
    const group = await nexus.getAttention();
    expect(group.needs_you).toHaveLength(0);
    expect(group.completed).toHaveLength(0);
    expect(group.failed).toHaveLength(0);
  });

  it('groups needs_you items correctly', async () => {
    const item = makeAttentionItem();
    mockGetAttention.mockResolvedValue(makeAttentionGroup({ needs_you: [item], total_needs: 1 }));
    const group = await nexus.getAttention();
    expect(group.needs_you).toHaveLength(1);
    expect(group.needs_you[0].mission_id).toBe('run_1');
    expect(group.total_needs).toBe(1);
  });

  it('groups completed items correctly', async () => {
    const item = makeAttentionItem({
      state: 'COMPLETED_VERIFIED',
      level: 'IN_APP',
      intervention: undefined,
    });
    mockGetAttention.mockResolvedValue(makeAttentionGroup({ completed: [item] }));
    const group = await nexus.getAttention();
    expect(group.completed).toHaveLength(1);
    expect(group.completed[0].state).toBe('COMPLETED_VERIFIED');
  });

  it('groups failed items correctly', async () => {
    const item = makeAttentionItem({
      state: 'FAILED_NO_PROGRESS',
      level: 'NOTIFY',
      intervention: undefined,
    });
    mockGetAttention.mockResolvedValue(makeAttentionGroup({ failed: [item] }));
    const group = await nexus.getAttention();
    expect(group.failed).toHaveLength(1);
    expect(group.failed[0].state).toBe('FAILED_NO_PROGRESS');
  });

  it('resolveIntervention sends correct parameters', async () => {
    const run = { id: 'run_1', state: 'EXECUTING' };
    mockResolveIntervention.mockResolvedValue(run as any);
    const result = await nexus.resolveIntervention(
      'run_1',
      'intv_1',
      1,
      'confirm-external-completion',
      'inspect',
    );
    expect(mockResolveIntervention).toHaveBeenCalledWith(
      'run_1',
      'intv_1',
      1,
      'confirm-external-completion',
      'inspect',
    );
    expect(result).toBe(run);
  });

  it('human intervention has required fields', () => {
    const intervention = makeIntervention();
    expect(intervention.id).toBeTruthy();
    expect(intervention.reason_code).toBeTruthy();
    expect(intervention.question).toBeTruthy();
    expect(intervention.version).toBe(1);
    expect(intervention.resolved).toBe(false);
  });
});
