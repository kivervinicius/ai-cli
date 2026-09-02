import type { WorkspaceSurface } from '../workspace/model';

export type ProjectSurfaceKind =
  | 'projects'
  | 'overview'
  | 'work'
  | 'missions'
  | 'agents'
  | 'maestro'
  | 'sessions'
  | 'settings'
  | 'resources'
  | 'legacy-runtimes'
  | 'legacy-providers'
  | 'legacy-events';

const labels: Record<ProjectSurfaceKind, string> = {
  projects: 'Workspace Desktops',
  overview: 'Overview',
  work: 'Composer',
  missions: 'Flow Runs',
  agents: 'Agents',
  maestro: 'Maestro',
  sessions: 'Sessions',
  settings: 'Settings',
  resources: 'Resources',
  'legacy-runtimes': 'Runtime Control',
  'legacy-providers': 'Providers',
  'legacy-events': 'Events',
};

export function projectSurface(projectId: string, kind: ProjectSurfaceKind): WorkspaceSurface {
  const key =
    kind === 'projects'
      ? 'desktopsTitle'
      : kind === 'legacy-runtimes'
      ? 'runtimes'
      : kind === 'legacy-providers'
      ? 'providers'
      : kind === 'legacy-events'
      ? 'events'
      : kind;
  const titleKey = kind === 'projects' ? 'projectManager.desktopsTitle' : `nav.${key}`;
  return {
    id: `project:${projectId}:${kind}`,
    type: kind,
    title: labels[kind],
    titleKey,
    closable: kind !== 'overview',
    data: { projectId },
  };
}

export function agentTerminalSurface(agentId: string, agentName: string, initialPrompt = ''): WorkspaceSurface {
  return {
    id: `agent:${agentId}:terminal`,
    type: 'terminal',
    title: agentName,
    subtitle: 'Persistent Agent terminal',
    data: initialPrompt ? { agentId, initialPrompt } : { agentId },
    closable: true,
  };
}

export function agentConfigSurface(agentId: string, agentName: string): WorkspaceSurface {
  return {
    id: `agent:${agentId}:config`,
    type: 'agent-config',
    title: `${agentName} · Configure`,
    titleKey: 'workspace.configureAgent',
    titleParams: { name: agentName },
    data: { agentId },
    closable: true,
  };
}

export function projectShellSurface(projectId: string, runtimeId: string, title = 'Project Shell'): WorkspaceSurface {
  const logicalKey = `shell:${runtimeId}`;
  return {
    id: logicalKey,
    viewId: `view:${logicalKey}`,
    logicalKey,
    type: 'project-shell',
    title,
    subtitle: 'Project shell',
    data: { projectId, runtimeId },
    closable: true,
  };
}

export function flowRunSurface(runId: string, title = 'Flow Run'): WorkspaceSurface {
  const logicalKey = `flow-run:${runId}`;
  return {
    id: logicalKey,
    viewId: `view:${logicalKey}`,
    logicalKey,
    type: 'flow-run',
    title,
    subtitle: 'Durable Flow execution',
    data: { runId },
    closable: true,
  };
}
