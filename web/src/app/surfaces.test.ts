import { describe, expect, it } from 'vitest';
import { agentConfigSurface, agentTerminalSurface, projectSurface } from './surfaces';

describe('workspace surfaces', () => {
  it('creates stable project surface ids', () => expect(projectSurface('prj_1', 'overview').id).toBe('project:prj_1:overview'));
  it('keys terminals by AgentID', () => expect(agentTerminalSurface('agt_9', 'Backend').id).toBe('agent:agt_9:terminal'));
  it('keys config by AgentID', () => expect(agentConfigSurface('agt_9', 'Backend').id).toBe('agent:agt_9:config'));
});
