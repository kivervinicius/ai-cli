import React from 'react';
import { CheckCircle2, RefreshCw, TriangleAlert, XCircle } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Badge, Button, Card, InlineAlert } from '../../design-system';
import type { SystemDoctorCheck, SystemDoctorReport } from '../../nexus/api';
import { asArray } from '../../lib/safeArray';
import styles from './SystemDiagnosticsCard.module.scss';

type Props = {
  report: SystemDoctorReport | null;
  loading: boolean;
  error: string;
  onRefresh: () => void;
};

const statusTone = (status: string): 'success' | 'warning' | 'danger' | 'default' => {
  if (status === 'PASS') return 'success';
  if (status === 'WARN') return 'warning';
  if (status === 'FAIL') return 'danger';
  return 'default';
};

const StatusIcon: React.FC<{ status: string }> = ({ status }) => {
  if (status === 'PASS') return <CheckCircle2 size={14} aria-hidden="true" />;
  if (status === 'FAIL') return <XCircle size={14} aria-hidden="true" />;
  return <TriangleAlert size={14} aria-hidden="true" />;
};

export const SystemDiagnosticsCard: React.FC<Props> = ({ report, loading, error, onRefresh }) => {
  const { t } = useTranslation();

  return (
    <Card className={`${styles.card} nx-settings-card`}>
      <div className="nx-settings-card__title">
        <TriangleAlert size={17} />
        <div>
          <strong>{t('settings.diagnostics')}</strong>
          <small>{t('settings.diagnosticsDescription')}</small>
        </div>
      </div>
      <div className={styles.toolbar}>
        {report ? (
          <span className={styles.meta}>
            {t('settings.diagnosticsPlatform', { os: report.os, arch: report.arch })} · v
            {report.version}
          </span>
        ) : (
          <span className={styles.meta}>{t('settings.diagnosticsUnavailable')}</span>
        )}
        <Button onClick={onRefresh} disabled={loading} size="sm">
          <RefreshCw size={14} className={loading ? 'nx-spin' : undefined} aria-hidden="true" />
          {loading ? t('settings.diagnosticsChecking') : t('settings.diagnosticsRefresh')}
        </Button>
      </div>
      {error && (
        <InlineAlert tone="danger" title={t('settings.diagnosticsError')}>
          {error}
        </InlineAlert>
      )}
      {report && (
        <div className={styles.checks} role="list" aria-label={t('settings.diagnosticsChecks')}>
          {asArray<SystemDoctorCheck>(report.checks).map((check) => (
            <div className={styles.check} role="listitem" key={check.id}>
              <span className={styles.checkId}>{check.id}</span>
              <span className={styles.summary}>{check.summary}</span>
              <Badge tone={statusTone(check.status)}>
                <StatusIcon status={check.status} /> {check.status}
              </Badge>
              {check.remediation && (
                <span className={styles.remediation}>
                  {t('settings.diagnosticsRemediation')}: {check.remediation}
                </span>
              )}
            </div>
          ))}
        </div>
      )}
    </Card>
  );
};
