import React, { useMemo, useState } from 'react';
import { BrainCircuit, Play, Route, Sparkles, TerminalSquare } from 'lucide-react';
import { Badge, Button, Card, Segmented, Textarea } from '../../design-system';
import type { Agent, Project } from '../../types';

export const WorkSurface: React.FC<{ project: Project; agents: Agent[]; onDirect: (agent: Agent) => void; onPlan: () => void; onMaestro: () => void }> = ({ project, agents, onDirect, onPlan, onMaestro }) => {
  const [mode, setMode] = useState('assisted');
  const [prompt, setPrompt] = useState('');
  const preferred = useMemo(() => agents.find((agent) => agent.status === 'WORKING') ?? agents[0], [agents]);
  return <div className="nx-surface-scroll nx-work-surface"><div className="nx-page-header"><div><span className="nx-eyebrow">Nexus Work</span><h1>What do you want to accomplish?</h1><p>Nexus keeps the Project context while you decide how much planning and assistance you want.</p></div><Badge tone="brand">{project.name}</Badge></div>
    <Card className="nx-work-composer"><div className="nx-work-composer__top"><Segmented ariaLabel="Work mode" value={mode} onChange={setMode} options={[{value:'direct',label:'Direct'},{value:'assisted',label:'Assisted'},{value:'planned',label:'Planned'}]}/><span className="nx-muted-copy">{mode === 'direct' ? 'Send work directly to an Agent.' : mode === 'assisted' ? 'Prepare richer context with Maestro guidance.' : 'Turn the goal into a Mission/WorkPlan before execution.'}</span></div><Textarea rows={8} value={prompt} onChange={setPrompt} placeholder="Describe the outcome, constraints and anything Nexus must preserve…"/><div className="nx-work-composer__actions"><Button onClick={onMaestro}><BrainCircuit size={14}/> Ask Maestro</Button>{mode === 'planned' ? <Button tone="brand" onClick={onPlan}><Route size={14}/> Build plan</Button> : <Button tone="brand" disabled={!preferred || !prompt.trim()} onClick={() => preferred && onDirect(preferred)}><Play size={14}/> {mode === 'direct' ? 'Open Agent' : 'Prepare with Agent'}</Button>}</div></Card>
    <div className="nx-work-cards"><Card><span className="nx-work-card-icon"><TerminalSquare size={17}/></span><strong>Direct</strong><p>For small, explicit changes. You stay close to the terminal and control the prompt yourself.</p></Card><Card><span className="nx-work-card-icon"><Sparkles size={17}/></span><strong>Assisted</strong><p>Nexus Intelligence can later enrich intent, context and prompts without forcing a full Mission workflow.</p></Card><Card><span className="nx-work-card-icon"><Route size={17}/></span><strong>Planned</strong><p>Use Missions/WorkPlan for phases, packages, dependencies, skills and governed autonomous execution.</p></Card></div>
  </div>;
};
