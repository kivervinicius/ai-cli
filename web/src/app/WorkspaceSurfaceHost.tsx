import React, { useMemo, useState } from 'react';
import { Gauge, Network } from 'lucide-react';
import { Button, Card, EmptyState } from '../design-system';
import { Dashboard } from '../components/Dashboard';
import { ProvidersView } from '../components/ProvidersView';
import { EventsView } from '../components/EventsView';
import { StartModal } from '../components/StartModal';
import { HandoffModal } from '../components/HandoffModal';
import { ContinueModal } from '../components/ContinueModal';
import { TerminalPane } from '../components/TerminalPane';
import { AgentTerminal } from '../nexus/AgentTerminal';
import { recoverOrStartAgent } from '../nexus/agentRecover';
import { ResourcePicker } from '../nexus/ResourcePicker';
import { nexus } from '../nexus/api';
import { api } from '../api';
import type { Agent, EventRecord, ProfileInfo, Project, ProviderInfo, RuntimeSession, Workspace } from '../types';
import type { WorkspaceSurface } from '../workspace/model';
import { ProjectManagerSurface } from '../features/projects/ProjectManagerSurface';
import { ProjectOverviewSurface } from '../features/overview/ProjectOverviewSurface';
import { WorkSurface } from '../features/work/WorkSurface';
import { FlowRunSurface } from '../features/work/FlowRunSurface';
import { FlowRunsHistorySurface } from '../features/work/FlowRunsHistorySurface';
import { AgentsSurface } from '../features/agents/AgentsSurface';
import { AgentConfigurationSurface } from '../features/agents/AgentConfigurationSurface';
import { SessionsSurface } from '../features/sessions/SessionsSurface';
import { SettingsSurface } from '../features/settings/SettingsSurface';
import { ProjectShellSurface } from '../features/shell/ProjectShellSurface';
import { useTranslation } from 'react-i18next';
import { agentConfigSurface, agentTerminalSurface, flowRunSurface, projectSurface } from './surfaces';
import { useWorkspacePresentation } from '../workspace/WorkspacePresentationProvider';

