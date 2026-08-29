import type { WorkspaceSurface } from '../workspace/model';

export type ProjectSurfaceKind = 'overview' | 'work' | 'missions' | 'agents' | 'maestro' | 'sessions' | 'settings' | 'resources' | 'legacy-runtimes' | 'legacy-providers' | 'legacy-events';

const labels: Record<ProjectSurfaceKind, string> = {
  overview: 'Overview', work: 'Work', missions: 'Plan', agents: 'Agents', maestro: 'Maestro', sessions: 'Sessions', settings: 'Settings', resources: 'Resources',
  'legacy-runtimes': 'Runtime Control', 'legacy-providers': 'Providers', 'legacy-events': 'Events',
};

export function projectSurface(projectId: string, kind: ProjectSurfaceKind): WorkspaceSurface {
  return { id: `project:${projectId}:${kind}`, type: kind, title: labels[kind], closable: kind !== 'overview', data: { projectId } };
}
export function agentTerminalSurface(agentId: string, agentName: string): WorkspaceSurface {
  return { id: `agent:${agentId}:terminal`, type: 'terminal', title: agentName, subtitle: 'Persistent Agent terminal', data: { agentId }, closable: true };
}
export function agentConfigSurface(agentId: string, agentName: string): WorkspaceSurface {
  return { id: `agent:${agentId}:config`, type: 'agent-config', title: `${agentName} · Configure`, data: { agentId }, closable: true };
}
