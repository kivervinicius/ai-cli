import React, { useState } from 'react';
import { Play, RotateCcw, TerminalSquare } from 'lucide-react';
import { Badge, Button, Card, Dialog, EmptyState } from '../../design-system';
import { nexus } from '../../nexus/api';
import { ResourcePicker } from '../../nexus/ResourcePicker';
import type { Agent, Project, RuntimeSession } from '../../types';
import { translateStatus } from '../../i18n';
import { useTranslation } from 'react-i18next';

const tone = (status: string) =>
  status === 'WORKING'
    ? 'success'
    : status === 'FAILED' || status === 'STALE'
      ? 'danger'
      : status === 'RECOVERABLE' || status === 'WAITING'
        ? 'warning'
        : 'default';

export const ProjectOverviewSurface: React.FC<{
  project: Project;
  agents: Agent[];
  onOpenAgent: (agent: Agent, runtimeId?: string) => void;
  onNewAISession: () => void;
  onProjectShell: () => void;
  onOpenComposer?: () => void;
  onOpenFlow?: () => void;
  onNewAgent?: () => void;
  onConfigureAgent?: (agent: Agent) => void;
  refreshAgents?: () => Promise<void>;
  onStartAgent?: (agent: Agent) => Promise<{ runtime?: RuntimeSession } | void>;
  onRecoverAgent?: (agent: Agent) => Promise<{ runtime?: RuntimeSession } | void>;
}> = ({
  project,
  agents,
  onOpenAgent,
  onNewAISession,
  onProjectShell,
  onOpenComposer,
  onOpenFlow: _onOpenFlow,
  onNewAgent,
  onConfigureAgent,
  refreshAgents,
  onStartAgent,
  onRecoverAgent,
}) => {
  const { t } = useTranslation();
  const [busy, setBusy] = useState<string>('');
  const [error, setError] = useState<string>('');
  const [resourceAgent, setResourceAgent] = useState<Agent | null>(null);

  const agentList = Array.isArray(agents) ? agents : [];
  const working = agentList.filter((agent) => agent.status === 'WORKING').length;
  const degraded = agentList.filter((agent) =>
    ['FAILED', 'STALE', 'RECOVERABLE', 'RATE_LIMITED'].includes(agent.status)
  ).length;

  const handleRecover = async (agent: Agent) => {
    setBusy(agent.id);
    setError('');
    try {
      const res = await (onRecoverAgent ? onRecoverAgent(agent) : nexus.recoverAgent(agent.id));
      await refreshAgents?.();
      const runtimeId = res && typeof res === 'object' && 'runtime' in res ? (res as any).runtime?.runtime_id : undefined;
      onOpenAgent(agent, runtimeId);
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      if (message.includes('REQUIRED_RESOURCE_SELECTION')) {
        setResourceAgent(agent);
      } else {
        setError(message);
      }
    } finally {
      setBusy('');
    }
  };

  const handleStart = async (agent: Agent) => {
    setBusy(agent.id);
    setError('');
    try {
      const res = await (onStartAgent ? onStartAgent(agent) : nexus.startAgent(agent.id));
      await refreshAgents?.();
      const runtimeId = res && typeof res === 'object' && 'runtime' in res ? (res as any).runtime?.runtime_id : undefined;
      onOpenAgent(agent, runtimeId);
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      if (message.includes('REQUIRED_RESOURCE_SELECTION')) {
        setResourceAgent(agent);
      } else {
        setError(message);
      }
    } finally {
      setBusy('');
    }
  };

  const allocateAndStart = async () => {
    if (!resourceAgent) return;
    setBusy(resourceAgent.id);
    setError('');
    try {
      const res = await nexus.startAgent(resourceAgent.id);
      await refreshAgents?.();
      setResourceAgent(null);
      onOpenAgent(resourceAgent, res?.runtime?.runtime_id);
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
          <span className="nx-eyebrow">Workspace OS · Terminais do Projeto</span>
          <h1>{project.name}</h1>
          <p>{project.canonical_path} · <code style={{ color: 'var(--nx-accent-text)' }}>{project.default_branch || 'main'}</code></p>
        </div>
        <div className="nx-page-header__actions">
          {onOpenComposer ? (
            <Button tone="ghost" onClick={onOpenComposer}>
              {t('overview.openComposer')}
            </Button>
          ) : null}
        </div>
      </div>

      <div style={{ display: 'grid', gap: '16px' }}>
        <div className="nx-section-title">
          <div>
            <h2>{t('overview.fleet')} ({agentList.length})</h2>
            <p>{t('overview.fleetDescription')}</p>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <Badge tone={degraded > 0 ? 'warning' : 'success'}>
              {degraded > 0
                ? t('overview.degraded', { count: degraded, defaultValue: `${degraded} degradados` })
                : t('overview.healthy')}
            </Badge>
            {working > 0 && (
              <Badge tone="success">
                {working} em execução
              </Badge>
            )}
          </div>
        </div>

        {error && <Card className="nx-inline-error">{error}</Card>}

        {agentList.length === 0 ? (
          <EmptyState
            icon={<TerminalSquare size={36} />}
            title={t('overview.noAgents')}
            hint={t('overview.noAgentsHint')}
            action={(
              <div className="nx-project-create-actions" data-size="md" style={{ marginTop: 4, justifyContent: 'center' }}>
                {onNewAgent && (
                  <Button tone="brand" onClick={onNewAgent}>Novo Agente</Button>
                )}
                <Button onClick={onNewAISession}>{t('overview.newAISession')}</Button>
                <Button onClick={onProjectShell}>{t('overview.projectShell')}</Button>
              </div>
            )}
          />
        ) : (
          <div className="nx-agent-grid">
            {agentList.map((agent) => (
              <Card key={agent.id} className="nx-agent-card" style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
                <div className="nx-agent-card__head">
                  <span className="nx-agent-avatar nx-agent-avatar--large">
                    {(agent.name || 'AG').slice(0, 2).toUpperCase()}
                  </span>
                  <div style={{ minWidth: 0, overflow: 'hidden' }}>
                    <strong style={{ fontSize: '14px', display: 'block', textOverflow: 'ellipsis', overflow: 'hidden', whiteSpace: 'nowrap' }}>
                      {agent.name || agent.id}
                    </strong>
                    <small style={{ color: 'var(--nx-muted)', fontSize: '11.5px' }}>
                      {agent.role || t('overview.developmentAgent')} · {agent.id}
                    </small>
                  </div>
                  <Badge tone={tone(agent.status)}>
                    {agent.status === 'RECOVERABLE'
                      ? t('overview.runtimeStopped', 'Runtime parado')
                      : translateStatus(agent.status)}
                  </Badge>
                </div>

                <div className="nx-agent-card__meta">
                  <span>{t('agents.continuity')}</span>
                  <strong>{agent.continuity_status || t('common.unknown')}</strong>
                  <span>{t('agents.lastStart')}</span>
                  <strong>{agent.last_started_at || t('common.never')}</strong>
                </div>

                <div className="nx-agent-card__actions" style={{ marginTop: 'auto', paddingTop: '8px', borderTop: '1px solid var(--nx-border)' }}>
                  <Button size="sm" tone="brand" onClick={() => onOpenAgent(agent)}>
                    <TerminalSquare size={13} /> {t('overview.openTerminal', 'Abrir Terminal')}
                  </Button>
                  {agent.status === 'RECOVERABLE' ? (
                    <Button
                      size="sm"
                      tone="warning"
                      disabled={busy === agent.id}
                      onClick={(e) => {
                        e.stopPropagation();
                        void handleRecover(agent);
                      }}
                    >
                      <RotateCcw size={12} />{' '}
                      {busy === agent.id ? t('overview.recovering', 'Recuperando…') : t('overview.recover', 'Recuperar')}
                    </Button>
                  ) : agent.status === 'STOPPED' ? (
                    <Button
                      size="sm"
                      tone="brand"
                      disabled={busy === agent.id}
                      onClick={(e) => {
                        e.stopPropagation();
                        void handleStart(agent);
                      }}
                    >
                      <Play size={12} />{' '}
                      {busy === agent.id ? t('overview.starting', 'Iniciando…') : t('overview.start', 'Iniciar')}
                    </Button>
                  ) : null}
                  {onConfigureAgent && (
                    <Button size="sm" tone="ghost" onClick={() => onConfigureAgent(agent)}>
                      {t('agents.configure', 'Configurar')}
                    </Button>
                  )}
                </div>
              </Card>
            ))}
          </div>
        )}
      </div>

      <Dialog
        open={!!resourceAgent}
        onClose={() => setResourceAgent(null)}
        title={resourceAgent ? `Select resource for ${resourceAgent.name}` : 'Select resource'}
        wide
      >
        <ResourcePicker agentId={resourceAgent?.id} onSelected={allocateAndStart} />
      </Dialog>
    </div>
  );
};
