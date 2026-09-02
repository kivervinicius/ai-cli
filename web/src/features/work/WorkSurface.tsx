import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { BrainCircuit, Clipboard, Network, Play, RefreshCw, Route, Send, Sparkles, TerminalSquare } from 'lucide-react';
import { Badge, Button, Card, Textarea } from '../../design-system';
import type { Agent, ContextReadiness, ContextReadinessState, MissionRun, Project } from '../../types';
import { nexus } from '../../nexus/api';
import { PlanBuilderSurface } from './PlanBuilderSurface';
import { composerGateForReadiness } from './composerModel';
import { askActionForStatus } from '../agents/askAgentModel';

const readinessTone = (state: ContextReadinessState) => state === 'READY' ? 'success' : state === 'FAILED' ? 'danger' : state === 'STALE' ? 'warning' : 'default';

export const WorkSurface: React.FC<{
  project: Project;
  agents: Agent[];
  onDirect: (agent: Agent) => void;
  onStartSession?: (mode: 'direct' | 'assisted', prompt: string) => void;
  onPlan: () => void;
  onMaestro: () => void;
  onFlowRun?: (run: MissionRun) => void;
}> = ({ project, agents, onDirect, onStartSession, onPlan, onMaestro, onFlowRun }) => {
  const [prompt, setPrompt] = useState('');
  const [selectedAgentId, setSelectedAgentId] = useState('');
  const [readiness, setReadiness] = useState<ContextReadiness | null>(null);
  const [readinessBusy, setReadinessBusy] = useState(false);
  const [sending, setSending] = useState(false);
  const [showFlow, setShowFlow] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');

  const preferred = useMemo(() => agents.find((agent) => agent.id === selectedAgentId) ?? agents.find((agent) => agent.status === 'WORKING') ?? agents[0], [agents, selectedAgentId]);
  useEffect(() => { if (preferred && !selectedAgentId) setSelectedAgentId(preferred.id); }, [preferred, selectedAgentId]);

  const refreshContext = useCallback(async () => {
    try { setReadiness(await nexus.getContextReadiness(project.id)); }
    catch (err) { setError(err instanceof Error ? err.message : String(err)); }
  }, [project.id]);
  useEffect(() => { void refreshContext(); }, [refreshContext]);

  const state: ContextReadinessState = readiness?.state ?? 'MISSING';
  const gate = composerGateForReadiness(state);

  const prepareContext = async () => {
    setReadinessBusy(true); setError('');
    try { setReadiness(await nexus.prepareContext(project.id)); }
    catch (err) { setError(err instanceof Error ? err.message : String(err)); }
    finally { setReadinessBusy(false); }
  };

  const sendExisting = async () => {
    if (!preferred || !prompt.trim()) return;
    setSending(true); setError(''); setNotice('');
    try {
      const action = askActionForStatus(preferred.status);
      await nexus.askAgent(preferred.id, prompt.trim(), action.startIfNeeded);
      setNotice(`Sent to ${preferred.name}; the same Agent identity was reused.`);
      onDirect(preferred);
    } catch (err) { setError(err instanceof Error ? err.message : String(err)); }
    finally { setSending(false); }
  };

  return <div className="nx-surface-scroll nx-composer-surface">
    <div className="nx-page-header"><div><span className="nx-eyebrow"><Sparkles size={13}/> COMPOSER</span><h1>Direct work when simple. Flow when structure matters.</h1><p>Composer uses durable project context for planning, but never blocks direct Agent or shell work.</p></div><div className="nx-composer-header-actions"><Badge tone="brand">{project.name}</Badge><Badge tone={readinessTone(state)}>Context {state}</Badge><Button size="sm" onClick={onPlan}><Route size={13}/> Flow Runs</Button></div></div>

    <Card className="nx-context-readiness-card" data-state={state}>
      <div className="nx-context-readiness-card__status"><Network size={17}/><div><strong>Context Readiness · {state}</strong><p>{gate.reason}</p>{readiness?.error && <small>{readiness.error}</small>}</div></div>
      <div className="nx-context-readiness-card__meta"><span>{readiness?.current_fingerprint?.branch || project.default_branch || 'main'}</span>{readiness?.current_fingerprint?.head && <code>{(readiness.current_fingerprint.head || '').slice(0,8)}</code>}<span>Maestro {readiness?.maestro_version || 'checking'}</span></div>
      {!gate.canCompose && gate.action !== 'WAIT' && <Button size="sm" tone="brand" disabled={readinessBusy} onClick={() => void prepareContext()}><RefreshCw size={13}/> {readinessBusy ? 'Checking…' : gate.action === 'PREPARE' ? 'Prepare Context' : 'Refresh Context'}</Button>}
    </Card>

    <Card className="nx-work-composer nx-composer-editor">
      <Textarea rows={9} value={prompt} onChange={setPrompt} placeholder="Describe the outcome you want…" />
      <div className="nx-composer-target-row"><label><span>Existing Agent</span><select className="nx-select" value={preferred?.id || ''} onChange={(event) => setSelectedAgentId(event.target.value)} disabled={!agents.length}>{!agents.length && <option value="">No Agent available</option>}{agents.map((agent) => <option key={agent.id} value={agent.id}>{agent.name} · {agent.status}</option>)}</select></label><span className="nx-muted-copy">Direct work stays available regardless of Context Readiness.</span></div>
      {error && <div className="nx-inline-error">{error}</div>}{notice && <div className="nx-inline-notice">{notice}</div>}
      <div className="nx-composer-actions">
        <Button onClick={onMaestro}><BrainCircuit size={14}/> Maestro Status</Button>
        <Button disabled={!prompt.trim()} onClick={() => void navigator.clipboard.writeText(prompt.trim()).then(() => setNotice('Prompt copied.')).catch(() => setError('Could not copy prompt.'))}><Clipboard size={14}/> Copy Prompt</Button>
        <Button disabled={!preferred || !prompt.trim() || sending} onClick={() => void sendExisting()}><Send size={14}/> {sending ? 'Sending…' : 'Send to Agent'}</Button>
        <Button disabled={!prompt.trim() || !onStartSession} onClick={() => onStartSession?.('direct', prompt.trim())}><TerminalSquare size={14}/> New AI Session</Button>
        <Button tone="brand" disabled={!gate.canCompose || !prompt.trim()} onClick={() => setShowFlow(true)}><Play size={14}/> Turn into Flow</Button>
      </div>
    </Card>

    {showFlow && gate.canCompose && <div className="nx-composer-flow-region"><div className="nx-section-title"><div><h2>Flow Draft</h2><p>Drafting is side-effect-free; execution remains an explicit action inside the existing Plan/Mission engine.</p></div><Button size="sm" onClick={() => setShowFlow(false)}>Hide Draft</Button></div><PlanBuilderSurface project={project} agents={agents} onOpenAgent={onDirect} onRunCreated={onFlowRun} initialGoal={prompt.trim()} /></div>}
  </div>;
};
