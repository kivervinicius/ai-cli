import React from 'react';
import { Sparkles } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Badge, Card, EmptyState, Spinner } from '../../design-system';
import type { RecommendationResult } from '../../types';
import { asArray } from '../../lib/safeArray';
import styles from './UsageRecommendRail.module.scss';

export const UsageRecommendRail: React.FC<{
  loading: boolean;
  result: RecommendationResult | null;
  error?: string;
}> = ({ loading, result, error }) => {
  const { t } = useTranslation();
  const recommended = result?.recommended;
  const candidates = asArray<RecommendationResult['candidates'][number]>(result?.candidates).slice(
    0,
    4,
  );

  return (
    <Card className={styles.card}>
      <header className={styles.header}>
        <Sparkles size={15} />
        <strong>{t('usage.recommendTitle', 'Recomendação do scheduler')}</strong>
      </header>
      {loading ? <Spinner label={t('usage.recommendLoading', 'Avaliando contas…')} /> : null}
      {!loading && error ? <p className={styles.error}>{error}</p> : null}
      {!loading && !error && !recommended ? (
        <EmptyState
          title={t('usage.recommendEmpty', 'Sem recomendação')}
          hint={t('usage.recommendEmptyHint', 'Nenhuma conta elegível para a política atual.')}
        />
      ) : null}
      {!loading && recommended ? (
        <div className={styles.recommended}>
          <div className={styles.row}>
            <strong>
              {recommended.account.display_name ||
                `${recommended.account.provider}:${recommended.account.profile}`}
            </strong>
            <Badge tone="brand">#{recommended.rank}</Badge>
            <Badge>{recommended.confidence}</Badge>
          </div>
          <p>{result?.explanation}</p>
          {asArray<string>(recommended.pros).length > 0 ? (
            <ul>
              {asArray<string>(recommended.pros)
                .slice(0, 3)
                .map((item) => (
                  <li key={item}>{item}</li>
                ))}
            </ul>
          ) : null}
          {asArray<string>(recommended.cons).length > 0 ? (
            <ul className={styles.cons}>
              {asArray<string>(recommended.cons)
                .slice(0, 2)
                .map((item) => (
                  <li key={item}>{item}</li>
                ))}
            </ul>
          ) : null}
        </div>
      ) : null}
      {!loading && candidates.length > 1 ? (
        <div className={styles.candidates}>
          <small>{t('usage.otherCandidates', 'Outros candidatos')}</small>
          {candidates
            .filter((item) => item.account.id !== recommended?.account.id)
            .map((item) => (
              <div key={item.account.id} className={styles.candidate}>
                <span>
                  #{item.rank} {item.account.display_name || item.account.profile}
                </span>
                <Badge tone={item.eligible ? 'success' : 'danger'}>
                  {item.eligible
                    ? t('usage.eligible', 'Elegível')
                    : item.rejection_reason || t('usage.ineligible', 'Inelegível')}
                </Badge>
              </div>
            ))}
        </div>
      ) : null}
    </Card>
  );
};
