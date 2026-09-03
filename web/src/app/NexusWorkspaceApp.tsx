import React, { useEffect, useMemo, useState } from 'react';
import { api, initSession, rotateSession, type BrowserSession } from '../api';
import { setNexusCSRF, nexus } from '../nexus/api';
import { Spinner } from '../design-system';
import { ThemeProvider } from '../design-system';
import { WorkspaceProvider, useWorkspace } from '../workspace/WorkspaceProvider';
import { WorkspacePresentationProvider } from '../workspace/WorkspacePresentationProvider';
import { WorkspaceRenderer } from '../workspace/WorkspaceRenderer';
import { createWorkspace, type WorkspaceSurface } from '../workspace/model';
import { serializeWorkspace } from '../workspace/state';
import { ProjectRail } from '../features/projects/ProjectRail';
import { ProjectHub } from '../features/projects/ProjectHub';
import { CommandPalette } from './commands/CommandPalette';
import type { NexusCommand } from './commands/registry';
import { ProductTour } from './tour/ProductTour';
import { WelcomeModal } from './modals/WelcomeModal';
import { MaestroControlModal } from './modals/MaestroControlModal';
import { NexusShell } from './NexusShell';
import { WorkspaceSurfaceHost } from './WorkspaceSurfaceHost';
import { agentConfigSurface, agentTerminalSurface, flowRunSurface, projectShellSurface, projectSurface } from './surfaces';
import { agentForRuntime, terminalSurfaceIDForRuntime } from './runtimeAgentMapping';
import { buildDocumentTitle } from './documentTitle';
import { planFocusAttention, type RadarRuntimeItem } from './attentionRadarModel';
import { resolveProjectSelection } from './projectSelection';
import { useNexusData } from './useNexusData';
import type { Agent, MissionRun, Project } from '../types';
import { useTranslation } from 'react-i18next';

const selectedProjectKey = 'iapro:nexus:selected-project:v1';
const tourKey = 'iapro:nexus:tour-complete:v1';

export const NexusWorkspaceApp: React.FC<{ popoutSurface?: WorkspaceSurface }> = ({ popoutSurface }) => {
  const { t } = useTranslation();
  const [sessionReady, setSessionReady] = useState(false);
  const [authenticated, setAuthenticated] = useState(true);

  useEffect(() => {
    let rotationTimer: ReturnType<typeof window.setTimeout> | undefined;
    const scheduleRotation = (session: BrowserSession) => {
      if (!session.expires_at) return;
      const expiresAt = Date.parse(session.expires_at);
      if (!Number.isFinite(expiresAt)) return;
      const delay = Math.max(30_000, expiresAt - Date.now() - 30 * 60 * 1000);
      rotationTimer = window.setTimeout(async () => {
        const rotated = await rotateSession();
        setAuthenticated(rotated.authenticated);
        if (rotated.authenticated) scheduleRotation(rotated);
      }, delay);
    };
    const onExpired = () => setAuthenticated(false);
    window.addEventListener('nexus:session-expired', onExpired);
    initSession()
      .then((session) => {
        setAuthenticated(session.authenticated);
        if (session.csrf_token) setNexusCSRF(session.csrf_token);
        if (session.authenticated) scheduleRotation(session);
      })
      .finally(() => setSessionReady(true));
    return () => {
      window.removeEventListener('nexus:session-expired', onExpired);
      if (rotationTimer) window.clearTimeout(rotationTimer);
    };
  }, []);

  if (!sessionReady) {
    return (
      <div className="nx-app-loading">
        <Spinner label={t('app.starting')} />
      </div>
    );
  }

  if (!authenticated) {
    return (
      <ThemeProvider>
        <div className="nx-app-unauthorized" style={{
          minHeight: '100vh',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          padding: '24px',
          background: 'var(--color-background, #080a0f)',
          color: 'var(--color-text, #f8fafc)',
          textAlign: 'center',
          fontFamily: 'system-ui, -apple-system, sans-serif'
        }}>
          <div style={{
            maxWidth: '460px',
            background: 'var(--color-surface, #10141e)',
            border: '1px solid var(--color-border, #1e293b)',
            borderRadius: '12px',
            padding: '32px',
            boxShadow: '0 20px 25px -5px rgba(0,0,0,0.5)'
          }}>
            <div style={{
              width: '48px',
              height: '48px',
              borderRadius: '50%',
              background: 'rgba(56, 189, 248, 0.1)',
              color: '#38bdf8',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              margin: '0 auto 16px',
              fontSize: '20px',
              fontWeight: 700
            }}>N</div>
            <h2 style={{ fontSize: '18px', fontWeight: 600, marginBottom: '8px' }}>Sessão Expirada ou Não Autenticada</h2>
            <p style={{ fontSize: '13px', color: '#94a3b8', lineHeight: 1.5, marginBottom: '24px' }}>
              Em <code>127.0.0.1</code>/<code>localhost</code>, reabra o mesmo link <strong>Bootstrap</strong> impresso no terminal do <code>nexus web</code> — o token local pode ser reutilizado enquanto o servidor estiver no ar.
              Se o link não estiver mais disponível (ou se o bind não for loopback), encerre com <code>Ctrl+C</code>, rode <code>nexus web</code> de novo e abra a URL exibida.
            </p>
            <button
              type="button"
              onClick={() => window.location.reload()}
              style={{
                background: '#38bdf8',
                color: '#090d16',
                border: 'none',
                borderRadius: '6px',
                padding: '10px 20px',
                fontSize: '13px',
                fontWeight: 600,
                cursor: 'pointer'
              }}
            >
              Recarregar Aplicação
            </button>
          </div>
        </div>
      </ThemeProvider>
    );
  }

  return (
    <ThemeProvider>
      <NexusWorkspaceSession popoutSurface={popoutSurface} />
    </ThemeProvider>
  );
};

