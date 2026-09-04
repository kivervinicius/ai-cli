import React, { useEffect, useState } from 'react';
import {
  ArrowUpCircle,
  Bell,
  BellRing,
  ChevronDown,
  CircleHelp,
  Command,
  Menu,
  MoonStar,
  Network,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Badge, IconButton } from '../design-system';
import { LanguagePicker } from './components/LanguagePicker';
import { WorkspaceTaskbar } from '../workspace/WorkspaceTaskbar';
import { ProjectCreateActions } from '../features/projects/ProjectCreateActions';
import { nexus } from '../nexus/api';
import { AttentionIntermediationBanner } from '../components/AttentionIntermediationBanner';
import { GlobalAttentionRadar } from '../components/GlobalAttentionRadar';
import { pushNotifications } from '../notifications/PushNotificationManager';
import { InAppNotificationCenter } from '../notifications/InAppNotificationCenter';
import type { Agent, Project, RuntimeSession } from '../types';
import type { RadarRuntimeItem } from './attentionRadarModel';

export const NexusShell: React.FC<{
  project: Project;
  agents: Agent[];
  runtimes?: RuntimeSession[];
  rail: React.ReactNode;
  children: React.ReactNode;
  onOpenRail: () => void;
  onOpenSurface?: (kind: string) => void;
  onCommand: () => void;
  onOpenWelcome: () => void;
  onOpenProjectManager: () => void;
  onSettings: () => void;
  onNewAgent?: () => void;
  onNewAISession?: () => void;
  onProjectShell?: () => void;
  onFocusRuntime?: (runtimeId: string) => void;
  onFocusAttention?: (item: RadarRuntimeItem) => void;
}> = ({
  project,
  agents,
  runtimes = [],
  rail,
  children,
  onOpenRail,
  onOpenSurface,
  onCommand,
  onOpenWelcome,
  onOpenProjectManager,
  onSettings,
  onNewAgent,
  onNewAISession,
  onProjectShell,
  onFocusRuntime,
  onFocusAttention,
}) => {
  const { t } = useTranslation();
  const agentList = Array.isArray(agents) ? agents : [];
  const [agentsOpen, setAgentsOpen] = useState(false);
  const liveStates = ['STARTING', 'RUNNING', 'HANDOFF', 'WAITING', 'APPROVAL'];
  const runtimeByAgent = new Map(runtimes.filter((runtime) => runtime.agent_id).map((runtime) => [runtime.agent_id as string, runtime]));
  const working = agentList.filter((agent) => runtimeByAgent.get(agent.id)?.state === 'RUNNING' || agent.status === 'WORKING').length;
  const awaiting = agentList.filter((agent) => {
    const runtime = runtimeByAgent.get(agent.id);
    return runtime?.state === 'WAITING' || runtime?.state === 'APPROVAL' || runtime?.attention_kind === 'needs_user';
  }).length;
  const disconnected = agentList.filter((agent) => !liveStates.includes(runtimeByAgent.get(agent.id)?.state || '')).length;
  const attention = agentList.filter((agent) => {
    const runtimeState = runtimeByAgent.get(agent.id)?.state;
    return ['FAILED', 'STALE', 'RECOVERABLE', 'RATE_LIMITED'].includes(agent.status) || ['FAILED', 'STALE'].includes(runtimeState || '');
  }).length;
  const [sysInfo, setSysInfo] = useState<{
    nexus_version: string;
    maestro_version: string;
    maestro_available: boolean;
    update_available: boolean;
  } | null>(null);

  useEffect(() => {
    const load = () => {
      nexus.getSystemUpdates().then(setSysInfo).catch(() => undefined);
    };
    load();
    window.addEventListener('nexus:system-updates', load);
    return () => window.removeEventListener('nexus:system-updates', load);
  }, []);


  return (
    <div className="nx-os-shell">
      <a href="#nexus-workspace" className="nx-skip-link">
        {t('shell.skip')}
      </a>
      {rail}

      <div className="nx-os-main">
        <div className="nx-shell-chrome">
        {/* Top OS Header */}
        <header className="nx-topbar">
          <div className="nx-topbar__context">
            <IconButton className="nx-mobile-menu" label={t('shell.openProjects')} onClick={onOpenRail}>
              <Menu size={16} />
            </IconButton>

            {/* Clickable Project Switcher */}
            <button
              type="button"
              className="nx-topbar__project-btn"
              onClick={onOpenProjectManager}
              title="Open Project Manager (Ctrl+P)"
            >
              <span className="nx-project-avatar nx-project-avatar--top">
                {(project.name || 'PR').slice(0, 2).toUpperCase()}
              </span>
              <span className="nx-project-info">
                <strong>{project.name}</strong>
                <small>{project.default_branch || 'unknown'}</small>
              </span>
              <ChevronDown size={12} className="nx-project-chevron" />
            </button>

            <span className="nx-topbar__path" title={project.canonical_path} style={{ maxWidth: '28vw', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
              {project.canonical_path}
            </span>

            <ProjectCreateActions
              onNewAgent={onNewAgent}
              onNewAISession={onNewAISession}
              onProjectShell={onProjectShell}
              size="sm"
              className="nx-topbar__create"
            />

            <GlobalAttentionRadar
              runtimes={runtimes}
              currentProjectId={project.id}
              onFocus={(item) => {
                if (onFocusAttention) onFocusAttention(item);
                else if (onFocusRuntime) onFocusRuntime(item.runtimeId);
              }}
            />
          </div>

          {/* Right Status Controls */}
          <div className="nx-topbar__status" data-tour="status">
            {sysInfo?.update_available && (
              <button
                type="button"
                onClick={onSettings}
                className="nx-update-indicator"
                title={t('settings.updates')}
              >
                <ArrowUpCircle size={13} className="nx-spin-slow" />
                <span>{t('settings.updates')}</span>
              </button>
            )}

            <div className="nx-agent-summary-wrap">
              <button
                type="button"
                className="nx-agent-summary-btn"
                aria-expanded={agentsOpen}
                aria-controls="nexus-agent-summary"
                onClick={() => setAgentsOpen((open) => !open)}
                title="Abrir resumo operacional dos Agentes"
              >
                <Badge tone={attention || disconnected ? 'warning' : 'success'}>
                  <Network size={10} />
                  <span>{working} trabalhando · {awaiting} aguardando · {disconnected} desconectado{disconnected === 1 ? '' : 's'}</span>
                </Badge>
              </button>
              {agentsOpen && (
                <div id="nexus-agent-summary" className="nx-agent-summary-popover" role="dialog" aria-label="Agentes do Projeto">
                  <div className="nx-agent-summary-popover__heading"><strong>Agentes do Projeto</strong><button type="button" onClick={() => onOpenSurface?.('agents')}>Abrir lista completa</button></div>
                  {agentList.length === 0 ? <p className="nx-muted-copy">Nenhum Agente configurado.</p> : agentList.map((agent) => {
                    const runtime = runtimeByAgent.get(agent.id);
                    const live = Boolean(runtime && liveStates.includes(runtime.state));
                    const degraded = ['RECOVERABLE', 'FAILED', 'STALE', 'RATE_LIMITED'].includes(agent.status) || ['FAILED', 'STALE'].includes(runtime?.state || '');
                    const status = degraded ? (agent.status === 'WORKING' ? runtime?.state : agent.status) : runtime?.state || 'DISCONNECTED';
                    return <button type="button" className="nx-agent-summary-row" key={agent.id} onClick={() => {
                      setAgentsOpen(false);
                      if (live && runtime) onFocusRuntime?.(runtime.runtime_id);
                      else onOpenSurface?.('agents');
                    }}>
                      <span className="nx-agent-summary-row__name"><strong>{agent.name}</strong><small>{runtime?.provider || runtime?.provider_id || 'provider n/d'} · {runtime?.profile || runtime?.profile_id || 'perfil n/d'}</small></span>
                      <span className="nx-agent-summary-row__status" data-degraded={degraded ? 'true' : undefined}>{status}{runtime?.attention_kind === 'needs_user' ? ' · atenção' : ''}</span>
                    </button>;
                  })}
                </div>
              )}
            </div>

            {/* System Versions Pill */}
            <div className="nx-topbar-version-pill" title="Nexus Version" style={{ display: 'none' }}>
              <span className="nx-ver-nexus">Nexus v{sysInfo?.nexus_version || 'unknown'}</span>
            </div>

            {/* Language Switcher */}
            <LanguagePicker />

            {/* Command Palette Trigger */}
            <button
              type="button"
              className="nx-command-trigger"
              data-tour="command"
              onClick={onCommand}
              title="Search & Commands (Ctrl+K)"
            >
              <Command size={13} />
              <span>{t('shell.search')}</span>
              <kbd>Ctrl K</kbd>
            </button>

            {/* Push Notifications Toggle */}
            <IconButton
              label="Notificações Push"
              onClick={async () => {
                const perm = await pushNotifications.requestPermission();
                if (perm === 'granted') {
                  pushNotifications.confirmEnabled(project.name);
                }
              }}
            >
              {pushNotifications.getPermission() === 'granted' ? (
                <BellRing size={15} className="nx-text-emerald-400" />
              ) : (
                <Bell size={15} />
              )}
            </IconButton>

            {/* Help & Welcome Guide */}
            <IconButton label={t('shell.tour')} onClick={onOpenWelcome}>
              <CircleHelp size={15} />
            </IconButton>

            {/* Theme & Settings */}
            <IconButton label={t('shell.appearance')} onClick={onSettings}>
              <MoonStar size={15} />
            </IconButton>
          </div>
        </header>

          {/* Attention & Intermediation Alert Banner */}
          <AttentionIntermediationBanner
            runtimes={runtimes}
            focusedProjectId={project.id}
            onFocusRuntime={(rtId) => {
              if (onFocusRuntime) onFocusRuntime(rtId);
            }}
          />
        </div>

        {/* Workspace Canvas (Tabs inside stacks) */}
        <main id="nexus-workspace" className="nx-workspace-host">
          {children}
        </main>

        {/* OS Status Bar */}
        <WorkspaceTaskbar project={project} agents={agents} />
      </div>
      <InAppNotificationCenter
        runtimes={runtimes}
        focusedProjectId={project.id}
        onFocusRuntime={(runtimeId) => onFocusRuntime?.(runtimeId)}
      />
    </div>
  );
};
