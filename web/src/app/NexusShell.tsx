import React from 'react';
import { BrainCircuit, ChevronDown, CircleHelp, Command, Menu, MoonStar, Network, Wifi } from 'lucide-react';
import { Badge, IconButton } from '../design-system';
import { WorkspaceTaskbar } from '../workspace/WorkspaceTaskbar';
import type { Agent, Project } from '../types';

export const PROJECT_NAV = [
  ['overview','Overview'],['work','Work'],['missions','Plan'],['agents','Agents'],['maestro','Maestro'],['sessions','Sessions'],['resources','Usage'],['legacy-events','Events'],
] as const;

export const NexusShell: React.FC<{
  project: Project;
  agents: Agent[];
  rail: React.ReactNode;
  children: React.ReactNode;
  onOpenRail: () => void;
  onOpenSurface: (kind: string) => void;
  onCommand: () => void;
  onTour: () => void;
  onSettings: () => void;
}> = ({ project, agents, rail, children, onOpenRail, onOpenSurface, onCommand, onTour, onSettings }) => {
  const working=agents.filter((agent)=>agent.status==='WORKING').length;
  const attention=agents.filter((agent)=>['FAILED','STALE','RECOVERABLE','RATE_LIMITED'].includes(agent.status)).length;
  return <div className="nx-os-shell"><a href="#nexus-workspace" className="nx-skip-link">Skip to workspace</a>{rail}<div className="nx-os-main"><header className="nx-topbar"><div className="nx-topbar__context"><IconButton className="nx-mobile-menu" label="Open Projects" onClick={onOpenRail}><Menu size={16}/></IconButton><span className="nx-topbar__project"><span className="nx-project-avatar nx-project-avatar--top">{project.name.slice(0,2).toUpperCase()}</span><span><strong>{project.name}</strong><small>{project.default_branch||'main'}</small></span><ChevronDown size={12}/></span><span className="nx-topbar__path">{project.canonical_path}</span></div><div className="nx-topbar__status" data-tour="status"><Badge tone={attention?'warning':'success'}><Network size={10}/>{working} working</Badge><Badge tone={project.maestro_mode==='OFF'?'default':'brand'}><BrainCircuit size={10}/>{project.maestro_mode||'ASSIST'}</Badge><span className="nx-connection"><Wifi size={12}/> local</span><button type="button" className="nx-command-trigger" data-tour="command" onClick={onCommand}><Command size={13}/><span>Search & commands</span><kbd>Ctrl K</kbd></button><IconButton label="Product tour" onClick={onTour}><CircleHelp size={15}/></IconButton><IconButton label="Appearance settings" onClick={onSettings}><MoonStar size={15}/></IconButton></div></header><nav className="nx-project-nav" data-tour="project-nav" aria-label="Project navigation">{PROJECT_NAV.map(([id,label])=><button key={id} type="button" onClick={()=>onOpenSurface(id)}>{label}</button>)}</nav><main id="nexus-workspace" className="nx-workspace-host">{children}</main><WorkspaceTaskbar/></div></div>;
};
