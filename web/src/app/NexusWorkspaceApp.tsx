import React, { useEffect, useMemo, useState } from 'react';
import { initSession } from '../api';
import { resolveSessionState, type BrowserSessionState } from './sessionModel';
import { setNexusCSRF, nexus } from '../nexus/api';
import { Spinner } from '../design-system';
import { ThemeProvider } from '../design-system';
import { WorkspaceProvider, useWorkspace } from '../workspace/WorkspaceProvider';
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
import { agentConfigSurface, agentTerminalSurface, projectSurface } from './surfaces';
import { agentForRuntime, terminalSurfaceIDForRuntime } from './runtimeAgentMapping';
import { resolveProjectSelection } from './projectSelection';
import { useNexusData } from './useNexusData';
import type { Agent, Project } from '../types';
import { useTranslation } from 'react-i18next';

const selectedProjectKey = 'iapro:nexus:selected-project:v1';
const tourKey = 'iapro:nexus:tour-complete:v1';

export const NexusWorkspaceApp: React.FC<{ popoutSurface?: WorkspaceSurface }> = ({ popoutSurface }) => {
  const { t } = useTranslation();
  const [sessionState, setSessionState] = useState<BrowserSessionState>('loading');

  useEffect(() => {
    initSession().then((session) => {
      if (session.csrf_token) setNexusCSRF(session.csrf_token);
      setSessionState(resolveSessionState(session));
    });
  }, []);

  if (sessionState === 'loading') {
    return (
      <div className="nx-app-loading">
        <Spinner label={t('app.starting')} />
      </div>
    );
  }

  if (sessionState === 'unauthenticated') {
    return (
      <div className="nx-app-loading" role="alert">
        <div>
          <strong>Nexus session is not authenticated.</strong>
          <p>Open Nexus again from the local launcher/bootstrap URL to establish a new browser session.</p>
        </div>
      </div>
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
    >
      <WorkspaceCoordinator
        project={selected}
        setProject={(project) => setSelectedId(project.id)}
        data={data}
        popout={Boolean(popoutSurface)}
      />
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

  const open = (surface: WorkspaceSurface) => workspace.open(surface);
  const openKind = (kind: string) => open(projectSurface(project.id, kind as any));
  const terminal = (agent: Agent) => open(agentTerminalSurface(agent.id, agent.name));
  const config = (agent: Agent) => open(agentConfigSurface(agent.id, agent.name));

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
      { id: 'work', label: t('commands.open', { name: t('nav.work') }), group: t('commands.project'), keywords: ['prompt', 'intelligence', 'goal'], run: () => openKind('work') },
      { id: 'plan', label: t('commands.open', { name: t('nav.missions') }), group: t('commands.project'), keywords: ['mission', 'workplan', 'tasks'], run: () => openKind('missions') },
      { id: 'agents', label: t('commands.open', { name: t('nav.agents') }), group: t('commands.project'), keywords: ['fleet', 'workers'], run: () => openKind('agents') },
      { id: 'resources', label: t('commands.open', { name: t('nav.resources') }), group: 'Nexus', keywords: ['quota', 'provider', 'accounts'], run: () => openKind('resources') },
      { id: 'maestro', label: t('commands.open', { name: 'Maestro' }), group: 'Nexus', keywords: ['skills', 'process', 'verification'], run: () => openKind('maestro') },
      { id: 'maestro-control', label: t('maestroControl.title'), group: 'Nexus', keywords: ['assist', 'autonomous', 'mode'], run: () => setMaestroControlOpen(true) },
      { id: 'project-manager', label: t('projectManager.title'), group: t('commands.project'), keywords: ['switch', 'create', 'workspace'], run: () => openKind('projects') },
      { id: 'sessions', label: t('commands.open', { name: t('nav.sessions') }), group: t('commands.project'), keywords: ['resume', 'continuity'], run: () => openKind('sessions') },
      { id: 'settings', label: t('commands.open', { name: t('nav.settings') }), group: 'Nexus', keywords: ['theme', 'accessibility'], run: () => openKind('settings') },
      { id: 'runtime', label: t('commands.open', { name: t('nav.runtimes') }), group: t('commands.advanced'), keywords: ['runtime', 'legacy'], run: () => openKind('legacy-runtimes') },
      { id: 'providers', label: t('commands.open', { name: t('nav.providers') }), group: t('commands.advanced'), keywords: ['provider'], run: () => openKind('legacy-providers') },
      ...data.agents.flatMap((agent) => [
        { id: `terminal-${agent.id}`, label: t('commands.open', { name: `${agent.name} terminal` }), group: t('nav.agents'), keywords: ['terminal', agent.role], run: () => terminal(agent) },
        { id: `config-${agent.id}`, label: t('commands.configure', { name: agent.name }), group: t('nav.agents'), keywords: ['settings', agent.role], run: () => config(agent) },
      ]),
      { id: 'welcome', label: t('welcome.title'), group: t('commands.help'), keywords: ['guide', 'help', 'onboarding'], run: () => setWelcomeOpen(true) },
      { id: 'tour', label: t('commands.tour'), group: t('commands.help'), keywords: ['help', 'onboarding'], run: () => setTour(true) },
    ],
    [project.id, data.agents, t, i18n.language]
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

  const handleFocusRuntime = (runtimeId: string) => {
    const runtime = data.runtimes.find((item) => item.runtime_id === runtimeId);
    const matchingAgent = runtime ? agentForRuntime(runtime, data.agents) : undefined;
    if (matchingAgent) terminal(matchingAgent);
    else openKind('legacy-runtimes');
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

    // Update document title if runtimes need attention
    const needingAttention = data.runtimes.filter(
      (r) => r.attention_reason === 'QUESTION' || r.attention_reason === 'APPROVAL'
    );
    if (needingAttention.length > 0) {
      document.title = `(${needingAttention.length} ❓) Nexus | ${project.name} - Atenção Necessária`;
    } else {
      document.title = `Nexus | ${project.name}`;
    }
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
      >
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
