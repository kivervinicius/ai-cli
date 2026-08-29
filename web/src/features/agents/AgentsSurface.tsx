import React, { useState } from 'react';
import { Bot, Play, Plus, RotateCcw, Settings2, Square, TerminalSquare } from 'lucide-react';
import { Badge, Button, Card, EmptyState, Input } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { Agent, Project } from '../../types';
import { translateStatus } from '../../i18n';
import { useTranslation } from 'react-i18next';

const tone = (status: string) => status === 'WORKING' ? 'success' : status === 'FAILED' || status === 'STALE' ? 'danger' : status === 'RECOVERABLE' || status === 'WAITING' || status === 'RATE_LIMITED' ? 'warning' : 'default';
export const AgentsSurface: React.FC<{ project: Project; agents: Agent[]; refresh: () => Promise<void>; onTerminal: (agent: Agent) => void; onConfigure: (agent: Agent) => void }> = ({ project, agents, refresh, onTerminal, onConfigure }) => {
  const { t } = useTranslation();
  const [name, setName] = useState(''); const [busy, setBusy] = useState('');
  const create = async () => { if (!name.trim()) return; setBusy('create'); try { await nexus.createAgent(project.id, name.trim(), 'developer'); setName(''); await refresh(); } finally { setBusy(''); } };
  const action = async (agent: Agent, kind: 'start'|'stop'|'recover') => { setBusy(agent.id); try { if (kind === 'start') await nexus.startAgent(agent.id); else if (kind === 'stop') await nexus.stopAgent(agent.id); else await nexus.recoverAgent(agent.id); await refresh(); } finally { setBusy(''); } };
  return <div className="nx-surface-scroll"><div className="nx-page-header"><div><span className="nx-eyebrow">{t('agents.eyebrow')}</span><h1>{t('agents.title')}</h1><p>{t('agents.intro')}</p></div><div className="nx-inline-create"><Input value={name} onChange={setName} onEnter={create} placeholder={t('agents.name')}/><Button tone="brand" disabled={!name.trim() || busy === 'create'} onClick={create}><Plus size={14}/> {t('agents.create')}</Button></div></div>
    {agents.length === 0 ? <EmptyState icon={<Bot size={22}/>} title={t('agents.empty')} hint={t('agents.emptyHint')}/> : <div className="nx-agent-grid">{agents.map((agent) => <Card key={agent.id} className="nx-agent-card"><div className="nx-agent-card__head"><span className="nx-agent-avatar nx-agent-avatar--large">{agent.name.slice(0,2).toUpperCase()}</span><div><strong>{agent.name}</strong><small>{agent.role || t('agents.developer')} · {agent.id}</small></div><Badge tone={tone(agent.status)}>{translateStatus(agent.status)}</Badge></div><div className="nx-agent-card__meta"><span>{t('agents.continuity')}</span><strong>{agent.continuity_status || t('common.unknown')}</strong><span>{t('agents.lastStart')}</span><strong>{agent.last_started_at || t('common.never')}</strong></div><div className="nx-agent-card__actions"><Button size="sm" onClick={() => onTerminal(agent)}><TerminalSquare size={13}/> {t('agents.terminal')}</Button><Button size="sm" onClick={() => onConfigure(agent)}><Settings2 size={13}/> {t('agents.configure')}</Button>{agent.status === 'WORKING' ? <Button size="sm" tone="danger" disabled={busy === agent.id} onClick={() => void action(agent,'stop')}><Square size={12}/> {t('agents.stop')}</Button> : agent.status === 'RECOVERABLE' ? <Button size="sm" tone="warning" disabled={busy === agent.id} onClick={() => void action(agent,'recover')}><RotateCcw size={13}/> {t('agents.recover')}</Button> : <Button size="sm" tone="brand" disabled={busy === agent.id} onClick={() => void action(agent,'start')}><Play size={13}/> {t('agents.start')}</Button>}</div></Card>)}</div>}
  </div>;
};
