import React, { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { AlertTriangle, CheckCircle2, Clock, XCircle } from 'lucide-react';
import { nexus } from '../../nexus/api';
import type { AttentionGroup, AttentionItem, HumanIntervention } from '../../types';
import { sanitizeAttentionText } from '../../components/attentionText';
import styles from './AttentionCenter.module.scss';

function formatAge(seconds: number): string {
  if (seconds < 60) return `${Math.round(seconds)}s`;
  if (seconds < 3600) return `${Math.round(seconds / 60)}m`;
  return `${Math.round(seconds / 3600)}h`;
}

const STATE_LABEL_KEYS: Record<string, string> = {
  BLOCKED_NEEDS_USER: 'attention.state.blockedNeedsUser',
  FAILED_NO_PROGRESS: 'attention.state.failedNoProgress',
  FAILED_VERIFICATION: 'attention.state.failedVerification',
  FAILED_BUDGET_EXCEEDED: 'attention.state.failedBudgetExceeded',
  FAILED: 'attention.state.failed',
  COMPLETED_VERIFIED: 'attention.state.completedVerified',
  PAUSED: 'attention.state.paused',
};

function stateLabel(state: string, t: (key: string) => string): string {
  const key = STATE_LABEL_KEYS[state];
  if (key) return t(key);
  return state;
}

function InterventionCard({
  item,
  onResolve,
}: {
  item: AttentionItem;
  onResolve: (runId: string, intervention: HumanIntervention, optionId: string) => void;
}) {
  const { t } = useTranslation();
  const intervention = item.intervention;
  if (!intervention) return null;

  return (
    <div className={styles['attention-center__resolve']} role="article">
      <div className={styles['attention-center__resolve-question']}>
        {sanitizeAttentionText(
          intervention.question,
          t('attention.center.decisionRequired', 'Decision required'),
        )}
      </div>
      {intervention.context && (
        <div className={styles['attention-center__resolve-context']}>
          {sanitizeAttentionText(intervention.context, '')}
        </div>
      )}
      {intervention.impact && (
        <div className={styles['attention-center__resolve-context']}>
          {t('attention.center.impactLabel', 'Impact:') + ' '}
          {sanitizeAttentionText(intervention.impact, '')}
        </div>
      )}
      <div className={styles['attention-center__resolve-actions']}>
        {(intervention.options || []).map((option) => (
          <button
            key={option.id}
            type="button"
            className="nx-button nx-button--sm nx-button--brand"
            onClick={() => onResolve(item.mission_id, intervention, option.id)}
          >
            {sanitizeAttentionText(option.label, option.id)}
          </button>
        ))}
        {(intervention.options || []).length === 0 && (
          <span>{t('attention.center.noOptions', 'No safe action available')}</span>
        )}
      </div>
    </div>
  );
}

export const AttentionCenter: React.FC<{
  onNavigateToMission?: (missionId: string) => void;
}> = ({ onNavigateToMission }) => {
  const { t } = useTranslation();
  const [group, setGroup] = useState<AttentionGroup | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [resolving, setResolving] = useState<string | null>(null);

  const fetchAttention = useCallback(async () => {
    try {
      const data = await nexus.getAttention();
      setGroup(data);
      setError(null);
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : t('attention.center.fetchError', 'Failed to load attention'),
      );
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    void fetchAttention();
    const interval = setInterval(() => void fetchAttention(), 10_000);
    return () => clearInterval(interval);
  }, [fetchAttention]);

  const handleResolve = useCallback(
    async (runId: string, intervention: HumanIntervention, optionId: string) => {
      setResolving(runId);
      try {
        await nexus.resolveIntervention(runId, intervention.id, intervention.version, optionId);
        await fetchAttention();
      } catch (err) {
        setError(
          err instanceof Error
            ? err.message
            : t('attention.center.resolveError', 'Failed to resolve'),
        );
      } finally {
        setResolving(null);
      }
    },
    [fetchAttention, t],
  );

  const handleClickItem = useCallback(
    (item: AttentionItem) => {
      if (onNavigateToMission) onNavigateToMission(item.mission_id);
    },
    [onNavigateToMission],
  );

  if (loading) {
    return (
      <div className={styles['attention-center']}>
        <div className={styles['attention-center__empty']}>
          <Clock size={24} />
          <p>{t('common.loading', 'Loading...')}</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className={styles['attention-center']}>
        <div className={styles['attention-center__empty']}>
          <XCircle size={24} />
          <p>{error}</p>
        </div>
      </div>
    );
  }

  const needsYou = group?.needs_you || [];
  const completed = group?.completed || [];
  const failed = group?.failed || [];

  if (needsYou.length === 0 && completed.length === 0 && failed.length === 0) {
    return (
      <div className={styles['attention-center']}>
        <div className={styles['attention-center__header']}>
          <span className={styles['attention-center__title']}>
            {t('attention.center.title', 'Attention Center')}
          </span>
        </div>
        <div className={styles['attention-center__empty']}>
          <CheckCircle2 size={24} />
          <p>{t('attention.center.allClear', 'All missions running smoothly.')}</p>
        </div>
      </div>
    );
  }

  return (
    <div
      className={styles['attention-center']}
      role="region"
      aria-label={t('attention.center.title', 'Attention Center')}
    >
      <div className={styles['attention-center__header']}>
        <span className={styles['attention-center__title']}>
          {t('attention.center.title', 'Attention Center')}
        </span>
        {needsYou.length > 0 && (
          <span
            className={styles['attention-center__badge']}
            aria-label={t('attention.center.decisionCount', {
              count: needsYou.length,
              defaultValue: `${needsYou.length} decisions pending`,
            })}
          >
            {needsYou.length}
          </span>
        )}
      </div>

      {needsYou.length > 0 && (
        <div className={styles['attention-center__section']}>
          <div className={styles['attention-center__section-header']}>
            {t('attention.center.needsYou', 'Needs You')}
          </div>
          {needsYou.map((item) => (
            <div key={item.mission_id}>
              <button
                type="button"
                className={styles['attention-center__item']}
                onClick={() => handleClickItem(item)}
              >
                <div
                  className={`${styles['attention-center__item-icon']} ${styles['attention-center__item-icon--needs-you']}`}
                >
                  <AlertTriangle size={14} />
                </div>
                <div className={styles['attention-center__item-body']}>
                  <div className={styles['attention-center__item-title']}>
                    {sanitizeAttentionText(item.summary, stateLabel(item.state, t))}
                  </div>
                  <div className={styles['attention-center__item-meta']}>
                    {item.reason_code && `${item.reason_code} · `}
                    {item.mission_id.slice(0, 12)}
                  </div>
                </div>
                <div className={styles['attention-center__item-age']}>
                  {formatAge(item.age_seconds)}
                </div>
              </button>
              {item.intervention &&
                !item.intervention.resolved &&
                resolving !== item.mission_id && (
                  <InterventionCard item={item} onResolve={handleResolve} />
                )}
              {resolving === item.mission_id && (
                <div className={styles['attention-center__resolve']}>
                  <p>{t('common.resolving', 'Resolving...')}</p>
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {completed.length > 0 && (
        <div className={styles['attention-center__section']}>
          <div className={styles['attention-center__section-header']}>
            {t('attention.center.completed', 'Completed')}
          </div>
          {completed.map((item) => (
            <button
              type="button"
              key={item.mission_id}
              className={styles['attention-center__item']}
              onClick={() => handleClickItem(item)}
            >
              <div
                className={`${styles['attention-center__item-icon']} ${styles['attention-center__item-icon--completed']}`}
              >
                <CheckCircle2 size={14} />
              </div>
              <div className={styles['attention-center__item-body']}>
                <div className={styles['attention-center__item-title']}>
                  {sanitizeAttentionText(item.summary, stateLabel(item.state, t))}
                </div>
                <div className={styles['attention-center__item-meta']}>
                  {item.mission_id.slice(0, 12)}
                </div>
              </div>
              <div className={styles['attention-center__item-age']}>
                {formatAge(item.age_seconds)}
              </div>
            </button>
          ))}
        </div>
      )}

      {failed.length > 0 && (
        <div className={styles['attention-center__section']}>
          <div className={styles['attention-center__section-header']}>
            {t('attention.center.failed', 'Failed / No Progress')}
          </div>
          {failed.map((item) => (
            <button
              type="button"
              key={item.mission_id}
              className={styles['attention-center__item']}
              onClick={() => handleClickItem(item)}
            >
              <div
                className={`${styles['attention-center__item-icon']} ${styles['attention-center__item-icon--failed']}`}
              >
                <XCircle size={14} />
              </div>
              <div className={styles['attention-center__item-body']}>
                <div className={styles['attention-center__item-title']}>
                  {sanitizeAttentionText(item.summary, stateLabel(item.state, t))}
                </div>
                <div className={styles['attention-center__item-meta']}>
                  {item.reason_code && `${item.reason_code} · `}
                  {item.mission_id.slice(0, 12)}
                </div>
              </div>
              <div className={styles['attention-center__item-age']}>
                {formatAge(item.age_seconds)}
              </div>
            </button>
          ))}
        </div>
      )}
    </div>
  );
};
