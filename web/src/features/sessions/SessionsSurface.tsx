import React, { useEffect, useState } from 'react';
import { GitBranch, History } from 'lucide-react';
import { Badge, Card, EmptyState, Spinner } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { Agent, AgentDetail } from '../../types';
import { useTranslation } from 'react-i18next';

export const SessionsSurface: React.FC<{ agents: Agent[] }> = ({ agents }) => {
  const { t } = useTranslation();
  const [details, setDetails] = useState<AgentDetail[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all(agents.map((agent) => nexus.getAgent(agent.id).catch(() => null)))
      .then((values) => setDetails(values.filter(Boolean) as AgentDetail[]))
      .finally(() => setLoading(false));
  }, [agents]);

  if (loading) {
    return (
      <div className="nx-surface-center">
        <Spinner label={t('sessions.loading')} />
      </div>
    );
  }

  return (
    <div className="nx-surface-scroll">
      <div className="nx-page-header">
        <div>
          <span className="nx-eyebrow">{t('sessions.eyebrow')}</span>
          <h1>{t('sessions.title')}</h1>
          <p>{t('sessions.intro')}</p>
        </div>
      </div>
      {details.length === 0 ? (
        <EmptyState
          icon={<History size={22} />}
          title={t('sessions.empty')}
          hint={t('sessions.emptyHint')}
        />
      ) : (
        <div className="nx-session-list">
          {details.map((detail) => {
            const generations = Array.isArray(detail.generations) ? detail.generations : [];
            return (
              <Card key={detail.agent.id} className="nx-session-card">
                <div className="nx-session-card__head">
                  <div>
                    <strong>{detail.agent.name}</strong>
                    <small>{detail.agent.id}</small>
                  </div>
                  <Badge>{detail.agent.continuity_status}</Badge>
                </div>
                <div className="nx-generation-list">
                  {generations.length === 0 ? (
                    <span className="nx-muted-copy">{t('sessions.noGenerations')}</span>
                  ) : (
                    generations.map((generation) => (
                      <div key={generation.id}>
                        <GitBranch size={13} />
                        <div>
                          <strong>
                            {generation.provider}/{generation.profile}
                          </strong>
                          <small>
                            {generation.runtime_id} ·{' '}
                            {generation.provider_session || t('sessions.unknown')}
                          </small>
                        </div>
                        <Badge>{generation.state}</Badge>
                        <Badge
                          tone={
                            generation.continuity === 'NATIVE_RESUME_VERIFIED'
                              ? 'success'
                              : 'default'
                          }
                        >
                          {generation.continuity}
                        </Badge>
                      </div>
                    ))
                  )}
                </div>
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
};
