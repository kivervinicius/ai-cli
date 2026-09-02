import { describe, expect, it } from 'vitest';
import { agentConfigSurface, agentTerminalSurface, flowRunSurface, projectShellSurface, projectSurface } from './surfaces';

describe('workspace surfaces', () => {
  it('creates stable project surface ids', () => expect(projectSurface('prj_1', 'overview').id).toBe('project:prj_1:overview'));
  it('creates projects hub surface id', () => expect(projectSurface('prj_1', 'projects').id).toBe('project:prj_1:projects'));
  it('keys terminals by AgentID', () => expect(agentTerminalSurface('agt_9', 'Backend').id).toBe('agent:agt_9:terminal'));
  it('keys config by AgentID', () => expect(agentConfigSurface('agt_9', 'Backend').id).toBe('agent:agt_9:config'));
  it('keys independent Project Shell views by runtime id', () => {
    const shell = projectShellSurface('prj_1', 'rt_shell_1', 'Shell 1');
    expect(shell.type).toBe('project-shell');
    expect(shell.logicalKey).toBe('shell:rt_shell_1');
    expect(shell.viewId).toBe('view:shell:rt_shell_1');
    expect(shell.data).toMatchObject({ projectId: 'prj_1', runtimeId: 'rt_shell_1' });
  });
  it('keys Flow Run views by durable run id', () => {
    const surface = flowRunSurface('run_123', 'Flow Run · 123');
    expect(surface.type).toBe('flow-run');
    expect(surface.logicalKey).toBe('flow-run:run_123');
    expect(surface.viewId).toBe('view:flow-run:run_123');
    expect(surface.data).toEqual({ runId: 'run_123' });
  });

});
