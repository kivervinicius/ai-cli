import { describe, expect, it, vi, beforeEach } from 'vitest';

vi.mock('./api', () => ({
  nexus: {
    recoverAgent: vi.fn(),
    startAgent: vi.fn(),
  },
}));

import { nexus } from './api';
import {
  isRequiredResourceError,
  recoverOrStartAgent,
  RequiredResourceError,
} from './agentRecover';

describe('recoverOrStartAgent', () => {
  beforeEach(() => {
    vi.mocked(nexus.recoverAgent).mockReset();
    vi.mocked(nexus.startAgent).mockReset();
  });

  it('returns recover runtime on success', async () => {
    vi.mocked(nexus.recoverAgent).mockResolvedValue({ runtime: { runtime_id: 'rt_ok' } as any });
    const result = await recoverOrStartAgent('agt_1');
    expect(result.runtime?.runtime_id).toBe('rt_ok');
    expect(nexus.startAgent).not.toHaveBeenCalled();
  });

  it('falls back to start when recover says STOPPED', async () => {
    vi.mocked(nexus.recoverAgent).mockRejectedValue(
      new Error('agent is STOPPED (use StartAgent to restart)'),
    );
    vi.mocked(nexus.startAgent).mockResolvedValue({ runtime: { runtime_id: 'rt_started' } as any });
    const result = await recoverOrStartAgent('agt_1');
    expect(result.runtime?.runtime_id).toBe('rt_started');
  });

  it('throws RequiredResourceError without calling start when recover lacks provider', async () => {
    vi.mocked(nexus.recoverAgent).mockRejectedValue(
      new Error('REQUIRED_RESOURCE_SELECTION: agent agt_1 has no configured provider'),
    );
    await expect(recoverOrStartAgent('agt_1')).rejects.toBeInstanceOf(RequiredResourceError);
    expect(nexus.startAgent).not.toHaveBeenCalled();
  });

  it('soft-succeeds on legacy already-alive without start', async () => {
    vi.mocked(nexus.recoverAgent).mockRejectedValue(
      new Error('agent runtime is already alive (no recovery needed)'),
    );
    const result = await recoverOrStartAgent('agt_1');
    expect(result.runtime).toBeUndefined();
    expect(nexus.startAgent).not.toHaveBeenCalled();
    expect(isRequiredResourceError(new RequiredResourceError('REQUIRED_RESOURCE_SELECTION'))).toBe(
      true,
    );
  });
});
