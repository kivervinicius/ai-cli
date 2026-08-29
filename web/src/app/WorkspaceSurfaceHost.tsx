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
import { MaestroPage } from '../nexus/MaestroPage';
import { MissionsPage } from '../nexus/MissionsPage';
import { ResourcePicker } from '../nexus/ResourcePicker';
import { api } from '../api';
import type { Agent, EventRecord, ProfileInfo, Project, ProviderInfo, RuntimeSession, Workspace } from '../types';
import type { WorkspaceSurface } from '../workspace/model';
import { ProjectOverviewSurface } from '../features/overview/ProjectOverviewSurface';
import { WorkSurface } from '../features/work/WorkSurface';
import { AgentsSurface } from '../features/agents/AgentsSurface';
import { AgentConfigurationSurface } from '../features/agents/AgentConfigurationSurface';
import { SessionsSurface } from '../features/sessions/SessionsSurface';
import { SettingsSurface } from '../features/settings/SettingsSurface';

export const WorkspaceSurfaceHost: React.FC<{
  surface: WorkspaceSurface;
  project: Project;
  agents: Agent[];
  workspaces: Workspace[];
  runtimes: RuntimeSession[];
  providers: ProviderInfo[];
  profiles: ProfileInfo[];
  events: EventRecord[];
  refreshAgents: () => Promise<void>;
  refreshGlobal: () => Promise<void>;
  openSurface: (surface: WorkspaceSurface) => void;
  onTour: () => void;
}> = ({ surface, project, agents, workspaces, runtimes, providers, profiles, events, refreshAgents, refreshGlobal, openSurface, onTour }) => {
  const agent = useMemo(() => agents.find((item) => item.id === surface.data?.agentId), [agents, surface.data?.agentId]);
  const [showStart,setShowStart]=useState(false); const [handoff,setHandoff]=useState<RuntimeSession|null>(null); const [cont,setCont]=useState<RuntimeSession|null>(null);
  const open = (kind: string) => openSurface({ id:`project:${project.id}:${kind}`, type:kind, title: kind==='missions'?'Plan':kind[0].toUpperCase()+kind.slice(1), data:{projectId:project.id}, closable:kind!=='overview' });
  const terminal = (target: Agent) => openSurface({ id:`agent:${target.id}:terminal`, type:'terminal', title:target.name, subtitle:'Persistent Agent terminal', data:{agentId:target.id} });
  const config = (target: Agent) => openSurface({ id:`agent:${target.id}:config`, type:'agent-config', title:`${target.name} · Configure`, data:{agentId:target.id} });

  if(surface.type==='overview') return <ProjectOverviewSurface project={project} agents={agents} onOpenAgent={terminal} onOpenWork={()=>open('work')} onOpenPlan={()=>open('missions')}/>;
  if(surface.type==='work') return <WorkSurface project={project} agents={agents} onDirect={terminal} onPlan={()=>open('missions')} onMaestro={()=>open('maestro')}/>;
  if(surface.type==='agents') return <AgentsSurface project={project} agents={agents} refresh={refreshAgents} onTerminal={terminal} onConfigure={config}/>;
  if(surface.type==='agent-config') return <AgentConfigurationSurface agent={agent} onApplied={refreshAgents}/>;
  if(surface.type==='terminal') return surface.data?.agentId ? <div className="nx-agent-terminal-surface"><AgentTerminal agentId={surface.data.agentId}/></div> : <EmptyState title="Agent terminal unavailable"/>;
  if(surface.type==='resources') return <div className="nx-surface-scroll"><div className="nx-page-header"><div><span className="nx-eyebrow">Resource intelligence</span><h1>Providers, accounts & quota</h1><p>Nexus explains selection rather than pretending the largest quota is always the best resource.</p></div></div><div className="nx-resource-layout"><Card className="nx-resource-card"><ResourcePicker onSelect={()=>undefined}/></Card><Card className="nx-resource-card"><div className="nx-feature-heading"><Gauge size={18}/><div><strong>Allocation policy</strong><small>{project.resource_policy||'BALANCED'}</small></div></div><p className="nx-muted-copy">Capabilities, health, quota, continuity, Project preference and switching cost contribute to allocation. UNKNOWN quota remains unknown.</p></Card></div></div>;
  if(surface.type==='maestro') return <div className="nx-surface-scroll"><div className="nx-page-header"><div><span className="nx-eyebrow">Engineering intelligence</span><h1>Maestro</h1><p>Process, skills, risk and verification guidance remain separate from provider allocation.</p></div></div><MaestroPage projectId={project.id}/></div>;
  if(surface.type==='missions') return <div className="nx-surface-scroll"><div className="nx-page-header"><div><span className="nx-eyebrow">Plan & Missions</span><h1>From goal to executable work</h1><p>This Beta surface persists Mission goals/tasks today and is the integration point for WorkPlan, phases, packages and the future durable Mission Runner.</p></div></div><MissionsPage projectId={project.id}/></div>;
  if(surface.type==='sessions') return <SessionsSurface agents={agents}/>;
  if(surface.type==='settings') return <SettingsSurface onTour={onTour}/>;
  if(surface.type==='legacy-providers') return <div className="nx-legacy-surface"><ProvidersView providers={providers}/></div>;
  if(surface.type==='legacy-events') return <div className="nx-legacy-surface"><EventsView events={events}/></div>;
  if(surface.type==='legacy-terminal') {
    const runtime=runtimes.find((item)=>item.runtime_id===surface.data?.runtimeId);
    if(!runtime)return <EmptyState title="Runtime unavailable"/>;
    return <TerminalPane runtimeId={runtime.runtime_id} title={runtime.title} provider={runtime.provider_id||runtime.provider||'AI'} profile={runtime.profile_id||runtime.profile||'default'} onUpdateTitle={async(id,title)=>{await api.updateRuntimeTitle(id,title);await refreshGlobal();}}/>;
  }
  if(surface.type==='legacy-runtimes') return <div className="nx-legacy-surface"><Dashboard runtimes={runtimes} providers={providers} workspaces={workspaces} activeWorkspace={project.canonical_path} onOpenTerminal={(runtimeId)=>{const runtime=runtimes.find((item)=>item.runtime_id===runtimeId);openSurface({id:`runtime:${runtimeId}:terminal`,type:'legacy-terminal',title:runtime?.title||runtimeId,data:{runtimeId}});}} onOpenStartModal={()=>setShowStart(true)} onOpenHandoffModal={setHandoff} onOpenContinueModal={setCont} onStopRuntime={async(id)=>{await api.stopRuntime(id);await refreshGlobal();}} onDeleteRuntime={async(id)=>{await api.deleteRuntime(id);await refreshGlobal();}} onCleanInactive={async()=>{await api.cleanRuntimes();await refreshGlobal();}}/>{showStart&&<StartModal providers={providers} profiles={profiles} workspace={project.canonical_path} workspaces={workspaces} onClose={()=>setShowStart(false)} onSuccess={async()=>{setShowStart(false);await refreshGlobal();}}/>}{handoff&&<HandoffModal runtime={handoff} profiles={profiles} onClose={()=>setHandoff(null)} onSuccess={async()=>{setHandoff(null);await refreshGlobal();}}/>}{cont&&<ContinueModal runtime={cont} providers={providers} profiles={profiles} onClose={()=>setCont(null)} onSuccess={async()=>{setCont(null);await refreshGlobal();}}/>}</div>;
  return <div className="nx-surface-center"><EmptyState icon={<Network size={22}/>} title="Surface unavailable" hint={`Unknown workspace surface: ${surface.type}`} action={<Button onClick={()=>open('overview')}>Open Overview</Button>}/></div>;
};
