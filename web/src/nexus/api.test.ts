import { beforeEach, describe, expect, it, vi } from 'vitest';
import { NexusAPIError, nexusApi } from './api';

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
