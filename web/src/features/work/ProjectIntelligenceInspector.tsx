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
  if (state === 'RUNNING' || state === 'QUEUED') return 'warning';
  return 'default';
};

const isScanInFlight = (state?: string) => state === 'QUEUED' || state === 'RUNNING';

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

  const scanState = view?.current_scan?.state;
  useEffect(() => {
    if (!isScanInFlight(scanState)) {
      return;
    }
    const timer = window.setInterval(() => {
      void refresh();
    }, 1500);
    return () => window.clearInterval(timer);
  }, [refresh, scanState]);

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
  const scanning = busy || isScanInFlight(scan?.state);
  const completenessLabel = snapshot
    ? t(`work.projectIntelligence.completeness.${snapshot.completeness}`, snapshot.completeness)
    : t('work.projectIntelligence.notReady');
  const scanLabel = scan ? t(`work.projectIntelligence.scanState.${scan.state}`, scan.state) : '';

  return (
    <Card className={styles.inspector}>
      <div className={styles.header}>
        <button
          type="button"
          className={styles.toggle}
          aria-expanded={expanded}
          aria-controls="project-intelligence-panel"
          onClick={() => setExpanded((current) => !current)}
        >
          {expanded ? (
            <ChevronDown size={15} aria-hidden="true" />
          ) : (
            <ChevronRight size={15} aria-hidden="true" />
          )}
          <span>{t('work.projectIntelligence.title')}</span>
        </button>
        <div className={styles.actions}>
          <Badge
            tone={
              snapshot ? (snapshot.completeness === 'COMPLETE' ? 'success' : 'warning') : 'default'
            }
          >
            {completenessLabel}
          </Badge>
          <Button size="sm" tone="ghost" disabled={scanning} onClick={() => void requestScan()}>
            <RefreshCw size={13} aria-hidden="true" />
            {scanning
              ? t('work.projectIntelligence.scanning')
              : t('work.projectIntelligence.refresh')}
          </Button>
        </div>
      </div>
      {(error || scan?.error) && !expanded ? (
        <p className={styles.error} role="status">
          {error || scan?.error}
        </p>
      ) : null}
      {expanded && (
        <div className={styles.body} id="project-intelligence-panel">
          {error && <p className={styles.error}>{error}</p>}
          <div className={styles.meta}>
            <span>{t('work.projectIntelligence.identity')}</span>
            <code>{view?.identity.identity_digest || '—'}</code>
          </div>
          {scan && (
            <div className={styles.meta}>
              <span>{t('work.projectIntelligence.scan')}</span>
              <Badge tone={toneForScan(scan.state)}>{scanLabel}</Badge>
            </div>
          )}
          {scan?.error ? <p className={styles.error}>{scan.error}</p> : null}
          {warnings.length ? (
            <p className={styles.warning}>
              {t('work.projectIntelligence.warnings', {
                count: warnings.length,
              })}
            </p>
          ) : null}
          <div className={styles.facts} aria-live="polite">
            {facts.length === 0 ? (
              <p>{t('work.projectIntelligence.empty')}</p>
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
