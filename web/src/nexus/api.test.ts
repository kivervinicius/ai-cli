import { beforeEach, describe, expect, it, vi } from 'vitest';
import { NexusAPIError, nexusApi } from './api';
import type { AutonomyContract } from '../types';

describe('nexus API request errors', () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
    vi.stubGlobal('fetch', vi.fn());
  });

  it('preserves status and structured clarification payload on 409', async () => {
    const payload = {
      error: 'clarification_required',
      clarification: {
        id: 'clr-1',
        project_id: 'proj-1',
        goal: 'Build product',
        status: 'PENDING',
        intent: { intent: 'Build product', scope: 'product', risk_level: 'MEDIUM' },
        unknowns: [{ key: 'target', question: 'Which target?', level: 'BLOCKING', options: ['web', 'mobile'] }],
        facts: {},
      },
    };
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      status: 409,
      statusText: 'Conflict',
      json: () => Promise.resolve(payload),
    }));

    try {
      await nexusApi.createPlan('proj-1', { goal: 'Build product', auto_plan: true });
      throw new Error('expected request to fail');
    } catch (error) {
      expect(error).toBeInstanceOf(NexusAPIError);
      const apiError = error as NexusAPIError<typeof payload>;
      expect(apiError.status).toBe(409);
      expect(apiError.payload.clarification.id).toBe('clr-1');
      expect(apiError.message).toBe('clarification_required');
    }
  });
});

describe('mission manual-control API routes', () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it('uses dedicated take-control endpoint instead of generic pause', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({ id: 'run-1', state: 'PAUSED' }) });
    vi.stubGlobal('fetch', fetchMock);
    await nexusApi.takeControlRun('run-1', 'manual');
    expect(String(fetchMock.mock.calls[0][0])).toContain('/api/v1/runs/run-1/take-control');
  });

  it('uses dedicated return-to-mission endpoint instead of generic resume', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({ id: 'run-1', state: 'EXECUTING' }) });
    vi.stubGlobal('fetch', fetchMock);
    await nexusApi.returnToMission('run-1');
    expect(String(fetchMock.mock.calls[0][0])).toContain('/api/v1/runs/run-1/return-to-mission');
  });
});

describe('mission autonomy contract API', () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  const contract: AutonomyContract = {
    max_retries: 4,
    max_total_iterations: 80,
    max_no_progress: 3,
    package_timeout_seconds: 1800,
    auto_remediate: true,
    require_verification: true,
    disallow_destructive_git: true,
    allowed_file_patterns: ['src/**'],
    verification_commands: ['npm test'],
    escalate_on_failure: true,
    allow_tool_auto_approval: false,
    allow_git_push: false,
    allow_deploy: false,
    allow_external_network: false,
    allow_secret_access: false,
    allow_paid_services: false,
  };

  it('sends the approved contract as a nested run contract without forcing a default agent', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({ id: 'run-1' }) });
    vi.stubGlobal('fetch', fetchMock);
    await nexusApi.runPlan('plan-1', { contract: { ...contract }, autonomous: true, approvedRevision: 7 });
    const body = JSON.parse(String(fetchMock.mock.calls[0][1]?.body));
    expect(body.contract).toEqual(contract);
    expect(body.approved_revision).toBe(7);
    expect(body.agent_id).toBeUndefined();
  });

  it('persists the same approved contract in a scheduled Mission', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({ id: 'schedule-1' }) });
    vi.stubGlobal('fetch', fetchMock);
    await nexusApi.schedulePlan('plan-1', 'WHEN_RESOURCES', { contract: { ...contract }, approvedRevision: 7 });
    const body = JSON.parse(String(fetchMock.mock.calls[0][1]?.body));
    expect(body.contract).toEqual(contract);
    expect(body.approved_revision).toBe(7);
    expect(body.agent_id).toBeUndefined();
  });
});