const NexusWorkspaceSession: React.FC<{ popoutSurface?: WorkspaceSurface }> = ({ popoutSurface }) => {
  const { t } = useTranslation();
  const data = useNexusData();
  const [selectedId, setSelectedId] = useState(() => window.localStorage.getItem(selectedProjectKey) || '');
  const selected = resolveProjectSelection(data.projects, selectedId);
  const [layout, setLayout] = useState<string | undefined>();

  useEffect(() => {
    if (!selected) return;
    setSelectedId(selected.id);
    window.localStorage.setItem(selectedProjectKey, selected.id);
    void data.refreshAgents(selected.id);
    nexus
      .getProject(selected.id)
      .then((detail) => setLayout(detail.layout || undefined))
      .catch(() => setLayout(undefined));
  }, [selected?.id]);

  if (data.loading) {
    return (
      <div className="nx-app-loading">
        <Spinner label={t('app.loading')} />
      </div>
    );
  }

  if (!selected) {
    return (
      <ProjectHub
        onCreated={(project) => {
          data.setProjects((current) => [project, ...current]);
          setSelectedId(project.id);
        }}
      />
    );
  }

  const initial = popoutSurface ? serializeWorkspace(createWorkspace(popoutSurface)) : layout;

  return (
    <WorkspaceProvider
      key={`${selected.id}:${popoutSurface?.id || 'main'}`}
      projectId={selected.id}
      initialLayout={initial}
      saveLayout={popoutSurface ? undefined : (next) => nexus.saveLayout(selected.id, next)}
      onSurfaceClosed={async (surface) => {
        if (surface.type === 'project-shell' && surface.data?.runtimeId) {
          await api.stopRuntime(surface.data.runtimeId).catch(() => undefined);
          await data.refreshGlobal().catch(() => undefined);
        }
      }}
    >
      <WorkspacePresentationProvider projectId={selected.id}>
        <WorkspaceCoordinator
          project={selected}
          setProject={(project) => setSelectedId(project.id)}
          data={data}
          popout={Boolean(popoutSurface)}
        />
      </WorkspacePresentationProvider>
    </WorkspaceProvider>
  );
};

