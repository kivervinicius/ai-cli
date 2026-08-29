import React, { useEffect, useState } from 'react';
import { GitBranch, History } from 'lucide-react';
import { Badge, Card, EmptyState, Spinner } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { Agent, AgentDetail } from '../../types';
export const SessionsSurface: React.FC<{ agents: Agent[] }> = ({ agents }) => {
  const [details,setDetails]=useState<AgentDetail[]>([]); const [loading,setLoading]=useState(true);
  useEffect(()=>{ Promise.all(agents.map((agent)=>nexus.getAgent(agent.id).catch(()=>null))).then((values)=>setDetails(values.filter(Boolean) as AgentDetail[])).finally(()=>setLoading(false)); },[agents]);
  if(loading) return <div className="nx-surface-center"><Spinner label="Loading session lineage…"/></div>;
  return <div className="nx-surface-scroll"><div className="nx-page-header"><div><span className="nx-eyebrow">Continuity & lineage</span><h1>Sessions</h1><p>Runtime generations are implementation details under a stable Agent identity.</p></div></div>{details.length===0?<EmptyState icon={<History size={22}/>} title="No session history" hint="Start an Agent to create its first runtime generation."/>:<div className="nx-session-list">{details.map((detail)=><Card key={detail.agent.id} className="nx-session-card"><div className="nx-session-card__head"><div><strong>{detail.agent.name}</strong><small>{detail.agent.id}</small></div><Badge>{detail.agent.continuity_status}</Badge></div><div className="nx-generation-list">{detail.generations.length===0?<span className="nx-muted-copy">No generations recorded.</span>:detail.generations.map((generation)=><div key={generation.id}><GitBranch size={13}/><div><strong>{generation.provider}/{generation.profile}</strong><small>{generation.runtime_id} · {generation.provider_session || 'session unknown'}</small></div><Badge>{generation.state}</Badge><Badge tone={generation.continuity==='NATIVE_RESUME_VERIFIED'?'success':'default'}>{generation.continuity}</Badge></div>)}</div></Card>)}</div>}</div>;
};
