import { describe, expect, it } from 'vitest';
import { agentForRuntime, terminalSurfaceIDForRuntime } from './runtimeAgentMapping';

const agents = [
  { id: 'agt-a', project_id: 'p1', name: 'A' },
  { id: 'agt-b', project_id: 'p1', name: 'B' },
] as any[];

describe('runtime to Agent mapping', () => {
  it('maps only by explicit agent_id and never by project fallback', () => {
    expect(agentForRuntime({ runtime_id: 'rt-x', agent_id: 'agt-b' } as any, agents)?.id).toBe(
      'agt-b',
    );
    expect(agentForRuntime({ runtime_id: 'agt-a' } as any, agents)).toBeUndefined();
  });

  it('uses the canonical Agent terminal surface id', () => {
    expect(terminalSurfaceIDForRuntime({ runtime_id: 'rt-x', agent_id: 'agt-a' } as any)).toBe(
      'agent:agt-a:terminal',
    );
    expect(terminalSurfaceIDForRuntime({ runtime_id: 'rt-x' } as any)).toBeUndefined();
  });
});
