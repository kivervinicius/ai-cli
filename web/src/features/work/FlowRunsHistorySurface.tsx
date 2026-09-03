import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { History, Play, RefreshCw, Workflow } from 'lucide-react';
import { Badge, Button, Card, EmptyState, Spinner } from '../../design-system';
import { nexusApi } from '../../nexus/api';
import type { MissionRun, Project } from '../../types';
import { asArray } from '../../lib/safeArray';
import { flowRunStateFromMission } from './flowRunModel';

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
        return Boolean(item && typeof item === 'object' && item.id) && (!item.project_id || item.project_id === project.id);
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

  const sorted = useMemo(
    () =>
      [...runs].sort((a, b) => String(b.started_at || b.id).localeCompare(String(a.started_at || a.id))),
    [runs]
  );

  return (
    <div className="nx-surface-scroll">
      <div className="nx-page-header">
        <div>
          <span className="nx-eyebrow">
            <Workflow size={13} /> FLOW RUNS
          </span>
          <h1>Histórico de execuções</h1>
          <p>Abra um run para ver evidências. Para criar um novo Flow, use o Composer.</p>
        </div>
        <div className="nx-composer-header-actions">
          <Button size="sm" onClick={() => void load()}>
            <RefreshCw size={13} /> Atualizar
          </Button>
          {onOpenComposer && (
            <Button size="sm" tone="brand" onClick={onOpenComposer}>
              <Play size={13} /> Composer
            </Button>
          )}
        </div>
      </div>

      {loading && (
        <div style={{ display: 'grid', placeItems: 'center', padding: 40 }}>
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
      <div style={{ display: 'grid', gap: 10 }}>
        {sorted.map((run) => {
          const state = flowRunStateFromMission(run.state || '');
          return (
            <Card key={run.id} className="nx-flow-run-history-card">
              <button
                type="button"
                onClick={() => onOpenRun(run)}
                style={{
                  width: '100%',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  gap: 12,
                  background: 'transparent',
                  border: 0,
                  color: 'inherit',
                  textAlign: 'left',
                  cursor: 'pointer',
                  padding: 4,
                }}
              >
                <div style={{ minWidth: 0, display: 'grid', gap: 4 }}>
                  <strong style={{ display: 'inline-flex', alignItems: 'center', gap: 8 }}>
                    <History size={14} />
                    Flow Run · {(run.id || '').slice(-8)}
                  </strong>
                  <small style={{ color: 'var(--nx-muted)' }}>
                    {run.started_at || 'sem horário'} · plano {(run.plan_id || '').slice(-6) || '—'}
                  </small>
                </div>
                <Badge tone={state === 'COMPLETED' ? 'success' : state === 'FAILED' || state === 'CANCELED' ? 'danger' : 'brand'}>
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
