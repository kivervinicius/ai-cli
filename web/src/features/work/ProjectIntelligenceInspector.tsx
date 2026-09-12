import React, { useCallback, useEffect, useState } from 'react';
import { ChevronDown, ChevronRight, RefreshCw } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Badge, Button, Card } from '../../design-system';
import { asArray } from '../../lib/safeArray';
import { nexus } from '../../nexus/api';
import type { ProjectFact, ProjectFactProvenance, ProjectIntelligenceView } from '../../types';
import styles from './ProjectIntelligenceInspector.module.scss';

const toneForScan = (state?: string) => {
  if (state === 'SUCCEEDED') return 'success';
  if (state === 'FAILED' || state === 'CANCELED') return 'danger';
  if (state === 'RUNNING') return 'warning';
  return 'default';
};

export const ProjectIntelligenceInspector: React.FC<{ projectId: string }> = ({ projectId }) => {
  const { t } = useTranslation();
  const [view, setView] = useState<ProjectIntelligenceView | null>(null);
  const [expanded, setExpanded] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const refresh = useCallback(async () => {
    try {
      setError('');
      setView(await nexus.getProjectIntelligence(projectId));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }, [projectId]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const requestScan = async () => {
    setBusy(true);
    setError('');
    try {
      await nexus.requestProjectIntelligenceScan(projectId);
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const snapshot = view?.current_snapshot;
  const scan = view?.current_scan;
  const facts = asArray<ProjectFact>(snapshot?.facts);
  const warnings = asArray<string>(snapshot?.warnings);

  return (
    <Card className={styles.inspector}>
      <div className={styles.header}>
        <button
          type="button"
          className={styles.toggle}
          aria-expanded={expanded}
          onClick={() => setExpanded((current) => !current)}
        >
          {expanded ? (
            <ChevronDown size={15} aria-hidden="true" />
          ) : (
            <ChevronRight size={15} aria-hidden="true" />
          )}
          <span>{t('work.projectIntelligence.title', 'Project Intelligence')}</span>
        </button>
        <div className={styles.actions}>
          <Badge
            tone={
              snapshot ? (snapshot.completeness === 'COMPLETE' ? 'success' : 'warning') : 'default'
            }
          >
            {snapshot
              ? snapshot.completeness
              : t('work.projectIntelligence.notReady', 'Not analyzed')}
          </Badge>
          <Button size="sm" tone="ghost" disabled={busy} onClick={() => void requestScan()}>
            <RefreshCw size={13} aria-hidden="true" />
            {busy
              ? t('work.projectIntelligence.scanning', 'Analyzing…')
              : t('work.projectIntelligence.refresh', 'Analyze again')}
          </Button>
        </div>
      </div>
      {expanded && (
        <div className={styles.body}>
          {error && <p className={styles.error}>{error}</p>}
          <div className={styles.meta}>
            <span>{t('work.projectIntelligence.identity', 'Identity')}</span>
            <code>{view?.identity.identity_digest || '—'}</code>
          </div>
          {scan && (
            <div className={styles.meta}>
              <span>{t('work.projectIntelligence.scan', 'Scan')}</span>
              <Badge tone={toneForScan(scan.state)}>{scan.state}</Badge>
            </div>
          )}
          {warnings.length ? (
            <p className={styles.warning}>
              {t('work.projectIntelligence.warnings', '{{count}} warnings', {
                count: warnings.length,
              })}
            </p>
          ) : null}
          <div className={styles.facts} aria-live="polite">
            {facts.length === 0 ? (
              <p>{t('work.projectIntelligence.empty', 'No structured facts available yet.')}</p>
            ) : (
              facts.map((fact: ProjectFact) => (
                <details key={`${fact.category}:${fact.key}`} className={styles.fact}>
                  <summary>{fact.key}</summary>
                  <div className={styles.factBody}>
                    <code>{JSON.stringify(fact.value)}</code>
                    <span>
                      {fact.confidence} · {fact.basis}
                    </span>
                    {asArray<ProjectFactProvenance>(fact.provenance).map((source) => (
                      <span key={`${source.source}:${source.locator || ''}`}>
                        {source.source}
                        {source.locator ? ` · ${source.locator}` : ''}
                      </span>
                    ))}
                  </div>
                </details>
              ))
            )}
          </div>
        </div>
      )}
    </Card>
  );
};
