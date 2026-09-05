import React, { useCallback, useEffect, useState } from 'react';
import { Network, RefreshCw, Sparkles } from 'lucide-react';
import { Badge, Button, Card } from '../../design-system';
import type {
  Agent,
  ContextReadiness,
  ContextReadinessState,
  MissionRun,
  Project,
  PromptArtifact,
  WorkPlan,
} from '../../types';
import { nexus } from '../../nexus/api';
import { PlanBuilderSurface } from './PlanBuilderSurface';
import { ComposerSurface } from './ComposerSurface';
import { composerGateForReadiness } from './composerModel';

const readinessTone = (state: ContextReadinessState) =>
  state === 'READY'
    ? 'success'
    : state === 'FAILED'
      ? 'danger'
      : state === 'STALE'
        ? 'warning'
        : 'default';

/**
 * Composer is the goal bar for Flow — not a second IDE and not a send-to-agent surface.
 * PlanBuilder owns Generate / canvas / inspector / Approve & Run.
 */
export const WorkSurface: React.FC<{
  project: Project;
  agents: Agent[];
  onDirect: (agent: Agent) => void;
  onFlowRun?: (run: MissionRun) => void;
}> = ({ project, agents, onDirect, onFlowRun }) => {
  const [readiness, setReadiness] = useState<ContextReadiness | null>(null);
  const [readinessBusy, setReadinessBusy] = useState(false);
  const [error, setError] = useState('');
  const [intelligence, setIntelligence] = useState<{
    available: boolean;
    provider?: string;
    error?: string;
  } | null>(null);
  const [flowArtifact, setFlowArtifact] = useState<PromptArtifact | null>(null);
  const [flowPlan, setFlowPlan] = useState<WorkPlan | null>(null);
  const [flowError, setFlowError] = useState('');

  const refreshContext = useCallback(async () => {
    try {
      setReadiness(await nexus.getContextReadiness(project.id));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }, [project.id]);

  const refreshIntelligence = useCallback(async () => {
    try {
      const status = await nexus.getIntelligence(project.id);
      setIntelligence({
        available: status.available,
        provider: status.provider || status.mode,
        error: status.error,
      });
    } catch (err) {
      setIntelligence({
        available: false,
        error: err instanceof Error ? err.message : String(err),
      });
    }
  }, [project.id]);

  useEffect(() => {
    void refreshContext();
    void refreshIntelligence();
  }, [refreshContext, refreshIntelligence]);

  const state: ContextReadinessState = readiness?.state ?? 'MISSING';
  const gate = composerGateForReadiness(state);
  const intelligenceTone = intelligence?.available ? 'success' : 'warning';

  const prepareContext = async (createContext = false) => {
    setReadinessBusy(true);
    setError('');
    try {
      setReadiness(await nexus.prepareContext(project.id, createContext));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setReadinessBusy(false);
    }
  };

  const materializeFlow = async (artifact: PromptArtifact) => {
    setFlowArtifact(artifact);
    setFlowError('');
    try {
      setFlowPlan(await nexus.materializePromptArtifact(artifact.id));
    } catch (err) {
      setFlowError(err instanceof Error ? err.message : String(err));
    }
  };

  return (
    <div className="nx-surface-scroll nx-composer-surface nx-composer-workbench">
      <div className="nx-page-header">
        <div>
          <span className="nx-eyebrow">
            <Sparkles size={13} /> COMPOSER
          </span>
          <h1>Sua solicitação</h1>
          <p>
            Descreva o objetivo. O Flow embaixo nasce ao gerar o rascunho. Trabalho direto no agente
            fica no terminal.
          </p>
        </div>
        <div className="nx-composer-header-actions">
          <Badge tone={readinessTone(state)}>Context {state}</Badge>
          <Badge tone={intelligenceTone}>
            Intelligence{' '}
            {intelligence?.available
              ? `READY${intelligence.provider ? ` · ${intelligence.provider}` : ''}`
              : 'OFF'}
          </Badge>
        </div>
      </div>

      {!gate.canCompose && (
        <Card className="nx-context-readiness-card" data-state={state}>
          <div className="nx-context-readiness-card__status">
            <Network size={17} />
            <div>
              <strong>Context Readiness · {state}</strong>
              <p>{gate.reason}</p>
              {readiness?.error && <small>{readiness.error}</small>}
            </div>
          </div>
          {gate.action !== 'WAIT' && (
            <div style={{ display: 'grid', gap: '8px', justifyItems: 'start' }}>
              {(gate.action === 'PREPARE' ||
                (gate.action === 'RETRY' &&
                  readiness?.error?.includes('durable project context is missing'))) && (
                <small style={{ color: 'var(--nx-muted)', maxWidth: '560px' }}>
                  Este projeto não possui contexto durável. Ao continuar, o Nexus criará um{' '}
                  <code>AGENTS.md</code> base na raiz, sem sobrescrever arquivos existentes.
                </small>
              )}
              <Button
                size="sm"
                tone="brand"
                disabled={readinessBusy}
                onClick={() =>
                  void prepareContext(
                    gate.action === 'PREPARE' ||
                      Boolean(readiness?.error?.includes('durable project context is missing')),
                  )
                }
              >
                <RefreshCw size={13} />{' '}
                {readinessBusy
                  ? 'Checking…'
                  : gate.action === 'PREPARE'
                    ? 'Criar contexto base'
                    : 'Refresh Context'}
              </Button>
            </div>
          )}
        </Card>
      )}

      {error && <div className="nx-inline-error">{error}</div>}

      <div className="nx-composer-flow-region" data-gate={gate.canCompose ? 'ready' : 'blocked'}>
        <ComposerSurface
          project={project}
          onTransformFlow={(artifact) => void materializeFlow(artifact)}
        />
        {flowError && (
          <div className="nx-inline-error">Flow materialization failed: {flowError}</div>
        )}
        {flowArtifact && flowPlan && (
          <PlanBuilderSurface
            project={project}
            agents={agents}
            onOpenAgent={onDirect}
            onRunCreated={onFlowRun}
            initialPlan={flowPlan}
            compactGoal
          />
        )}
      </div>
    </div>
  );
};
