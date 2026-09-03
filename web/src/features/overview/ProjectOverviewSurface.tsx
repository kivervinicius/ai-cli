import React from 'react';
import { Activity, Bot, FolderGit2, Gauge, ShieldCheck, TerminalSquare } from 'lucide-react';
import { Badge, Button, Card, EmptyState, Progress } from '../../design-system';
import type { Agent, Project } from '../../types';
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
  onOpenAgent: (agent: Agent) => void;
  onNewAISession: () => void;
  onProjectShell: () => void;
  onOpenComposer?: () => void;
  onOpenFlow?: () => void;
}> = ({ project, agents, onOpenAgent, onNewAISession, onProjectShell, onOpenComposer, onOpenFlow }) => {
  const { t } = useTranslation();
  const agentList = Array.isArray(agents) ? agents : [];
  const working = agentList.filter((agent) => agent.status === 'WORKING').length;
  const attention = agentList.filter((agent) =>
    ['FAILED', 'STALE', 'RECOVERABLE', 'RATE_LIMITED'].includes(agent.status)
  ).length;

  return (
    <div className="nx-surface-scroll">
      <div className="nx-page-header">
        <div>
          <span className="nx-eyebrow">{t('overview.eyebrow')}</span>
          <h1>{project.name}</h1>
          <p>{project.canonical_path}</p>
        </div>
        <div className="nx-page-header__actions">
          <Button tone="brand" onClick={onNewAISession}>
            <TerminalSquare size={14} /> {t('overview.newAISession')}
          </Button>
          <Button onClick={onProjectShell}>{t('overview.projectShell')}</Button>
          {onOpenComposer ? (
            <Button tone="ghost" onClick={onOpenComposer}>
              {t('overview.openComposer')}
            </Button>
          ) : null}
          {onOpenFlow ? (
            <Button tone="ghost" onClick={onOpenFlow}>
              {t('overview.openFlow')}
            </Button>
          ) : null}
        </div>
      </div>
      <div className="nx-metric-grid">
        <Card className="nx-metric-card">
          <span className="nx-metric-card__icon">
            <Bot size={18} />
          </span>
          <div>
            <strong>{agentList.length}</strong>
            <span>{t('overview.persistentAgents')}</span>
          </div>
        </Card>
        <Card className="nx-metric-card">
          <span className="nx-metric-card__icon">
            <Activity size={18} />
          </span>
          <div>
            <strong>{working}</strong>
            <span>{t('overview.workingNow')}</span>
          </div>
        </Card>
        <Card className="nx-metric-card">
          <span className="nx-metric-card__icon">
            <Gauge size={18} />
          </span>
          <div>
            <strong>{project.resource_policy || 'BALANCED'}</strong>
            <span>{t('overview.resourcePolicy')}</span>
          </div>
        </Card>
        <Card className="nx-metric-card">
          <span className="nx-metric-card__icon">
            <ShieldCheck size={18} />
          </span>
          <div>
            <strong>{project.maestro_mode || 'ASSIST'}</strong>
            <span>{t('overview.maestroMode')}</span>
          </div>
        </Card>
      </div>
      <div className="nx-overview-columns">
        <Card className="nx-overview-card">
          <div className="nx-section-title">
            <div>
              <h2>{t('overview.fleet')}</h2>
              <p>{t('overview.fleetDescription')}</p>
            </div>
            <Badge tone={attention > 0 ? 'warning' : 'success'}>
              {attention > 0 ? t('overview.needsAttention', { count: attention }) : t('overview.healthy')}
            </Badge>
          </div>
          {agentList.length === 0 ? (
            <EmptyState icon={<Bot size={20} />} title={t('overview.noAgents')} hint={t('overview.noAgentsHint')} />
          ) : (
            <div className="nx-agent-mini-list">
              {agentList.slice(0, 8).map((agent) => (
                <button className="nx-agent-mini" key={agent.id} onClick={() => onOpenAgent(agent)}>
                  <span className="nx-agent-avatar">{(agent.name || 'AG').slice(0, 2).toUpperCase()}</span>
                  <span className="nx-agent-mini__copy">
                    <strong>{agent.name || agent.id}</strong>
                    <small>
                      {agent.role || t('overview.developmentAgent')} ·{' '}
                      {agent.continuity_status || t('overview.continuityPending')}
                    </small>
                  </span>
                  <Badge tone={tone(agent.status)}>{translateStatus(agent.status)}</Badge>
                </button>
              ))}
            </div>
          )}
        </Card>
        <Card className="nx-overview-card">
          <div className="nx-section-title">
            <div>
              <h2>{t('overview.readiness')}</h2>
              <p>{t('overview.readinessDescription')}</p>
            </div>
          </div>
          <div className="nx-readiness-list">
            <div>
              <span>
                <FolderGit2 size={14} /> {t('overview.repository')}
              </span>
              <strong>{project.default_branch || 'main'}</strong>
            </div>
            <div>
              <span>Maestro</span>
              <strong>{project.maestro_mode || 'ASSIST'}</strong>
            </div>
            <div>
              <span>{t('overview.isolation')}</span>
              <strong>{project.default_isolation || 'project'}</strong>
            </div>
            <div>
              <span>{t('overview.activeAgents')}</span>
              <strong>{working}</strong>
            </div>
          </div>
          <div className="nx-readiness-progress">
            <Progress
              value={Math.min(
                100,
                40 + (agentList.length ? 25 : 0) + (project.maestro_mode !== 'OFF' ? 20 : 0) + (project.resource_policy ? 15 : 0)
              )}
              label={t('overview.readinessLabel')}
            />
          </div>
        </Card>
      </div>
    </div>
  );
};
