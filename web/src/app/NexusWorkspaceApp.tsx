import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { api, initSession, rotateSession, type BrowserSession } from '../api';
import { setNexusCSRF, nexus } from '../nexus/api';
import { Spinner } from '../design-system';
import { ThemeProvider } from '../design-system';
import { NexusSplashScreen } from './NexusSplashScreen';
import { WorkspaceProvider, useWorkspace } from '../workspace/WorkspaceProvider';
import { WorkspacePresentationProvider, useWorkspacePresentation } from '../workspace/WorkspacePresentationProvider';
import { PtyLiveChromeProvider } from '../workspace/PtyLiveChromeContext';
import { WorkspaceRenderer } from '../workspace/WorkspaceRenderer';
import { createWorkspace, isSurfaceMatch, listStacks, listSurfaces, surfaceViewId, type WorkspaceSurface } from '../workspace/model';
import { serializeWorkspace } from '../workspace/state';
import { ProjectRail } from '../features/projects/ProjectRail';
import { ProjectHub } from '../features/projects/ProjectHub';
import { CommandPalette } from './commands/CommandPalette';
import type { NexusCommand } from './commands/registry';
import { ProductTour } from './tour/ProductTour';
import { WelcomeModal } from './modals/WelcomeModal';
import { MaestroControlModal } from './modals/MaestroControlModal';
import { NewAgentModal } from '../features/agents/NewAgentModal';
import { DirectSessionLauncher, type DirectSessionRequest } from '../features/work/DirectSessionLauncher';
import { NexusShell } from './NexusShell';
import { WorkspaceSurfaceHost } from './WorkspaceSurfaceHost';
import { agentConfigSurface, agentTerminalSurface, flowRunSurface, projectShellSurface, projectSurface } from './surfaces';
import { attentionFingerprintOf, buildDocumentTitle, isHonestNeedsUser } from './documentTitle';
import { planFocusAttention, type RadarRuntimeItem } from './attentionRadarModel';
import { resolveProjectSelection } from './projectSelection';
import { useNexusData } from './useNexusData';
import type { Agent, MissionRun, Project } from '../types';
import { useTranslation } from 'react-i18next';
import { shouldMarkUnread, surfaceTitleFromAgent } from '../workspace/surfaceAttention';
import { KeyboardShortcutRegistry } from '../keyboard/KeyboardShortcutRegistry';
import { pushNotifications } from '../notifications/PushNotificationManager';
import {
  loadNotificationPrefs,
  playAttentionSound,
} from '../notifications/notificationPrefs';
import { formatAttentionPushBody } from '../notifications/attentionPushCopy';
import { isPtyAttentionFocused } from '../notifications/attentionDelivery';
import { TerminalActionDialog } from '../nexus/TerminalActionDialog';

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
    return <NexusSplashScreen stage="starting" />;
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
          background: 'var(--nx-bg)',
          color: 'var(--nx-text)',
          textAlign: 'center',
        }}>
          <div style={{
            maxWidth: '480px',
            background: 'var(--nx-surface)',
            border: '1px solid var(--nx-border)',
            borderRadius: 'var(--nx-radius-lg)',
            padding: '32px',
            boxShadow: '0 20px 40px rgba(0,0,0,0.3)',
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: '12px',
          }}>
            <span className="nx-brand-mark nx-brand-mark--hero">
              <img src="./nexus-icon.png" alt="Nexus" className="nx-brand-mark__img" />
            </span>
            <h2 style={{ fontSize: '18px', fontWeight: 700, margin: '6px 0 0', color: 'var(--nx-text)' }}>
              {t('auth.sessionExpired')}
            </h2>
            <p style={{ fontSize: '12.5px', color: 'var(--nx-muted)', lineHeight: 1.6, margin: '0 0 12px' }}>
              {t('auth.sessionExpiredDesc')}
            </p>
            <button
              type="button"
              className="nx-button"
              data-tone="brand"
              onClick={() => window.location.reload()}
            >
              {t('auth.reloadApp')}
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
    return <NexusSplashScreen stage="loading" />;
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
      <WorkspacePresentationProvider projectId={selected.id}>
        <PtyLiveChromeProvider>
        <WorkspaceCoordinator
          project={selected}
          setProject={(project) => setSelectedId(project.id)}
          data={data}
          popout={Boolean(popoutSurface)}
        />
        </PtyLiveChromeProvider>
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
  const presentation = useWorkspacePresentation();
  const [railOpen, setRailOpen] = useState(false);
  const [palette, setPalette] = useState(false);
  const [welcomeOpen, setWelcomeOpen] = useState(false);
  const [maestroControlOpen, setMaestroControlOpen] = useState(false);
  const [newAgentOpen, setNewAgentOpen] = useState(false);
  const [directSession, setDirectSession] = useState<DirectSessionRequest | null>(null);
  const [tour, setTour] = useState(false);
  const [shellError, setShellError] = useState('');
  const [flowRuns, setFlowRuns] = useState<MissionRun[]>([]);
  const [closeTarget, setCloseTarget] = useState<WorkspaceSurface | null>(null);
  const shellInFlight = useRef(false);

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
  const requestCloseSurface = (surface: WorkspaceSurface) => setCloseTarget(surface);
  const closeConfirmed = async (stopRuntime: boolean) => {
    if (!closeTarget) return;
    const target = closeTarget;
    const runtimeId = target.data?.runtimeId;
    const agentId = target.data?.agentId;
    setCloseTarget(null);
    workspace.close(target.id);
    if (!stopRuntime) return;
    void (async () => {
      try {
        if (target.type === 'project-shell' && runtimeId) {
          await api.stopRuntime(runtimeId);
        } else if (target.type === 'terminal' && agentId) {
          await nexus.stopAgent(agentId);
        }
      } finally {
        await data.refreshGlobal().catch(() => undefined);
      }
    })();
  };
  const openKind = (kind: string) => open(projectSurface(project.id, kind as any));
  const openTerminals = () => open(projectSurface(project.id, 'terminals'));
  const openNewAISession = () => setDirectSession({ mode: 'direct', prompt: '' });
  const terminal = (agent: Agent) => {
    open(agentTerminalSurface(agent.id, agent.name));
    openTerminals();
  };
  const config = (agent: Agent) => open(agentConfigSurface(agent.id, agent.name));
  const shell = useCallback(async () => {
    if (shellInFlight.current) return;
    shellInFlight.current = true;
    setShellError('');
    try {
      const result = await nexus.startProjectShell(project.id);
      workspace.open(
        projectShellSurface(project.id, result.runtime.runtime_id, result.runtime.title || 'Terminal')
      );
      workspace.open(projectSurface(project.id, 'terminals'));
      await data.refreshGlobal().catch(() => undefined);
    } catch (error) {
      setShellError(error instanceof Error ? error.message : String(error));
    } finally {
      shellInFlight.current = false;
    }
  }, [project.id, workspace, data]);

  // Ensure pinned product tabs exist without stealing Overview focus.
  useEffect(() => {
    workspace.ensure(projectSurface(project.id, 'terminals'));
    workspace.ensure(projectSurface(project.id, 'work'));
  }, [project.id, workspace]);

  useEffect(() => {
    const handleNewAgentEvent = () => setNewAgentOpen(true);
    const handleProjectShellEvent = () => {
      void shell();
    };
    const handleNewAISession = () => setDirectSession({ mode: 'direct', prompt: '' });
    window.addEventListener('nexus:new-agent', handleNewAgentEvent);
    window.addEventListener('nexus:new-ai-session', handleNewAISession);
    window.addEventListener('nexus:project-shell', handleProjectShellEvent);
    return () => {
      window.removeEventListener('nexus:new-agent', handleNewAgentEvent);
      window.removeEventListener('nexus:new-ai-session', handleNewAISession);
      window.removeEventListener('nexus:project-shell', handleProjectShellEvent);
    };
  }, [shell]);

  const commands = useMemo<NexusCommand[]>(
    () => [
      { id: 'projects', label: t('commands.open', { name: t('projectManager.desktopsTitle') }), group: t('commands.project'), keywords: ['workspace', 'desktops', 'hub'], run: () => openKind('projects') },
      { id: 'overview', label: t('commands.open', { name: t('nav.overview') }), group: t('commands.project'), keywords: ['home'], run: () => openKind('overview') },
      { id: 'terminals', label: t('commands.open', { name: t('nav.terminals') }), group: t('commands.project'), keywords: ['pty', 'shell'], run: () => openKind('terminals') },
      { id: 'new-ai-session', label: 'New AI Session', group: t('commands.project'), keywords: ['agent', 'session', 'direct', 'create', 'terminal'], run: openNewAISession },
      { id: 'project-shell', label: 'New Terminal', group: t('commands.project'), keywords: ['shell', 'terminal', 'bash', 'powershell'], run: () => void shell() },
      { id: 'agents', label: t('commands.open', { name: t('nav.agents') }), group: t('commands.project'), keywords: ['fleet', 'workers', 'terminals'], run: () => openKind('agents') },
      ...data.agents.flatMap((agent) => [
        { id: `terminal-${agent.id}`, label: t('commands.open', { name: `${agent.name} terminal` }), group: t('nav.agents'), keywords: ['terminal', agent.role], run: () => terminal(agent) },
        { id: `config-${agent.id}`, label: t('commands.configure', { name: agent.name }), group: t('nav.agents'), keywords: ['settings', agent.role], run: () => config(agent) },
      ]),
      { id: 'work', label: 'Open Composer', group: t('commands.project'), keywords: ['composer', 'prompt', 'goal', 'plan'], run: () => openKind('work') },
      { id: 'plan', label: 'Open Flow Runs history', group: t('commands.project'), keywords: ['flow', 'mission', 'history', 'runs'], run: () => openKind('missions') },
      { id: 'resources', label: t('commands.open', { name: t('nav.resources') }), group: 'Nexus', keywords: ['quota', 'provider', 'accounts'], run: () => openKind('resources') },
      { id: 'maestro-control', label: t('maestroControl.title'), group: 'Nexus', keywords: ['skills', 'gates', 'update', 'library'], run: () => setMaestroControlOpen(true) },
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
      ...([
        { id: 'preset-automatic', label: 'Layout: Automatic (Smart)', group: 'Layout Presets', preset: 'automatic' as const },
        { id: 'preset-terminal-focus', label: 'Layout: Terminal Focus (68/32)', group: 'Layout Presets', preset: 'terminal-focus' as const },
        { id: 'preset-two-columns', label: 'Layout: Two Columns (50/50)', group: 'Layout Presets', preset: 'two-columns' as const },
        { id: 'preset-three-columns', label: 'Layout: Three Columns (33/33/33)', group: 'Layout Presets', preset: 'three-columns' as const },
        { id: 'preset-terminal-chat', label: 'Layout: Terminal + Chat', group: 'Layout Presets', preset: 'terminal-chat' as const },
        { id: 'preset-terminal-flow', label: 'Layout: Terminal + Flow', group: 'Layout Presets', preset: 'terminal-flow' as const },
        { id: 'preset-focus-mode', label: 'Layout: Focus Mode (Active window)', group: 'Layout Presets', preset: 'focus-mode' as const },
        { id: 'preset-restore-default', label: 'Layout: Restore Default', group: 'Layout Presets', preset: 'restore-default' as const },
      ].map((p) => ({
        id: p.id,
        label: p.label,
        group: p.group,
        keywords: ['layout', 'preset', 'window', 'arrange', 'grid', 'columns', 'tile'],
        run: () => {
          presentation.setMode('MOSAIC');
          presentation.rearrangePreset(p.preset);
        },
      }))),
      { id: 'welcome', label: t('welcome.title'), group: t('commands.help'), keywords: ['guide', 'help', 'onboarding'], run: () => setWelcomeOpen(true) },
      { id: 'tour', label: t('commands.tour'), group: t('commands.help'), keywords: ['help', 'onboarding'], run: () => setTour(true) },
    ],
    [project.id, data.agents, flowRuns, t, i18n.language, presentation]
  );

  // Scoped Keyboard Shortcuts via KeyboardShortcutRegistry
  useEffect(() => {
    const registry = KeyboardShortcutRegistry.getInstance();
    const unregisterK = registry.register({
      id: 'open-palette',
      key: 'k',
      ctrlOrMeta: true,
      scope: 'global',
      description: 'Open Command Palette',
      preventDefault: true,
      action: () => setPalette((value) => !value),
    });
    const unregisterShiftP = registry.register({
      id: 'open-palette-shift',
      key: 'p',
      ctrlOrMeta: true,
      shift: true,
      scope: 'global',
      description: 'Open Command Palette',
      preventDefault: true,
      action: () => setPalette(true),
    });
    const unregisterP = registry.register({
      id: 'open-projects',
      key: 'p',
      ctrlOrMeta: true,
      shift: false,
      scope: 'global',
      description: 'Open Project Hub',
      preventDefault: true,
      action: () => openKind('projects'),
    });

    return () => {
      unregisterK();
      unregisterShiftP();
      unregisterP();
    };
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
          closeSurface={workspace.close}
          onTour={() => setWelcomeOpen(true)}
        />
      )}
      onRequestClose={requestCloseSurface}
      createActions={{
        onNewAgent: () => setNewAgentOpen(true),
        onNewAISession: openNewAISession,
        onProjectShell: () => { void shell(); },
      }}
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
      agents={data.agents}
      onOpenAgent={(agent) => terminal(agent)}
      onNewAgent={() => setNewAgentOpen(true)}
      onNewAISession={openNewAISession}
      onProjectShell={() => { void shell(); }}
      onOpenGlobal={(kind) => {
        if (kind === 'overview') openKind('overview');
        else openKind(kind);
      }}
    />
  );

  const handleFocusAttention = (item: RadarRuntimeItem | { runtimeId: string; projectId?: string; agentId?: string }) => {
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

    const switchAction = actions.find((action) => action.type === 'switch-project');
    const openActions = actions.filter((action) => action.type !== 'switch-project');

    const runOpen = () => {
      for (const action of openActions) {
        if (action.type === 'open-agent-terminal') {
          open(agentTerminalSurface(action.agentId, action.title, '', action.runtimeId || ''));
          open(projectSurface(project.id, 'terminals'));
        } else if (action.type === 'open-project-shell') {
          open(projectShellSurface(action.projectId, action.runtimeId, action.title));
          open(projectSurface(action.projectId, 'terminals'));
        } else if (action.type === 'refresh-agents') {
          void data.refreshAgents(action.projectId).catch(() => undefined);
        }
      }
    };

    if (switchAction && switchAction.type === 'switch-project') {
      const next = data.projects.find((entry) => entry.id === switchAction.projectId);
      if (next) {
        setProject(next);
        window.setTimeout(runOpen, 0);
        return;
      }
    }
    runOpen();
  };

  const handleFocusRuntime = (runtimeId: string) => {
    handleFocusAttention({ runtimeId });
  };

  const notifiedFingerprints = useRef<Set<string>>(new Set());
  const prefsRef = useRef(loadNotificationPrefs());

  useEffect(() => {
    const refreshPrefs = () => {
      prefsRef.current = loadNotificationPrefs();
    };
    window.addEventListener('nexus:notification-prefs', refreshPrefs);
    return () => window.removeEventListener('nexus:notification-prefs', refreshPrefs);
  }, []);

  // Keep agent name stable, append short status suffix, and sync attention markers.
  useEffect(() => {
    const allSurfaces = listSurfaces(workspace.model.root);
    const stacks = listStacks(workspace.model.root);
    const focusedIds = new Set(stacks.map((stack) => stack.activeId));
    const terminalsProductId = projectSurface(project.id, 'terminals').id;
    const activePtyViewId = presentation.state.activePtyViewId;

    data.agents.forEach((agent) => {
      const surfaceId = `agent:${agent.id}:terminal`;
      const runtime = data.runtimes.find((r) => r.agent_id === agent.id);
      const surface = allSurfaces.find((s) => isSurfaceMatch(s, surfaceId));
      if (!surface) return;

      const next = surfaceTitleFromAgent(agent.name, runtime);
      const focused = isPtyAttentionFocused({
        terminalsProductSurfaceId: terminalsProductId,
        agentViewId: surfaceViewId(surface),
        stackActiveIds: focusedIds,
        activePtyViewId,
      });
      const previousFingerprint = surface.data?.attentionFingerprint || '';
      const previousUnread = surface.data?.unreadAttention === 'true';
      let unread = previousUnread;
      if (focused) {
        unread = false;
      } else if (
        shouldMarkUnread({
          previousFingerprint,
          nextFingerprint: next.fingerprint,
          hasAttention: next.hasAttention,
          surfaceFocused: focused,
          attentionKind: next.attentionKind,
        })
      ) {
        unread = true;
      } else if (!next.hasAttention) {
        unread = false;
      }

      const providerLabel = runtime
        ? `${runtime.provider_id || runtime.provider || 'claude'}${
            runtime.profile_id && runtime.profile_id !== 'default' ? `:${runtime.profile_id}` : ''
          }`
        : '';

      const needsUpdate =
        surface.title !== next.title ||
        surface.data?.hasAttention !== (next.hasAttention ? 'true' : 'false') ||
        surface.data?.unreadAttention !== (unread ? 'true' : 'false') ||
        surface.data?.attentionKind !== next.attentionKind ||
        surface.data?.statusSuffix !== next.statusSuffix ||
        surface.data?.dynamicTitle !== next.dynamicTitle ||
        surface.data?.attentionFingerprint !== next.fingerprint ||
        surface.data?.providerLabel !== providerLabel ||
        surface.data?.agentName !== agent.name;

      if (needsUpdate) {
        workspace.updateSurface(surface.id, {
          title: next.title,
          data: {
            ...surface.data,
            agentName: agent.name,
            hasAttention: next.hasAttention ? 'true' : 'false',
            unreadAttention: unread ? 'true' : 'false',
            attentionKind: next.attentionKind,
            statusSuffix: next.statusSuffix,
            dynamicTitle: next.dynamicTitle,
            attentionFingerprint: next.fingerprint,
            providerLabel,
          },
        });
      }
    });

    document.title = buildDocumentTitle(project.name, data.runtimes);
  }, [data.runtimes, data.agents, project.name, project.id, workspace.model.root, presentation.state.activePtyViewId]);

  // Attention watcher for the focused project only (radar remains global).
  useEffect(() => {
    const prefs = prefsRef.current;
    if (!prefs.notificationsEnabled && !prefs.soundEnabled) return;

    const stacks = listStacks(workspace.model.root);
    const focusedIds = new Set(stacks.map((stack) => stack.activeId));
    const allSurfaces = listSurfaces(workspace.model.root);
    const terminalsProductId = projectSurface(project.id, 'terminals').id;
    const activePtyViewId = presentation.state.activePtyViewId;

    for (const runtime of data.runtimes) {
      if (runtime.project_id && runtime.project_id !== project.id) continue;
      const provider = (runtime.provider_id || runtime.provider || '').toLowerCase();
      if (provider === 'shell') continue;

      const reason = runtime.attention_reason;
      if (reason !== 'QUESTION' && reason !== 'APPROVAL' && reason !== 'TASK_COMPLETED' && reason !== 'ERROR') {
        continue;
      }
      if (reason === 'QUESTION' || reason === 'APPROVAL') {
        if (!isHonestNeedsUser(runtime)) continue;
      }
      const fingerprint = attentionFingerprintOf(runtime);
      if (!fingerprint || notifiedFingerprints.current.has(fingerprint)) continue;

      const agentSurface = allSurfaces.find(
        (surface) => surface.type === 'terminal' && surface.data?.agentId && surface.data.agentId === runtime.agent_id
      );
      const focused = agentSurface
        ? isPtyAttentionFocused({
            terminalsProductSurfaceId: terminalsProductId,
            agentViewId: surfaceViewId(agentSurface),
            stackActiveIds: focusedIds,
            activePtyViewId,
          })
        : false;
      if (focused) {
        notifiedFingerprints.current.add(fingerprint);
        continue;
      }

      notifiedFingerprints.current.add(fingerprint);

      const agentName =
        data.agents.find((agent) => agent.id === runtime.agent_id)?.name ||
        runtime.dynamic_title ||
        runtime.title ||
        undefined;
      const body = formatAttentionPushBody({
        reason,
        context: runtime.attention_context || runtime.last_task_summary || '',
        promptKind: runtime.prompt_kind,
        agentName,
        projectName: runtime.project_name || project.name,
        rich: false,
      });
      if (!body) continue;

      if (prefs.soundEnabled) playAttentionSound();

      if (prefs.notificationsEnabled) {
        pushNotifications.sendPush({
          runtimeId: runtime.runtime_id,
          projectName: runtime.project_name || project.name,
          agentName,
          reason,
          context: runtime.attention_context || runtime.last_task_summary || body,
          dynamicTitle: runtime.dynamic_title,
          fingerprint,
          promptKind: runtime.prompt_kind,
          onClick: () =>
            handleFocusAttention({
              runtimeId: runtime.runtime_id,
              projectId: runtime.project_id,
              agentId: runtime.agent_id,
            }),
        });
      }
    }
  }, [data.runtimes, data.agents, project.id, project.name, workspace.model.root, presentation.state.activePtyViewId]);

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
        onSettings={() => openKind('settings')}
        onNewAgent={() => setNewAgentOpen(true)}
        onNewAISession={openNewAISession}
        onProjectShell={() => { void shell(); }}
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
      />

      <NewAgentModal
        open={newAgentOpen}
        onClose={() => setNewAgentOpen(false)}
        project={project}
        onCreated={(created) => {
          data.setAgents((cur) => [created, ...cur]);
          terminal(created);
        }}
      />

      <DirectSessionLauncher
        open={!!directSession}
        project={project}
        request={directSession}
        onClose={() => setDirectSession(null)}
        refreshAgents={() => data.refreshAgents(project.id)}
        onStarted={(created, prompt) => {
          open(agentTerminalSurface(created.id, created.name, prompt));
          openTerminals();
        }}
      />

      <ProductTour
        open={tour}
        onClose={() => {
          setTour(false);
          window.localStorage.setItem(tourKey, 'true');
        }}
      />
      {closeTarget && (
        <TerminalActionDialog
          close
          shell={closeTarget.type === 'project-shell'}
          onCancel={() => setCloseTarget(null)}
          onCloseTab={() => void closeConfirmed(false)}
          onStopRuntime={() => void closeConfirmed(true)}
        />
      )}
    </>
  );
};
