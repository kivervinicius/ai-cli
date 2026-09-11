import React from 'react';
import { useTranslation } from 'react-i18next';
import type { UsageSummary } from './usageModel';
import styles from './UsageSummaryStrip.module.scss';

export const UsageSummaryStrip: React.FC<{
  summary: UsageSummary;
  bestLabel?: string | null;
}> = ({ summary, bestLabel }) => {
  const { t } = useTranslation();
  return (
    <div className={styles.strip} role="group" aria-label={t('usage.summary', 'Resumo da frota')}>
      <div className={styles.metric}>
        <strong>{summary.usable}</strong>
        <span>{t('usage.usable', 'Utilizáveis')}</span>
      </div>
      <div className={styles.metric}>
        <strong>{summary.exhausted}</strong>
        <span>{t('usage.exhausted', 'Esgotadas')}</span>
      </div>
      <div className={styles.metric}>
        <strong>{summary.unknown}</strong>
        <span>{t('usage.unknown', 'Desconhecidas')}</span>
      </div>
      <div className={styles.metric}>
        <strong>{summary.blocked}</strong>
        <span>{t('usage.blocked', 'Rate limit')}</span>
      </div>
      {bestLabel ? (
        <div className={styles.best}>
          <small>{t('usage.bestChoice', 'Melhor escolha')}</small>
          <strong>{bestLabel}</strong>
        </div>
      ) : null}
    </div>
  );
};
