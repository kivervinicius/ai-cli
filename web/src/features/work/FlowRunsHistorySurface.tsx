import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { History, Play, RefreshCw } from 'lucide-react';
import { Badge, Button, Card, EmptyState, Spinner } from '../../design-system';
import { nexusApi } from '../../nexus/api';
import type { MissionRun, Project } from '../../types';
import { asArray } from '../../lib/safeArray';
import { flowRunStateFromMission } from './flowRunModel';
import styles from './FlowRunsHistorySurface.module.scss';

export const FlowRunsHistorySurface: React.FC<{
  project: Project;
  onOpenRun: (run: MissionRun) => void;
  onOpenComposer?: () => void;
}> = ({ project, onOpenRun, onOpenComposer }) => {
  const [runs, setRuns] = useState<MissionRun[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const list = asArray(await nexusApi.getRuns()).filter((run): run is MissionRun => {
        const item = run as MissionRun;
        return (
          Boolean(item && typeof item === 'object' && item.id) &&
          (!item.project_id || item.project_id === project.id)
        );
      });
      setRuns(list);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setRuns([]);
    } finally {
      setLoading(false);
    }
  }, [project.id]);

  useEffect(() => {
    void load();
  }, [load]);

  const [filterState, setFilterState] = useState<'ALL' | 'ACTIVE' | 'COMPLETED' | 'FAILED'>('ALL');

  const filtered = useMemo(() => {
    return runs.filter((run) => {
      const state = flowRunStateFromMission(run.state || '');
      if (filterState === 'ALL') return true;
      if (filterState === 'ACTIVE')
        return state === 'QUEUED' || state === 'VERIFYING' || state === 'READY';
      if (filterState === 'COMPLETED') return state === 'COMPLETED';
      if (filterState === 'FAILED') return state === 'FAILED' || state === 'CANCELED';
      return true;
    });
  }, [runs, filterState]);

  const sorted = useMemo(
    () =>
      [...filtered].sort((a, b) =>
        String(b.started_at || b.id).localeCompare(String(a.started_at || a.id)),
      ),
    [filtered],
  );

  return (
    <div className="nx-flow-runs-history-surface">
      <div className="nx-flow-runs-history-header">
        <div className="nx-flow-runs-history-title">
          <History size={16} />
          <strong>Execuções de Flow</strong>
          <Badge tone="default">{runs.length}</Badge>
        </div>
        <div className="nx-flow-runs-history-actions">
          <div className="nx-flow-runs-filter-group">
            {(['ALL', 'ACTIVE', 'COMPLETED', 'FAILED'] as const).map((s) => (
              <button
                key={s}
                type="button"
                className={`nx-filter-chip ${filterState === s ? 'active' : ''}`}
                onClick={() => setFilterState(s)}
              >
                {s === 'ALL'
                  ? 'Todos'
                  : s === 'ACTIVE'
                    ? 'Em andamento'
                    : s === 'COMPLETED'
                      ? 'Concluídos'
                      : 'Falhas'}
              </button>
            ))}
          </div>
          <Button size="sm" onClick={() => void load()} disabled={loading}>
            <RefreshCw size={13} className={loading ? 'nx-spin' : ''} />
            Atualizar
          </Button>
          {onOpenComposer && (
            <Button size="sm" tone="brand" onClick={onOpenComposer}>
              <Play size={13} /> Novo Flow
            </Button>
          )}
        </div>
      </div>

      {loading && (
        <div className={styles.loadingContainer}>
          <Spinner />
        </div>
      )}
      {error && <div className="nx-inline-error">{error}</div>}
      {!loading && !error && sorted.length === 0 && (
        <EmptyState
          title="Nenhuma execução ainda"
          hint="Gere e aprove um Flow no Composer para ver o histórico aqui."
          action={
            onOpenComposer ? (
              <Button tone="brand" onClick={onOpenComposer}>
                Abrir Composer
              </Button>
            ) : undefined
          }
        />
      )}
      <div className={styles.runsList}>
        {sorted.map((run) => {
          const state = flowRunStateFromMission(run.state || '');
          return (
            <Card key={run.id} className="nx-flow-run-history-card">
              <button type="button" onClick={() => onOpenRun(run)} className={styles.runButton}>
                <div className={styles.runInfo}>
                  <strong className={styles.runTitle}>
                    <History size={14} />
                    Flow Run · {(run.id || '').slice(-8)}
                  </strong>
                  <small className={styles.runMeta}>
                    {run.started_at || 'sem horário'} · plano {(run.plan_id || '').slice(-6) || '—'}
                  </small>
                </div>
                <Badge
                  tone={
                    state === 'COMPLETED'
                      ? 'success'
                      : state === 'FAILED' || state === 'CANCELED'
                        ? 'danger'
                        : 'brand'
                  }
                >
                  {state}
                </Badge>
              </button>
            </Card>
          );
        })}
      </div>
    </div>
  );
};