const WorkspaceCoordinator: React.FC<{
  project: Project;
  setProject: (project: Project) => void;
  data: ReturnType<typeof useNexusData>;
  popout: boolean;
}> = ({ project, setProject, data, popout }) => {
  const { t, i18n } = useTranslation();
  const workspace = useWorkspace();
  const [railOpen, setRailOpen] = useState(false);
  const [palette, setPalette] = useState(false);
  const [welcomeOpen, setWelcomeOpen] = useState(false);
  const [maestroControlOpen, setMaestroControlOpen] = useState(false);
  const [tour, setTour] = useState(false);
  const [shellError, setShellError] = useState('');
  const [flowRuns, setFlowRuns] = useState<MissionRun[]>([]);

  useEffect(() => {
    let mounted = true;
    void nexus.getRuns()
      .then((runs) => {
        if (!mounted) return;
        const active = runs.filter((run) => !['COMPLETED_VERIFIED', 'CANCELED_BY_USER', 'FAILED_BUDGET_EXCEEDED'].includes(run.state));
        setFlowRuns(active);
      })
      .catch(() => mounted && setFlowRuns([]));
    return () => { mounted = false; };
  }, [project.id, palette]);

  const open = (surface: WorkspaceSurface) => workspace.open(surface);
  const openKind = (kind: string) => open(projectSurface(project.id, kind as any));
  const terminal = (agent: Agent) => open(agentTerminalSurface(agent.id, agent.name));
  const config = (agent: Agent) => open(agentConfigSurface(agent.id, agent.name));
  const shell = async () => {
    setShellError('');
    try {
      const result = await nexus.startProjectShell(project.id);
      open(projectShellSurface(project.id, result.runtime.runtime_id, result.runtime.title || 'Project Shell'));
      await data.refreshGlobal().catch(() => undefined);
    } catch (error) {
      setShellError(error instanceof Error ? error.message : String(error));
    }
  };

  const popoutSurface = (surface: WorkspaceSurface) => {
    const encoded = encodeURIComponent(
      JSON.stringify({ ...surface, data: { ...surface.data, projectId: project.id } })
    );
    window.open(`${window.location.pathname}?popout=${encoded}`, '_blank', 'popup=yes,width=1200,height=800');
  };

  const commands = useMemo<NexusCommand[]>(
    () => [
      { id: 'projects', label: t('commands.open', { name: t('projectManager.desktopsTitle') }), group: t('commands.project'), keywords: ['workspace', 'desktops', 'hub'], run: () => openKind('projects') },
      { id: 'overview', label: t('commands.open', { name: t('nav.overview') }), group: t('commands.project'), keywords: ['home'], run: () => openKind('overview') },
      { id: 'new-ai-session', label: 'New AI Session', group: t('commands.project'), keywords: ['agent', 'session', 'direct', 'create', 'terminal'], run: () => { openKind('work'); window.dispatchEvent(new CustomEvent('nexus:new-ai-session')); } },
      { id: 'project-shell', label: 'New Project Shell', group: t('commands.project'), keywords: ['shell', 'terminal', 'bash', 'powershell'], run: () => void shell() },
      { id: 'agents', label: t('commands.open', { name: t('nav.agents') }), group: t('commands.project'), keywords: ['fleet', 'workers', 'terminals'], run: () => openKind('agents') },
      ...data.agents.flatMap((agent) => [
        { id: `terminal-${agent.id}`, label: t('commands.open', { name: `${agent.name} terminal` }), group: t('nav.agents'), keywords: ['terminal', agent.role], run: () => terminal(agent) },
        { id: `config-${agent.id}`, label: t('commands.configure', { name: agent.name }), group: t('nav.agents'), keywords: ['settings', agent.role], run: () => config(agent) },
      ]),
      { id: 'work', label: 'Open Composer (optional)', group: t('commands.project'), keywords: ['composer', 'prompt', 'goal', 'plan'], run: () => openKind('work') },
      { id: 'plan', label: 'Open Flow Runs (optional)', group: t('commands.project'), keywords: ['flow', 'mission', 'workplan', 'tasks'], run: () => openKind('missions') },
      { id: 'resources', label: t('commands.open', { name: t('nav.resources') }), group: 'Nexus', keywords: ['quota', 'provider', 'accounts'], run: () => openKind('resources') },
      { id: 'maestro', label: t('commands.open', { name: 'Maestro method' }), group: 'Nexus', keywords: ['skills', 'process', 'verification', 'community'], run: () => openKind('maestro') },
      { id: 'maestro-control', label: t('maestroControl.title'), group: 'Nexus', keywords: ['assist', 'autonomous', 'mode'], run: () => setMaestroControlOpen(true) },
      { id: 'project-manager', label: t('projectManager.title'), group: t('commands.project'), keywords: ['switch', 'create', 'workspace'], run: () => openKind('projects') },
      { id: 'sessions', label: t('commands.open', { name: t('nav.sessions') }), group: t('commands.project'), keywords: ['resume', 'continuity'], run: () => openKind('sessions') },
      { id: 'settings', label: t('commands.open', { name: t('nav.settings') }), group: 'Nexus', keywords: ['theme', 'accessibility'], run: () => openKind('settings') },
      { id: 'runtime', label: t('commands.open', { name: t('nav.runtimes') }), group: t('commands.advanced'), keywords: ['runtime', 'legacy'], run: () => openKind('legacy-runtimes') },
      { id: 'providers', label: t('commands.open', { name: t('nav.providers') }), group: t('commands.advanced'), keywords: ['provider'], run: () => openKind('legacy-providers') },
      ...flowRuns.map((run) => ({
        id: `flow-run-${run.id}`,
        label: `Flow Run · ${run.id.slice(-6)}`,
        group: t('commands.project'),
        keywords: ['flow', 'run', 'mission', run.state],
        run: () => open(flowRunSurface(run.id, `Flow Run · ${run.id.slice(-6)}`)),
      })),
      { id: 'welcome', label: t('welcome.title'), group: t('commands.help'), keywords: ['guide', 'help', 'onboarding'], run: () => setWelcomeOpen(true) },
      { id: 'tour', label: t('commands.tour'), group: t('commands.help'), keywords: ['help', 'onboarding'], run: () => setTour(true) },
    ],
    [project.id, data.agents, flowRuns, t, i18n.language]
  );

  // Global Keyboard Shortcuts (Ctrl+K = Palette, Ctrl+P = Project Manager surface)
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault();
        setPalette((value) => !value);
      }
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'p') {
        event.preventDefault();
        openKind('projects');
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [project.id]);

  useEffect(() => {
    const onShell = () => {
      void shell();
    };
    window.addEventListener('nexus:project-shell', onShell);
    return () => window.removeEventListener('nexus:project-shell', onShell);
  }, [project.id]);

  const handleProjectUpdated = (updated: Project) => {
    data.setProjects((cur) => cur.map((p) => (p.id === updated.id ? updated : p)));
    if (project.id === updated.id) {
      setProject(updated);
    }
  };

  const renderer = (
    <WorkspaceRenderer
      renderSurface={(surface) => (
        <WorkspaceSurfaceHost
          surface={surface}
          project={project}
          projects={data.projects}
          agents={data.agents}
          workspaces={data.workspaces}
          runtimes={data.runtimes}
          providers={data.providers}
          profiles={data.profiles}
          events={data.events}
          refreshAgents={() => data.refreshAgents(project.id)}
          refreshGlobal={data.refreshGlobal}
          onSelectProject={setProject}
          onProjectCreated={(created) => {
            data.setProjects((current) => [created, ...current]);
            setProject(created);
          }}
          onProjectUpdated={handleProjectUpdated}
          openSurface={open}
          onTour={() => setWelcomeOpen(true)}
        />
      )}
      popoutSurface={popoutSurface}
    />
  );

  if (popout) return <div className="nx-popout-shell">{renderer}</div>;

  const rail = (
    <ProjectRail
      projects={data.projects}
      selected={project}
      open={railOpen}
      onClose={() => setRailOpen(false)}
      onSelect={setProject}
      onCreated={(created) => {
        data.setProjects((current) => [created, ...current]);
        setProject(created);
      }}
      onOpenGlobal={(kind) => {
        if (kind === 'overview') openKind('overview');
        else openKind(kind);
      }}
    />
  );

  const handleFocusAttention = (item: RadarRuntimeItem | { runtimeId: string }) => {
    const runtimeId = item.runtimeId;
    const runtime = data.runtimes.find((entry) => entry.runtime_id === runtimeId);
    const projectId = 'projectId' in item && item.projectId ? item.projectId : runtime?.project_id;
    const agentId = 'agentId' in item && item.agentId ? item.agentId : runtime?.agent_id;
    const agentName =
      (agentId && data.agents.find((agent) => agent.id === agentId)?.name) ||
      runtime?.dynamic_title ||
      runtime?.title;

    const actions = planFocusAttention(
      { projectId, agentId, runtimeId },
      { currentProjectId: project.id, runtime, agentName }
    );

    for (const action of actions) {
      if (action.type === 'switch-project') {
        const next = data.projects.find((entry) => entry.id === action.projectId);
        if (next) setProject(next);
      } else if (action.type === 'open-agent-terminal') {
        open(agentTerminalSurface(action.agentId, action.title, '', action.runtimeId || ''));
      } else if (action.type === 'open-project-shell') {
        open(projectShellSurface(action.projectId, action.runtimeId, action.title));
      } else if (action.type === 'refresh-agents') {
        void data.refreshAgents(action.projectId).catch(() => undefined);
      }
    }
  };

  const handleFocusRuntime = (runtimeId: string) => {
    handleFocusAttention({ runtimeId });
  };

  // Sync dynamic titles & attention icons to workspace surfaces and browser document.title
  useEffect(() => {
    data.runtimes.forEach((r) => {
      if (r.dynamic_title) {
        const matchingAgent = agentForRuntime(r, data.agents);
        const surfaceId = terminalSurfaceIDForRuntime(r);
        if (matchingAgent && surfaceId) workspace.updateSurface(surfaceId, { title: r.dynamic_title });
      }
    });

    document.title = buildDocumentTitle(project.name, data.runtimes);
  }, [data.runtimes, data.agents, project.name, project.id]);

  return (
    <>
      <NexusShell
        project={project}
        agents={data.agents}
        runtimes={data.runtimes}
        rail={rail}
        onOpenRail={() => setRailOpen(true)}
        onOpenSurface={openKind}
        onCommand={() => setPalette(true)}
        onOpenWelcome={() => setWelcomeOpen(true)}
        onOpenProjectManager={() => openKind('projects')}
        onOpenMaestroControl={() => setMaestroControlOpen(true)}
        onSettings={() => openKind('settings')}
        onFocusRuntime={handleFocusRuntime}
        onFocusAttention={handleFocusAttention}
      >
        {shellError && <div className="nx-workspace-global-error" role="alert">{shellError}</div>}
        {renderer}
      </NexusShell>

      <CommandPalette open={palette} onClose={() => setPalette(false)} commands={commands} />

      <WelcomeModal
        open={welcomeOpen}
        onClose={() => setWelcomeOpen(false)}
        onStartTour={() => {
          setWelcomeOpen(false);
          setTour(true);
        }}
      />

      <MaestroControlModal
        open={maestroControlOpen}
        onClose={() => setMaestroControlOpen(false)}
        project={project}
        onProjectUpdated={handleProjectUpdated}
        onOpenMaestroSurface={() => openKind('maestro')}
      />

      <ProductTour
        open={tour}
        onClose={() => {
          setTour(false);
          window.localStorage.setItem(tourKey, 'true');
        }}
      />
    </>
  );
};