export const WorkspaceSurfaceHost: React.FC<{
  surface: WorkspaceSurface;
  project: Project;
  projects: Project[];
  agents: Agent[];
  workspaces: Workspace[];
  runtimes: RuntimeSession[];
  providers: ProviderInfo[];
  profiles: ProfileInfo[];
  events: EventRecord[];
  refreshAgents: () => Promise<void>;
  refreshGlobal: () => Promise<void>;
  onSelectProject: (project: Project) => void;
  onProjectCreated: (project: Project) => void;
  onProjectUpdated: (project: Project) => void;
  openSurface: (surface: WorkspaceSurface) => void;
  closeSurface: (surfaceId: string) => void;
  onTour: () => void;
}> = ({
  surface,
  project,
  projects,
  agents,
  workspaces,
  runtimes,
  providers,
  profiles,
  events,
  refreshAgents,
  refreshGlobal,
  onSelectProject,
  onProjectCreated,
  onProjectUpdated,
  openSurface,
  closeSurface,
  onTour,
}) => {
  const { t } = useTranslation();
  const presentation = useWorkspacePresentation();
  const terminalChrome = presentation.state.mode === 'DESKTOP' || presentation.state.mode === 'MOSAIC' ? 'window' : 'full';
  const agent = useMemo(
    () => agents.find((item) => item.id === surface.data?.agentId),
    [agents, surface.data?.agentId]
  );
  const [showStart, setShowStart] = useState(false);
  const [handoff, setHandoff] = useState<RuntimeSession | null>(null);
  const [cont, setCont] = useState<RuntimeSession | null>(null);

  const open = (kind: string) =>
    openSurface(projectSurface(project.id, kind as Parameters<typeof projectSurface>[1]));
  const terminal = (target: Agent, initialPrompt = '', runtimeId = '') => {
    openSurface(agentTerminalSurface(target.id, target.name, initialPrompt, runtimeId));
    openSurface(projectSurface(project.id, 'terminals'));
  };
  const config = (target: Agent) => openSurface(agentConfigSurface(target.id, target.name));

  if (surface.type === 'projects') {
    return (
      <ProjectManagerSurface
        projects={projects}
        selectedProject={project}
        agents={agents}
        onSelectProject={onSelectProject}
        onProjectCreated={onProjectCreated}
        onProjectUpdated={onProjectUpdated}
        onOpenOverview={() => open('overview')}
      />
    );
  }

  if (surface.type === 'overview')
    return (
      <ProjectOverviewSurface
        project={project}
        agents={agents}
        onOpenAgent={(agent, runtimeId) => terminal(agent, '', runtimeId || '')}
        refreshAgents={refreshAgents}
        onNewAgent={() => window.dispatchEvent(new CustomEvent('nexus:new-agent'))}
        onConfigureAgent={config}
        onNewAISession={() => window.dispatchEvent(new CustomEvent('nexus:new-ai-session'))}
        onProjectShell={() => window.dispatchEvent(new CustomEvent('nexus:project-shell'))}
        onOpenComposer={() => open('work')}
        onOpenFlow={undefined}
      />
    );

  if (surface.type === 'work')
    return (
      <WorkSurface
        project={project}
        agents={agents}
        onDirect={terminal}
        onFlowRun={(run) => openSurface(flowRunSurface(run.id, `Flow Run · ${(run.id || '').slice(-6)}`))}
      />
    );

  if (surface.type === 'flow-run')
    return surface.data?.runId ? (
      <FlowRunSurface runId={surface.data.runId} project={project} agents={agents} onOpenAgent={terminal} />
    ) : (
      <EmptyState title="Flow Run unavailable" />
    );

  if (surface.type === 'agents')
    return (
      <AgentsSurface
        project={project}
        agents={agents}
        refresh={refreshAgents}
        onTerminal={terminal}
        onConfigure={config}
        onRemoved={(agentId) => closeSurface(`agent:${agentId}:terminal`)}
      />
    );

  if (surface.type === 'agent-config')
    return <AgentConfigurationSurface agent={agent} onApplied={refreshAgents} />;

  if (surface.type === 'project-shell')
    return surface.data?.runtimeId ? <ProjectShellSurface runtimeId={surface.data.runtimeId} title={surface.title} onRuntimeChanged={refreshGlobal} /> : <EmptyState title="Project shell unavailable" />;

  if (surface.type === 'terminal') {
    // A recovered Agent receives a new runtime generation. Never keep using
    // the surface's historical runtime_id; the agent-scoped endpoint can
    // resolve the current generation when the list is briefly stale/empty.
    const runtime = runtimes.find((r) => r.agent_id === surface.data?.agentId);
    return surface.data?.agentId ? (
      <div className="nx-agent-terminal-surface">
        <AgentTerminal
          agentId={surface.data.agentId}
          runtimeId={runtime?.runtime_id}
          initialPrompt={surface.data.initialPrompt}
          provider={runtime?.provider_id || runtime?.provider || 'claude'}
          profile={runtime?.profile_id || runtime?.profile || 'default'}
          agentName={agent?.name || surface.title}
          chrome={terminalChrome}
          onRecover={async () => {
            if (!surface.data?.agentId) return;
            try {
              const result = await recoverOrStartAgent(surface.data.agentId);
              await refreshAgents();
              await refreshGlobal();
              return result?.runtime;
            } catch (err) {
              throw err instanceof Error
                ? err
                : new Error(
                    'Não foi possível iniciar ou recuperar o runtime do agente. Verifique o provedor e a conta configurados.'
                  );
            }
          }}
          onRestartWithMode={async (newMode: 'Safe' | 'YOLO') => {
            if (!agent) return;
            const currentCfg = await nexus.getAgentConfig(agent.id);
            const configuredArgs = Array.isArray(currentCfg.config.options?.extra_args)
              ? currentCfg.config.options.extra_args
              : [];
            // `mode` is a Nexus-level contract. Only pass a native flag when
            // the selected CLI actually supports it: Codex has no `--plan`
            // flag, so forwarding that canonical alias makes it abort before
            // starting. The persisted mode still keeps the UI state explicit.
            const extraArgs = configuredArgs.filter((arg) => arg !== '--plan' && arg !== '--yolo' && arg !== '-y');
            if (newMode === 'YOLO') extraArgs.push('--yolo');
            await nexus.applyAgentConfig(agent.id, {
              ...currentCfg.config,
              options: {
                ...currentCfg.config.options,
                mode: newMode,
                extra_args: extraArgs,
              },
            });
            const result = await recoverOrStartAgent(agent.id);
            await refreshAgents();
            await refreshGlobal();
            return result.runtime;
          }}
          onClose={async (stopRuntime) => {
            if (stopRuntime) {
              const currentRuntime = runtimes.find((item) => item.agent_id === surface.data?.agentId);
              if (currentRuntime?.runtime_id) await api.stopRuntime(currentRuntime.runtime_id);
              else if (surface.data?.agentId) await nexus.stopAgent(surface.data.agentId);
              await refreshGlobal();
            }
            closeSurface(surface.id);
          }}
          onDelete={async () => {
            const agentId = surface.data?.agentId;
            if (!agentId) return;
            try {
              await nexus.stopAgent(agentId);
            } catch {
              /* may already be dead / unreachable */
            }
            await nexus.deleteAgent(agentId);
            await refreshAgents();
            await refreshGlobal();
            closeSurface(surface.id);
          }}
        />
      </div>
    ) : (
      <EmptyState title={t('surfaces.terminalUnavailable')} />
    );
  }

  if (surface.type === 'resources')
    return (
      <div className="nx-surface-scroll">
        <div className="nx-page-header">
          <div>
            <span className="nx-eyebrow">{t('surfaces.resourcesEyebrow')}</span>
            <h1>{t('surfaces.resourcesTitle')}</h1>
            <p>{t('surfaces.resourcesIntro')}</p>
          </div>
        </div>
        <div className="nx-resource-layout">
          <Card className="nx-resource-card">
            <ResourcePicker />
          </Card>
          <Card className="nx-resource-card">
            <div className="nx-feature-heading">
              <Gauge size={18} />
              <div>
                <strong>{t('surfaces.allocation')}</strong>
                <small>{project.resource_policy || 'BALANCED'}</small>
              </div>
            </div>
            <p className="nx-muted-copy">{t('surfaces.allocationBody')}</p>
          </Card>
        </div>
      </div>
    );

  if (surface.type === 'maestro')
    return (
      <div className="nx-surface-scroll">
        <div className="nx-page-header">
          <div>
            <span className="nx-eyebrow">{t('surfaces.maestroEyebrow')}</span>
            <h1>Maestro</h1>
            <p>{t('maestroControl.legacySurface')}</p>
          </div>
        </div>
      </div>
    );

  if (surface.type === 'missions')
    return (
      <FlowRunsHistorySurface
        project={project}
        onOpenRun={(run) => openSurface(flowRunSurface(run.id, `Flow Run · ${run.id.slice(-6)}`))}
        onOpenComposer={() => open('work')}
      />
    );

  if (surface.type === 'sessions') return <SessionsSurface agents={agents} />;

  if (surface.type === 'settings') return <SettingsSurface onTour={onTour} />;

  if (surface.type === 'legacy-providers')
    return (
      <div className="nx-legacy-surface">
        <ProvidersView providers={providers} />
      </div>
    );

  if (surface.type === 'legacy-events')
    return (
      <div className="nx-legacy-surface">
        <EventsView events={events} />
      </div>
    );

  if (surface.type === 'legacy-terminal') {
    const runtime = runtimes.find((item) => item.runtime_id === surface.data?.runtimeId);
    if (!runtime) return <EmptyState title={t('surfaces.runtimeUnavailable')} />;
    return (
      <TerminalPane
        runtimeId={runtime.runtime_id}
        title={runtime.title}
        provider={runtime.provider_id || runtime.provider || 'AI'}
        profile={runtime.profile_id || runtime.profile || 'default'}
        onUpdateTitle={async (id, title) => {
          await api.updateRuntimeTitle(id, title);
          await refreshGlobal();
        }}
      />
    );
  }

  if (surface.type === 'legacy-runtimes')
    return (
      <div className="nx-legacy-surface">
        <Dashboard
          runtimes={runtimes}
          providers={providers}
          workspaces={workspaces}
          activeWorkspace={project.canonical_path}
          onOpenTerminal={(runtimeId) => {
            const runtime = runtimes.find((item) => item.runtime_id === runtimeId);
            openSurface({
              id: `runtime:${runtimeId}:terminal`,
              type: 'legacy-terminal',
              title: runtime?.title || runtimeId,
              data: { runtimeId },
            });
          }}
          onOpenStartModal={() => setShowStart(true)}
          onOpenHandoffModal={setHandoff}
          onOpenContinueModal={setCont}
          onStopRuntime={async (id) => {
            await api.stopRuntime(id);
            await refreshGlobal();
          }}
          onDeleteRuntime={async (id) => {
            await api.deleteRuntime(id);
            await refreshGlobal();
          }}
          onCleanInactive={async () => {
            await api.cleanRuntimes();
            await refreshGlobal();
          }}
        />
        {showStart && (
          <StartModal
            providers={providers}
            profiles={profiles}
            workspace={project.canonical_path}
            workspaces={workspaces}
            onClose={() => setShowStart(false)}
            onSuccess={async () => {
              setShowStart(false);
              await refreshGlobal();
            }}
          />
        )}
        {handoff && (
          <HandoffModal
            runtime={handoff}
            profiles={profiles}
            onClose={() => setHandoff(null)}
            onSuccess={async () => {
              setHandoff(null);
              await refreshGlobal();
            }}
          />
        )}
        {cont && (
          <ContinueModal
            runtime={cont}
            providers={providers}
            profiles={profiles}
            onClose={() => setCont(null)}
            onSuccess={async () => {
              setCont(null);
              await refreshGlobal();
            }}
          />
        )}
      </div>
    );

  return (
    <div className="nx-surface-center">
      <EmptyState
        icon={<Network size={22} />}
        title={t('surfaces.unavailable')}
        hint={t('surfaces.unknown', { type: surface.type })}
        action={<Button onClick={() => open('overview')}>{t('surfaces.openOverview')}</Button>}
      />
    </div>
  );
};
