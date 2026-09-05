import React, { useState } from 'react';
import {
  Bot,
  MessageSquareText,
  Play,
  Plus,
  RotateCcw,
  Settings2,
  Square,
  TerminalSquare,
  Trash2,
} from 'lucide-react';
import { Badge, Button, Card, ConfirmDialog, Dialog, EmptyState, Input } from '../../design-system';
import { nexus } from '../../nexus/api';
import { ResourcePicker } from '../../nexus/ResourcePicker';
import type { Agent, Project } from '../../types';
import { translateStatus } from '../../i18n';
import { useTranslation } from 'react-i18next';
import { AskAgentDialog } from './AskAgentDialog';

const tone = (status: string) =>
  status === 'WORKING'
    ? 'success'
    : status === 'FAILED' || status === 'STALE'
      ? 'danger'
      : status === 'RECOVERABLE' || status === 'WAITING' || status === 'RATE_LIMITED'
        ? 'warning'
        : 'default';

export const AgentsSurface: React.FC<{
  project: Project;
  agents: Agent[];
  refresh: () => Promise<void>;
  onTerminal: (agent: Agent) => void;
  onConfigure: (agent: Agent) => void;
  onRemoved?: (agentId: string) => void;
}> = ({ project, agents, refresh, onTerminal, onConfigure, onRemoved }) => {
  const { t } = useTranslation();
  const [name, setName] = useState('');
  const [busy, setBusy] = useState('');
  const [resourceAgent, setResourceAgent] = useState<Agent | null>(null);
  const [askAgent, setAskAgent] = useState<Agent | null>(null);
  const [agentToDelete, setAgentToDelete] = useState<Agent | null>(null);
  const [error, setError] = useState('');

  const create = async () => {
    if (!name.trim()) return;
    setBusy('create');
    try {
      await nexus.createAgent(project.id, name.trim(), 'developer');
      setName('');
      await refresh();
    } finally {
      setBusy('');
    }
  };

  const action = async (agent: Agent, kind: 'start' | 'stop' | 'recover') => {
    setBusy(agent.id);
    setError('');
    try {
      if (kind === 'start') await nexus.startAgent(agent.id);
      else if (kind === 'stop') await nexus.stopAgent(agent.id);
      else await nexus.recoverAgent(agent.id);
      await refresh();
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      if (kind === 'start' && message.includes('REQUIRED_RESOURCE_SELECTION'))
        setResourceAgent(agent);
      else setError(message);
    } finally {
      setBusy('');
    }
  };

  const remove = (agent: Agent) => {
    setAgentToDelete(agent);
  };

  const executeRemove = async () => {
    if (!agentToDelete) return;
    const agent = agentToDelete;
    setAgentToDelete(null);
    setBusy(agent.id);
    setError('');
    try {
      try {
        await nexus.stopAgent(agent.id);
      } catch {
        /* may already be dead */
      }
      await nexus.deleteAgent(agent.id);
      onRemoved?.(agent.id);
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy('');
    }
  };

  const allocateAndStart = async () => {
    if (!resourceAgent) return;
    setBusy(resourceAgent.id);
    setError('');
    try {
      await nexus.startAgent(resourceAgent.id);
      await refresh();
      setResourceAgent(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy('');
    }
  };

  return (
    <div className="nx-surface-scroll">
      <div className="nx-page-header">
        <div>
          <span className="nx-eyebrow">{t('agents.eyebrow')}</span>
          <h1>{t('agents.title')}</h1>
          <p>{t('agents.intro')}</p>
        </div>
        <div className="nx-inline-create">
          <Input value={name} onChange={setName} onEnter={create} placeholder={t('agents.name')} />
          <Button tone="brand" disabled={!name.trim() || busy === 'create'} onClick={create}>
            <Plus size={14} /> {t('agents.create')}
          </Button>
        </div>
      </div>
      {error && <Card className="nx-inline-error">{error}</Card>}
      {agents.length === 0 ? (
        <EmptyState
          icon={<Bot size={22} />}
          title={t('agents.empty')}
          hint={t('agents.emptyHint')}
        />
      ) : (
        <div className="nx-agent-grid">
          {agents.map((agent) => (
            <Card key={agent.id} className="nx-agent-card">
              <div className="nx-agent-card__head">
                <span className="nx-agent-avatar nx-agent-avatar--large">
                  {(agent.name || 'AG').slice(0, 2).toUpperCase()}
                </span>
                <div>
                  <strong>{agent.name || agent.id}</strong>
                  <small>
                    {agent.role || t('agents.developer')} · {agent.id}
                  </small>
                </div>
                <Badge tone={tone(agent.status)}>{translateStatus(agent.status)}</Badge>
              </div>
              <div className="nx-agent-card__meta">
                <span>{t('agents.continuity')}</span>
                <strong>{agent.continuity_status || t('common.unknown')}</strong>
                <span>{t('agents.lastStart')}</span>
                <strong>{agent.last_started_at || t('common.never')}</strong>
              </div>
              <div className="nx-agent-card__actions">
                <Button size="sm" onClick={() => onTerminal(agent)}>
                  <TerminalSquare size={13} /> {t('agents.terminal')}
                </Button>
                <Button size="sm" onClick={() => setAskAgent(agent)}>
                  <MessageSquareText size={13} /> Ask
                </Button>
                <Button size="sm" onClick={() => onConfigure(agent)}>
                  <Settings2 size={13} /> {t('agents.configure')}
                </Button>
                {agent.status === 'WORKING' ? (
                  <Button
                    size="sm"
                    tone="danger"
                    disabled={busy === agent.id}
                    onClick={() => void action(agent, 'stop')}
                  >
                    <Square size={12} /> {t('agents.stop')}
                  </Button>
                ) : agent.status === 'RECOVERABLE' ? (
                  <Button
                    size="sm"
                    tone="warning"
                    disabled={busy === agent.id}
                    onClick={() => void action(agent, 'recover')}
                  >
                    <RotateCcw size={13} /> {t('agents.recover')}
                  </Button>
                ) : (
                  <Button
                    size="sm"
                    tone="brand"
                    disabled={busy === agent.id}
                    onClick={() => void action(agent, 'start')}
                  >
                    <Play size={13} /> {t('agents.start')}
                  </Button>
                )}
                <Button
                  size="sm"
                  tone="danger"
                  disabled={busy === agent.id}
                  onClick={() => void remove(agent)}
                >
                  <Trash2 size={13} /> {t('agents.remove')}
                </Button>
              </div>
            </Card>
          ))}
        </div>
      )}
      <AskAgentDialog
        agent={askAgent}
        onClose={() => setAskAgent(null)}
        onSent={async (target) => {
          await refresh();
          onTerminal(target);
        }}
      />
      <Dialog
        open={!!resourceAgent}
        onClose={() => setResourceAgent(null)}
        title={resourceAgent ? `Select resource for ${resourceAgent.name}` : 'Select resource'}
        wide
      >
        <ResourcePicker agentId={resourceAgent?.id} onSelected={allocateAndStart} />
      </Dialog>
      <ConfirmDialog
        open={Boolean(agentToDelete)}
        title={t('agents.confirmRemoveTitle', 'Remover Agente')}
        description={t('agents.confirmRemove', { name: agentToDelete?.name || agentToDelete?.id })}
        onConfirm={executeRemove}
        onCancel={() => setAgentToDelete(null)}
      />
    </div>
  );
};
