import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  CheckCircle2,
  ClipboardCopy,
  FileText,
  Layers,
  Lightbulb,
  MessageCircle,
  Plus,
  RefreshCw,
  Send,
  Sparkles,
} from 'lucide-react';
import { Badge, Button, Card, Input, Select } from '../../design-system';
import { nexus } from '../../nexus/api';
import type {
  ComposerSession,
  ComposerSessionView,
  PromptArtifact,
  PromptReadinessCheck,
  Agent,
  Project,
} from '../../types';
import { selectResumableComposerSession } from './composerSessionModel';
import { composerNeedsGapConfirmation } from './composerModel';
import type { ComposerGate } from './composerModel';
import { asArray, asStringArray } from '../../lib/safeArray';
import { TaskPreparationDialog } from '../../components/TaskPreparationDialog';
import { askActionForStatus } from '../agents/askAgentModel';
import styles from './ComposerSurface.module.scss';

const ARCHETYPE_LABELS: Record<string, string> = {
  SOFTWARE_FEATURE: 'Feature',
  BUG_FIX: 'Correção',
  ARCHITECTURE: 'Arquitetura',
  DEVOPS: 'DevOps',
  RESEARCH: 'Pesquisa',
  SECURITY: 'Segurança',
  GENERIC: 'Genérico',
};

