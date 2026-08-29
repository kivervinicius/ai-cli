import React, { useEffect, useState } from 'react';
import { ArrowUpCircle, BrainCircuit, ChevronDown, CircleHelp, Command, Menu, MoonStar, Network, Wifi } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Badge, IconButton } from '../design-system';
import { WorkspaceTaskbar } from '../workspace/WorkspaceTaskbar';
import { nexus } from '../nexus/api';
import type { Agent, Project } from '../types';

export const PROJECT_NAV = ['overview', 'work', 'missions', 'agents', 'maestro', 'sessions', 'resources', 'legacy-events'] as const;

export const NexusShell: React.FC<{ project: Project; agents: Agent[]; rail: React.ReactNode; children: React.ReactNode; onOpenRail: () => void; onOpenSurface: (kind: string) => void; onCommand: () => void; onTour: () => void; onSettings: () => void }> = ({ project, agents, rail, children, onOpenRail, onOpenSurface, onCommand, onTour, onSettings }) => {
  const { t } = useTranslation();
  const working = agents.filter((agent) => agent.status === 'WORKING').length;
  const attention = agents.filter((agent) => ['FAILED', 'STALE', 'RECOVERABLE', 'RATE_LIMITED'].includes(agent.status)).length;
  const [hasUpdate, setHasUpdate] = useState(false);
  useEffect(() => { nexus.getSystemUpdates().then((info) => setHasUpdate(info.update_available)).catch(() => undefined); }, []);
  const navKey = (id: typeof PROJECT_NAV[number]) => id === 'legacy-events' ? 'events' : id;
  return <div className="nx-os-shell"><a href="#nexus-workspace" className="nx-skip-link">{t('shell.skip')}</a>{rail}<div className="nx-os-main"><header className="nx-topbar"><div className="nx-topbar__context"><IconButton className="nx-mobile-menu" label={t('shell.openProjects')} onClick={onOpenRail}><Menu size={16}/></IconButton><span className="nx-topbar__project"><span className="nx-project-avatar nx-project-avatar--top">{project.name.slice(0, 2).toUpperCase()}</span><span><strong>{project.name}</strong><small>{project.default_branch || 'main'}</small></span><ChevronDown size={12}/></span><span className="nx-topbar__path">{project.canonical_path}</span></div><div className="nx-topbar__status" data-tour="status">{hasUpdate && <button type="button" onClick={onSettings} className="nx-update-indicator" title={t('settings.updates')}><ArrowUpCircle size={13} className="nx-spin-slow"/><span>{t('settings.updates')}</span></button>}<Badge tone={attention ? 'warning' : 'success'}><Network size={10}/>{t('shell.working', { count: working })}</Badge><Badge tone={project.maestro_mode === 'OFF' ? 'default' : 'brand'}><BrainCircuit size={10}/>{project.maestro_mode || 'ASSIST'}</Badge><span className="nx-connection"><Wifi size={12}/> {t('common.local')}</span><button type="button" className="nx-command-trigger" data-tour="command" onClick={onCommand}><Command size={13}/><span>{t('shell.search')}</span><kbd>Ctrl K</kbd></button><IconButton label={t('shell.tour')} onClick={onTour}><CircleHelp size={15}/></IconButton><IconButton label={t('shell.appearance')} onClick={onSettings}><MoonStar size={15}/></IconButton></div></header><nav className="nx-project-nav" data-tour="project-nav" aria-label={t('shell.navigation')}>{PROJECT_NAV.map((id) => <button key={id} type="button" onClick={() => onOpenSurface(id)}>{t(`nav.${navKey(id)}`)}</button>)}</nav><main id="nexus-workspace" className="nx-workspace-host">{children}</main><WorkspaceTaskbar/></div></div>;
};
