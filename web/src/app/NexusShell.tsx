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
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { IconButton } from '../design-system';
import { LanguagePicker } from './components/LanguagePicker';
import { WorkspaceTaskbar } from '../workspace/WorkspaceTaskbar';
import { nexus } from '../nexus/api';
import { AttentionIntermediationBanner } from '../components/AttentionIntermediationBanner';
import { pushNotifications } from '../notifications/PushNotificationManager';
import { InAppNotificationCenter } from '../notifications/InAppNotificationCenter';
import { ProjectCreateMenu } from '../features/projects/ProjectCreateMenu';
import type { Agent, Project, RuntimeSession } from '../types';
import type { RadarRuntimeItem } from './attentionRadarModel';
import styles from './NexusShell.module.scss';

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
  onFocusAgent?: (agentId: string) => void;
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
  onFocusAgent,
}) => {
  const { t } = useTranslation();
  const [notificationDrawerOpen, setNotificationDrawerOpen] = useState(false);
  const [sysInfo, setSysInfo] = useState<{
    nexus_version: string;
    maestro_version: string;
    maestro_available: boolean;
    update_available: boolean;
  } | null>(null);

  useEffect(() => {
    const load = () => {
      nexus
        .getSystemUpdates()
        .then(setSysInfo)
        .catch(() => undefined);
    };
    load();
    window.addEventListener('nexus:system-updates', load);
    return () => window.removeEventListener('nexus:system-updates', load);
  }, []);

  const hasAttentionAlerts = runtimes.some(
    (rt) =>
      rt.attention_kind === 'needs_user' ||
      rt.attention_reason === 'QUESTION' ||
      rt.attention_reason === 'APPROVAL' ||
      rt.attention_reason === 'ERROR' ||
      rt.state === 'FAILED',
  );

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
              <IconButton
                className="nx-mobile-menu"
                label={t('shell.openProjects')}
                onClick={onOpenRail}
              >
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

              <ProjectCreateMenu
                onNewAgent={onNewAgent}
                onNewAISession={onNewAISession}
                onProjectShell={onProjectShell}
                size="sm"
                variant="topbar"
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
                  <span className="nx-update-badge">Updates</span>
                </button>
              )}

              {/* System Versions Pill */}
              <div
                className="nx-topbar-version-pill"
                title="Nexus Version"
                style={{ display: 'none' }}
              >
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

              {/* Push & In-App Attention Notification Trigger */}
              <div className={styles.notificationWrap}>
                <IconButton
                  label="Central de Notificações e Radar"
                  onClick={() => setNotificationDrawerOpen((prev) => !prev)}
                >
                  {pushNotifications.getPermission() === 'granted' ? (
                    <BellRing size={15} className="nx-text-emerald-400" />
                  ) : (
                    <Bell size={15} />
                  )}
                </IconButton>
                {hasAttentionAlerts && <span className={styles.attentionIndicator} />}
              </div>

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

        {/* OS Status Bar with interactive direct agent focus */}
        <WorkspaceTaskbar project={project} agents={agents} onFocusAgent={onFocusAgent} />
      </div>

      <InAppNotificationCenter
        runtimes={runtimes}
        focusedProjectId={project.id}
        drawerOpen={notificationDrawerOpen}
        onCloseDrawer={() => setNotificationDrawerOpen(false)}
        onFocusRuntime={(runtimeId) => onFocusRuntime?.(runtimeId)}
        onFocusAttention={(item) => onFocusAttention?.(item)}
      />
    </div>
  );
};