export const ComposerSurface: React.FC<{
  project: Project;
  onTransformFlow: (artifact: PromptArtifact) => void;
  destinationGate?: ComposerGate;
}> = ({ project, onTransformFlow, destinationGate }) => {
  const { t } = useTranslation();
  const [sessions, setSessions] = useState<ComposerSession[]>([]);
  const [view, setView] = useState<ComposerSessionView | null>(null);
  const [inputMode, setInputMode] = useState<'IDEA' | 'EXISTING_PROMPT'>('IDEA');
  const [draft, setDraft] = useState('');
  const [sourcePrompt, setSourcePrompt] = useState('');
  const [message, setMessage] = useState('');
  const [artifact, setArtifact] = useState<PromptArtifact | null>(null);
  const [error, setError] = useState('');
  const [confirmGaps, setConfirmGaps] = useState(false);
  const [busy, setBusy] = useState(false);
  const [refineText, setRefineText] = useState('');
  const [showRefineInput, setShowRefineInput] = useState(false);
  const [unknownAnswers, setUnknownAnswers] = useState<Record<string, string>>({});
  const [agents, setAgents] = useState<Agent[]>([]);
  const [selectedAgentId, setSelectedAgentId] = useState('');
  const [preparationOpen, setPreparationOpen] = useState(false);
  const canMaterialize = destinationGate?.canMaterialize ?? true;
  const canExecute = destinationGate?.canExecute ?? true;

  const selectedSkillIds = useMemo(
    () =>
      (view?.skills || [])
        .filter((skill) => skill.state === 'ACCEPTED' || skill.state === 'APPLIED')
        .map((skill) => skill.skill_id),
    [view?.skills],
  );

  const refreshSessions = useCallback(async () => {
    const next = await nexus.listComposerSessions(project.id);
    setSessions(next || []);
    return next || [];
  }, [project.id]);
  useEffect(() => {
    void (async () => {
      try {
        const next = await refreshSessions();
        const id = selectResumableComposerSession(next);
        if (id) setView(await nexus.getComposerSession(id));
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      }
    })();
  }, [project.id, refreshSessions]);

  useEffect(() => {
    void nexus
      .listAgents(project.id)
      .then((items) => {
        const next = items || [];
        setAgents(next);
        if (!selectedAgentId && next[0]) setSelectedAgentId(next[0].id);
      })
      .catch(() => undefined);
  }, [project.id, selectedAgentId]);

  const briefItems = useMemo(
    () =>
      view
        ? [
            ['Entendido', view.brief.goal],
            [
              'Contexto',
              [
                ...asStringArray(view.brief.context),
                ...(view.brief.context &&
                typeof view.brief.context === 'object' &&
                !Array.isArray(view.brief.context)
                  ? asStringArray(
                      (view.brief.context as { existing_state?: unknown }).existing_state,
                    )
                  : []),
              ].join(' · '),
            ],
            ['Decisões', asStringArray(view.brief.decisions).join(' · ')],
            ['Critérios', asStringArray(view.brief.success_criteria).join(' · ')],
            ['Dúvidas abertas', asStringArray(view.brief.open_questions).join(' · ')],
          ].filter(([, value]) => value)
        : [],
    [view],
  );

  const create = async () => {
    if (!draft.trim()) return;
    setBusy(true);
    setError('');
    try {
      const next = await nexus.createComposerSessionWithMode(
        project.id,
        draft.trim(),
        inputMode,
        inputMode === 'EXISTING_PROMPT' ? sourcePrompt.trim() : undefined,
      );
      setView(next);
      setDraft('');
      setSourcePrompt('');
      setArtifact(null);
      await refreshSessions();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const send = async () => {
    if (!view || !message.trim() || busy) return;
    const textToSend = message.trim();
    setBusy(true);
    setError('');
    try {
      setView(await nexus.addComposerTurn(view.session.id, textToSend, view.session.revision));
      setMessage('');
      await refreshSessions();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      // message is preserved because setMessage('') was only called on success!
    } finally {
      setBusy(false);
    }
  };

  const finalize = async (confirmed = false) => {
    if (!view) return;
    if (!confirmed && composerNeedsGapConfirmation(view.brief)) {
      setConfirmGaps(true);
      setError(t('work.confirmGaps'));
      return;
    }
    setBusy(true);
    setError('');
    try {
      const res = await nexus.finalizeComposerSession(view.session.id, selectedSkillIds, confirmed);
      setArtifact(res);
      setView(await nexus.getComposerSession(view.session.id));
      await refreshSessions();
    } catch (err) {
      const detail = err instanceof Error ? err.message : String(err);
      setError(detail);
      setConfirmGaps(detail.includes('open questions') || detail.includes('gaps'));
    } finally {
      setBusy(false);
    }
  };

  const refine = async () => {
    if (!view) return;
    setBusy(true);
    setError('');
    try {
      const newArtifact = await nexus.refineComposerArtifact(view.session.id, refineText.trim());
      setArtifact(newArtifact);
      setShowRefineInput(false);
      setRefineText('');
      setView(await nexus.getComposerSession(view.session.id));
      await refreshSessions();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const resolveUnknown = async (unknownId: string, status: string) => {
    if (!view) return;
    setBusy(true);
    setError('');
    try {
      const answer = unknownAnswers[unknownId] || '';
      const updated = await nexus.resolveComposerUnknown(
        view.session.id,
        unknownId,
        answer,
        status,
        view.session.revision,
      );
      setView(updated);
      setUnknownAnswers((prev) => {
        const next = { ...prev };
        delete next[unknownId];
        return next;
      });
      await refreshSessions();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const updateSkill = async (skillId: string, state: 'ACCEPTED' | 'REJECTED') => {
    if (!view) return;
    setBusy(true);
    setError('');
    try {
      setView(await nexus.updateComposerSkillState(view.session.id, skillId, state));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const copy = async () => {
    if (!artifact) return;
    try {
      await navigator.clipboard.writeText(artifact.content);
    } catch {
      setError('Não foi possível copiar o prompt neste navegador.');
    }
  };

  const selectedAgent = agents.find((agent) => agent.id === selectedAgentId);
  const startAgentIfNeeded = !askActionForStatus(selectedAgent?.status || 'STOPPED').startIfNeeded;

  if (!view) {
    return (
      <Card className="nx-composer-goal-bar">
        <div className={`nx-composer-goal-bar__label ${styles.flexRowBetween}`}>
          <div className={styles.flexRowGap6}>
            <Sparkles size={16} />
            <span>Comece uma elaboração</span>
          </div>
          <div className={styles.flexRowGap4}>
            <Button
              size="sm"
              tone={inputMode === 'IDEA' ? 'brand' : 'default'}
              onClick={() => setInputMode('IDEA')}
              disabled={busy}
            >
              <Lightbulb size={13} /> Ideia / Explorar
            </Button>
            <Button
              size="sm"
              tone={inputMode === 'EXISTING_PROMPT' ? 'brand' : 'default'}
              onClick={() => setInputMode('EXISTING_PROMPT')}
              disabled={busy}
            >
              <FileText size={13} /> Prompt Existente
            </Button>
          </div>
        </div>

        <div className={styles.flexColGap8}>
          <Input
            value={draft}
            onChange={setDraft}
            placeholder={
              inputMode === 'IDEA'
                ? 'Qual ideia ou objetivo você quer transformar em um ótimo prompt?'
                : 'Qual o objetivo principal deste prompt existente?'
            }
            style={{ flex: 1 }}
            disabled={busy}
          />
          {inputMode === 'EXISTING_PROMPT' && (
            <textarea
              className={styles.sourceTextarea}
              value={sourcePrompt}
              onChange={(e) => setSourcePrompt(e.target.value)}
              placeholder="Cole aqui o prompt original para análise de gaps, completude e estruturação…"
              rows={4}
              disabled={busy}
            />
          )}
          <div className={styles.flexRowEnd}>
            <Button
              tone="brand"
              disabled={
                !draft.trim() || (inputMode === 'EXISTING_PROMPT' && !sourcePrompt.trim()) || busy
              }
              onClick={() => void create()}
            >
              <MessageCircle size={14} />{' '}
              {busy
                ? 'Iniciando…'
                : inputMode === 'EXISTING_PROMPT'
                  ? 'Analisar Prompt'
                  : 'Conversar'}
            </Button>
          </div>
        </div>

        {sessions.length > 0 && (
          <div className={styles.flexRowWrapGap8}>
            <small className={styles.mutedLabel}>Elaborações anteriores:</small>
            <Select
              placeholder="Retomar uma sessão salva…"
              value=""
              onChange={async (id) => {
                if (!id) return;
                setBusy(true);
                setError('');
                try {
                  setView(await nexus.getComposerSession(id));
                } catch (err) {
                  setError(err instanceof Error ? err.message : String(err));
                } finally {
                  setBusy(false);
                }
              }}
              options={sessions.map((s) => ({
                value: s.id,
                label: `${s.title || `Sessão ${s.id.slice(-6)}`} · ${s.state}`,
              }))}
              selectStyle={{ fontSize: '0.857rem', height: 28 }}
            />
          </div>
        )}
        {error && <div className="nx-inline-error">{error}</div>}
      </Card>
    );
  }

  const archetype = view.brief.intent?.archetype;
  const archetypeLabel = archetype ? ARCHETYPE_LABELS[archetype] || archetype : null;

  return (
    <div className="nx-composer-deliberative">
      <div className="nx-composer-deliberative__header">
        <div>
          <h2>{view.session.title || 'Elaboração'}</h2>
        </div>
        <div className={styles.flexRowGap8Center}>
          {archetypeLabel && <Badge tone="default">Arquétipo: {archetypeLabel}</Badge>}
          {sessions.length > 1 && (
            <Select
              value={view.session.id}
              onChange={async (id) => {
                if (!id || id === view.session.id) return;
                setBusy(true);
                setError('');
                try {
                  setView(await nexus.getComposerSession(id));
                  setArtifact(null);
                } catch (err) {
                  setError(err instanceof Error ? err.message : String(err));
                } finally {
                  setBusy(false);
                }
              }}
              options={sessions.map((s) => ({
                value: s.id,
                label: `${s.title || `Sessão ${s.id.slice(-6)}`} (${s.state})`,
              }))}
              selectStyle={{ fontSize: '0.857rem', height: 28 }}
            />
          )}
          <Badge tone={view.session.state === 'FINALIZED' ? 'success' : 'brand'}>
            {view.session.state}
          </Badge>
          <Button
            size="sm"
            disabled={busy}
            onClick={() => {
              setView(null);
              setArtifact(null);
            }}
          >
            <Plus size={13} /> Nova
          </Button>
        </div>
      </div>

      <div className="nx-composer-deliberative__grid">
        <Card>
          <strong>Conversa de elaboração</strong>
          <div className="nx-composer-turns">
            {(view.turns || []).length === 0 ? (
              <p className="nx-muted-copy">
                Descreva contexto, resultado desejado e limitações. O Composer preserva esta
                elaboração.
              </p>
            ) : (
              (view.turns || []).map((turn) => (
                <div
                  key={turn.id}
                  className={`nx-composer-turn nx-composer-turn--${turn.role.toLowerCase()}`}
                >
                  <small>{turn.role === 'USER' ? 'Você' : 'Composer'}</small>
                  <p>{turn.content}</p>
                </div>
              ))
            )}
          </div>
          {view.session.state !== 'FINALIZED' && (
            <div
              className={`nx-composer-goal-bar__row ${styles.flexRowGap8mt}`}
            >
              <textarea
                className={`nx-textarea ${styles.composerTextarea}`}
                value={message}
                onChange={(e) => setMessage(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && !e.shiftKey && !e.nativeEvent.isComposing) {
                    e.preventDefault();
                    void send();
                  }
                }}
                placeholder="Adicione requisito, decisão ou resposta a uma lacuna… (Enter envia, Shift+Enter pula linha)"
                rows={2}
                disabled={busy}
              />
              <Button
                tone="brand"
                disabled={!message.trim() || busy}
                onClick={() => void send()}
                className={styles.sendButton}
              >
                <Send size={14} /> {busy ? 'Enviando…' : 'Enviar'}
              </Button>
            </div>
          )}
        </Card>

        <Card>
          <strong>Briefing vivo</strong>
          <div className={styles.flexColGap10}>
            {briefItems.length > 0 ? (
              briefItems.map(([label, value]) => (
                <div key={label}>
                  <small className={styles.mutedLabel}>{label}</small>
                  <p className={styles.briefValue}>{value}</p>
                </div>
              ))
            ) : (
              <p className={`nx-muted-copy ${styles.briefEmptyHint}`}>
                O briefing vivo sintetiza objetivo, contexto, critérios e decisões conforme a
                conversa avança.
              </p>
            )}
          </div>
          <div className={styles.badgeSpacer}>
            <Badge tone="default">Maestro: sugestões reais ao refinar</Badge>
          </div>
        </Card>
      </div>

      <div className="nx-composer-deliberative__grid">
        <Card>
          <strong>Prompt Readiness</strong>
          <p>{view.brief.readiness?.summary || 'Ainda sem avaliação.'}</p>
          <Badge tone={view.brief.readiness?.state === 'READY' ? 'success' : 'warning'}>
            {view.brief.readiness?.state || 'UNKNOWN'} · {view.brief.readiness?.score ?? 0}%
          </Badge>

          {asArray<PromptReadinessCheck>(view.brief.readiness?.checks).length > 0 && (
            <div className={`${styles.flexColGap6} ${styles.readinessSection}`}>
              <small className={styles.mutedLabelBold}>Dimensões avaliadas</small>
              {asArray<PromptReadinessCheck>(view.brief.readiness?.checks).map((check) => (
                <div key={check.key} className={styles.readinessCheckItem}>
                  <div className={styles.readinessCheckContent}>
                    <span className={styles.readinessCheckLabel}>{check.label || check.key}</span>
                    {check.summary && (
                      <small className={styles.readinessCheckSummary}>
                        {check.summary}
                      </small>
                    )}
                  </div>
                  <span
                    className={
                      check.score >= 80
                        ? styles.readinessCheckScoreHigh
                        : styles.readinessCheckScore
                    }
                  >
                    {check.score}%
                  </span>
                </div>
              ))}
            </div>
          )}

          {(view.brief.assumptions || []).length > 0 && (
            <div className={styles.readinessSection}>
              <small className={styles.mutedLabelBold}>Premissas ativas</small>
              {view.brief.assumptions?.map((item, index) => (
                <p key={index} className={`nx-muted-copy ${styles.assumptionItem}`}>
                  • {typeof item === 'string' ? item : item.value}{' '}
                  {typeof item !== 'string' && item.status ? `(${item.status})` : ''}
                </p>
              ))}
            </div>
          )}
        </Card>

        <Card>
          <strong>Lacunas & Perguntas (Unknowns)</strong>
          {(view.brief.unknowns || []).length === 0 ? (
            <p className="nx-muted-copy">Nenhuma lacuna aberta.</p>
          ) : (
            view.brief.unknowns?.map((unknown) => {
              const isResolved =
                unknown.status === 'ANSWERED' ||
                unknown.status === 'CONFIRMED' ||
                unknown.status === 'DISMISSED';
              return (
                <div key={unknown.id} className={styles.unknownCard}>
                  <div className={styles.unknownHeader}>
                    <p className={styles.unknownQuestion}>{unknown.question}</p>
                    <Badge
                      tone={
                        unknown.status === 'ANSWERED'
                          ? 'success'
                          : unknown.severity === 'BLOCKING'
                            ? 'danger'
                            : 'warning'
                      }
                    >
                      {unknown.severity} · {unknown.status}
                    </Badge>
                  </div>
                  {unknown.answer && (
                    <small className={styles.unknownAnswer}>Resposta: {unknown.answer}</small>
                  )}
                  {!isResolved && (
                    <div className={styles.unknownResolveArea}>
                      <input
                        className={`nx-input ${styles.unknownAnswerInput}`}
                        placeholder="Sua resposta para esta lacuna…"
                        value={unknownAnswers[unknown.id] || ''}
                        onChange={(e) =>
                          setUnknownAnswers({ ...unknownAnswers, [unknown.id]: e.target.value })
                        }
                        disabled={busy}
                      />
                      <div className={styles.unknownAnswerButtons}>
                        <Button
                          size="sm"
                          tone="brand"
                          disabled={!unknownAnswers[unknown.id]?.trim() || busy}
                          onClick={() => void resolveUnknown(unknown.id, 'ANSWERED')}
                        >
                          <CheckCircle2 size={12} /> Responder
                        </Button>
                        <Button
                          size="sm"
                          disabled={busy}
                          onClick={() => void resolveUnknown(unknown.id, 'DISMISSED')}
                        >
                          Dispensar
                        </Button>
                      </div>
                    </div>
                  )}
                </div>
              );
            })
          )}
        </Card>
      </div>

      {(view.skills || []).length > 0 && (
        <Card className={styles.artifactCard}>
          <strong>Maestro skills</strong>
          {(view.skills || []).map((skill) => (
            <div key={skill.skill_id} className={styles.skillRow}>
              <div className={styles.skillInfo}>
                <strong>{skill.skill_id}</strong>
                <small className={styles.skillReason}>{skill.reason || skill.applicability}</small>
              </div>
              <Badge tone={skill.state === 'UNAVAILABLE' ? 'danger' : 'default'}>
                {skill.state}
              </Badge>
              {skill.state !== 'UNAVAILABLE' && skill.state !== 'APPLIED' && (
                <>
                  <Button
                    size="sm"
                    tone="brand"
                    disabled={busy}
                    onClick={() => void updateSkill(skill.skill_id, 'ACCEPTED')}
                  >
                    Aceitar
                  </Button>
                  <Button
                    size="sm"
                    disabled={busy}
                    onClick={() => void updateSkill(skill.skill_id, 'REJECTED')}
                  >
                    Dispensar
                  </Button>
                </>
              )}
            </div>
          ))}
        </Card>
      )}

      {error && <div className="nx-inline-error">{error}</div>}

      {!artifact && view.session.state !== 'FINALIZED' && (
        <div className="nx-composer-header-actions">
          <Button tone="brand" disabled={busy} onClick={() => void finalize()}>
            <Sparkles size={14} /> {busy ? 'Finalizando…' : 'Concluir elaboração'}
          </Button>
          {confirmGaps && (
            <Button tone="warning" disabled={busy} onClick={() => void finalize(true)}>
              Concluir com lacunas confirmadas
            </Button>
          )}
        </div>
      )}

      {artifact && (
        <Card className={styles.artifactCard}>
          <div className={styles.artifactHeader}>
            <strong>Prompt canônico · v{artifact.version}</strong>
            <Badge tone="success">Versão imutável #{artifact.version}</Badge>
          </div>
          <pre className={`nx-flow-step-compare ${styles.preWrap}`}>
            {artifact.content}
          </pre>
          <div
            className={`nx-composer-header-actions ${styles.artifactActions}`}
          >
            <Button onClick={() => void copy()}>
              <ClipboardCopy size={14} /> Copiar
            </Button>
            {agents.length > 0 && (
              <>
                <Select
                  value={selectedAgentId}
                  onChange={setSelectedAgentId}
                  options={agents.map((agent) => ({ value: agent.id, label: agent.name }))}
                  selectStyle={{ minWidth: 150 }}
                />
                <Button
                  tone="brand"
                  disabled={busy || !selectedAgentId || !canExecute}
                  onClick={() => setPreparationOpen(true)}
                >
                  <Send size={14} />{' '}
                  {t('work.composer.reviewBeforeSend', { defaultValue: 'Revisar antes de enviar' })}
                </Button>
              </>
            )}
            <Button
              tone="brand"
              disabled={!canMaterialize}
              title={!canMaterialize ? destinationGate?.reason : undefined}
              onClick={() => onTransformFlow(artifact)}
            >
              <Layers size={14} /> Transformar em Flow
            </Button>
            <Button disabled={busy} onClick={() => setShowRefineInput(!showRefineInput)}>
              <RefreshCw size={14} /> Refinar (v{artifact.version + 1})
            </Button>
          </div>
          {destinationGate && (!canMaterialize || !canExecute) && (
            <small className="nx-muted-copy">
              Copy permanece disponível. {destinationGate.reason} Prepare o contexto para habilitar
              Flow e Agent.
            </small>
          )}

          {showRefineInput && (
            <div className={styles.refinementPanel}>
              <small className={styles.refinementLabel}>
                Instrução adicional de refinamento (opcional):
              </small>
              <Input
                value={refineText}
                onChange={setRefineText}
                placeholder="Ex: 'Adicione suporte a PostgreSQL', 'Foque apenas na API REST'…"
                disabled={busy}
              />
              <div className={styles.refinementActions}>
                <Button size="sm" disabled={busy} onClick={() => setShowRefineInput(false)}>
                  Cancelar
                </Button>
                <Button size="sm" tone="brand" disabled={busy} onClick={() => void refine()}>
                  {busy
                    ? 'Gerando v' + (artifact.version + 1) + '…'
                    : 'Gerar nova revisão (v' + (artifact.version + 1) + ')'}
                </Button>
              </div>
            </div>
          )}
        </Card>
      )}
      <TaskPreparationDialog
        open={preparationOpen}
        agentId={selectedAgentId}
        agentName={selectedAgent?.name}
        projectId={project.id}
        initialPrompt={artifact?.content || ''}
        startIfNeeded={startAgentIfNeeded}
        onClose={() => setPreparationOpen(false)}
        onSubmit={async (prompt, skills, startIfNeeded, contextFingerprintId) => {
          await nexus.askAgent(selectedAgentId, prompt, startIfNeeded, skills, {
            projectId: project.id,
            contextFingerprintId: contextFingerprintId || '',
          });
          setMessage(t('work.composer.sentToAgent', { defaultValue: 'Tarefa enviada ao Agente.' }));
        }}
      />
    </div>
  );
};
