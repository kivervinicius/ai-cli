import React, { useEffect, useMemo, useState } from 'react';
import { ClipboardCopy, Layers, MessageCircle, Plus, Send, Sparkles } from 'lucide-react';
import { Badge, Button, Card, Input } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { ComposerSession, ComposerSessionView, PromptArtifact, Project } from '../../types';
import { selectResumableComposerSession } from './composerSessionModel';
import { asStringArray } from '../../lib/safeArray';

export const ComposerSurface: React.FC<{ project: Project; onTransformFlow: (artifact: PromptArtifact) => void }> = ({ project, onTransformFlow }) => {
  const [sessions, setSessions] = useState<ComposerSession[]>([]);
  const [view, setView] = useState<ComposerSessionView | null>(null);
  const [draft, setDraft] = useState('');
  const [message, setMessage] = useState('');
  const [artifact, setArtifact] = useState<PromptArtifact | null>(null);
  const [error, setError] = useState('');
  const [confirmGaps, setConfirmGaps] = useState(false);
  const [busy, setBusy] = useState(false);
  const selectedSkillIds = useMemo(() => (view?.skills || []).filter((skill) => skill.state === 'ACCEPTED' || skill.state === 'APPLIED').map((skill) => skill.skill_id), [view?.skills]);

  const refreshSessions = async () => {
    const next = await nexus.listComposerSessions(project.id);
    setSessions(next || []);
    return next || [];
  };
  useEffect(() => { void (async () => { try { const next = await refreshSessions(); const id = selectResumableComposerSession(next); if (id) setView(await nexus.getComposerSession(id)); } catch (err) { setError(err instanceof Error ? err.message : String(err)); } })(); }, [project.id]);

  const briefItems = useMemo(() => view ? [
    ['Entendido', view.brief.goal],
    ['Contexto', [
      ...asStringArray(view.brief.context),
      ...(view.brief.context && typeof view.brief.context === 'object' && !Array.isArray(view.brief.context)
        ? asStringArray((view.brief.context as { existing_state?: unknown }).existing_state)
        : []),
    ].join(' · ')],
    ['Decisões', asStringArray(view.brief.decisions).join(' · ')],
    ['Critérios', asStringArray(view.brief.success_criteria).join(' · ')],
    ['Dúvidas abertas', asStringArray(view.brief.open_questions).join(' · ')],
  ].filter(([, value]) => value) : [], [view]);

  const create = async () => { if (!draft.trim()) return; setBusy(true); setError(''); try { const next = await nexus.createComposerSession(project.id, draft.trim()); setView(next); setDraft(''); setArtifact(null); await refreshSessions(); } catch (err) { setError(err instanceof Error ? err.message : String(err)); } finally { setBusy(false); } };
  const send = async () => { if (!view || !message.trim()) return; setBusy(true); setError(''); try { setView(await nexus.addComposerTurn(view.session.id, message.trim())); setMessage(''); await refreshSessions(); } catch (err) { setError(err instanceof Error ? err.message : String(err)); } finally { setBusy(false); } };
  const finalize = async (confirmed = false) => { if (!view) return; setBusy(true); setError(''); try { setArtifact(await nexus.finalizeComposerSession(view.session.id, selectedSkillIds, confirmed)); setView(await nexus.getComposerSession(view.session.id)); await refreshSessions(); } catch (err) { const detail = err instanceof Error ? err.message : String(err); setError(detail); setConfirmGaps(detail.includes('open questions')); } finally { setBusy(false); } };
  const updateSkill = async (skillId: string, state: 'ACCEPTED' | 'REJECTED') => { if (!view) return; setBusy(true); setError(''); try { setView(await nexus.updateComposerSkillState(view.session.id, skillId, state)); } catch (err) { setError(err instanceof Error ? err.message : String(err)); } finally { setBusy(false); } };
  const copy = async () => { if (!artifact) return; try { await navigator.clipboard.writeText(artifact.content); } catch { setError('Não foi possível copiar o prompt neste navegador.'); } };

  if (!view) {
    return (
      <Card className="nx-composer-goal-bar">
        <div className="nx-composer-goal-bar__label">
          <Sparkles size={16} />
          <span>Comece uma elaboração</span>
        </div>
        <div className="nx-composer-goal-bar__row">
          <Input
            value={draft}
            onChange={setDraft}
            placeholder="Qual ideia você quer transformar em um ótimo prompt?"
            style={{ flex: 1 }}
            disabled={busy}
          />
          <Button tone="brand" disabled={!draft.trim() || busy} onClick={() => void create()}>
            <MessageCircle size={14} /> {busy ? 'Iniciando…' : 'Conversar'}
          </Button>
        </div>
        {sessions.length > 0 && (
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap', marginTop: 4 }}>
            <small style={{ color: 'var(--nx-muted)' }}>Elaborações anteriores:</small>
            <select
              className="nx-select"
              style={{ fontSize: 12, padding: '2px 8px', borderRadius: 6, background: 'var(--nx-surface)', color: 'var(--nx-text)' }}
              defaultValue=""
              onChange={async (e) => {
                const id = e.target.value;
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
            >
              <option value="">Retomar uma sessão salva…</option>
              {sessions.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.title || `Sessão ${s.id.slice(-6)}`} · {s.state}
                </option>
              ))}
            </select>
          </div>
        )}
        {error && <div className="nx-inline-error">{error}</div>}
      </Card>
    );
  }

  return (
    <div className="nx-composer-deliberative">
      <div className="nx-composer-deliberative__header">
        <div>
          <span className="nx-eyebrow"><Sparkles size={13} /> COMPOSER</span>
          <h2>{view.session.title || 'Elaboração'}</h2>
        </div>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
          {sessions.length > 1 && (
            <select
              className="nx-select"
              style={{ fontSize: 12, padding: '3px 8px', borderRadius: 6, background: 'var(--nx-surface)', color: 'var(--nx-text)' }}
              value={view.session.id}
              onChange={async (e) => {
                const id = e.target.value;
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
            >
              {sessions.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.title || `Sessão ${s.id.slice(-6)}`} ({s.state})
                </option>
              ))}
            </select>
          )}
          <Badge tone={view.session.state === 'FINALIZED' ? 'success' : 'brand'}>{view.session.state}</Badge>
          <Button size="sm" disabled={busy} onClick={() => { setView(null); setArtifact(null); }}>
            <Plus size={13} /> Nova
          </Button>
        </div>
      </div>

      <div className="nx-composer-deliberative__grid">
        <Card>
          <strong>Conversa de elaboração</strong>
          <div className="nx-composer-turns">
            {(view.turns || []).length === 0 ? (
              <p className="nx-muted-copy">Descreva contexto, resultado desejado e limitações. O Composer preserva esta elaboração.</p>
            ) : (
              (view.turns || []).map((turn) => (
                <div key={turn.id} className={`nx-composer-turn nx-composer-turn--${turn.role.toLowerCase()}`}>
                  <small>{turn.role === 'USER' ? 'Você' : 'Composer'}</small>
                  <p>{turn.content}</p>
                </div>
              ))
            )}
          </div>
          {view.session.state !== 'FINALIZED' && (
            <div className="nx-composer-goal-bar__row">
              <Input
                value={message}
                onChange={setMessage}
                placeholder="Adicione requisito, decisão ou pergunta…"
                style={{ flex: 1 }}
                disabled={busy}
              />
              <Button tone="brand" disabled={!message.trim() || busy} onClick={() => void send()}>
                <Send size={14} /> {busy ? 'Enviando…' : 'Enviar'}
              </Button>
            </div>
          )}
        </Card>

        <Card>
          <strong>Briefing vivo</strong>
          <div style={{ display: 'grid', gap: 10, marginTop: 12 }}>
            {briefItems.length > 0 ? (
              briefItems.map(([label, value]) => (
                <div key={label}>
                  <small style={{ color: 'var(--nx-muted)' }}>{label}</small>
                  <p style={{ margin: '2px 0 0' }}>{value}</p>
                </div>
              ))
            ) : (
              <p className="nx-muted-copy" style={{ margin: '6px 0 0' }}>
                O briefing vivo sintetiza objetivo, contexto, critérios e decisões conforme a conversa avança.
              </p>
            )}
          </div>
          <div style={{ marginTop: 16 }}>
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
          {(view.brief.assumptions || []).length > 0 && (
            <div style={{ marginTop: 12 }}>
              <small>Assumptions</small>
              {view.brief.assumptions?.map((item, index) => (
                <p key={index} className="nx-muted-copy">
                  {typeof item === 'string' ? item : item.value}{' '}
                  {typeof item !== 'string' && item.status ? `· ${item.status}` : ''}
                </p>
              ))}
            </div>
          )}
        </Card>

        <Card>
          <strong>Unknowns</strong>
          {(view.brief.unknowns || []).length === 0 ? (
            <p className="nx-muted-copy">Nenhuma lacuna aberta.</p>
          ) : (
            view.brief.unknowns?.map((unknown) => (
              <div key={unknown.id} style={{ marginTop: 10 }}>
                <p style={{ margin: 0 }}>{unknown.question}</p>
                <small>{unknown.severity} · {unknown.status}</small>
              </div>
            ))
          )}
        </Card>
      </div>

      {(view.skills || []).length > 0 && (
        <Card style={{ marginTop: 12 }}>
          <strong>Maestro skills</strong>
          {(view.skills || []).map((skill) => (
            <div key={skill.skill_id} style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 10 }}>
              <div style={{ flex: 1 }}>
                <strong>{skill.skill_id}</strong>
                <small style={{ display: 'block' }}>{skill.reason || skill.applicability}</small>
              </div>
              <Badge tone={skill.state === 'UNAVAILABLE' ? 'danger' : 'default'}>{skill.state}</Badge>
              {skill.state !== 'UNAVAILABLE' && skill.state !== 'APPLIED' && (
                <>
                  <Button size="sm" tone="brand" disabled={busy} onClick={() => void updateSkill(skill.skill_id, 'ACCEPTED')}>
                    Aceitar
                  </Button>
                  <Button size="sm" disabled={busy} onClick={() => void updateSkill(skill.skill_id, 'REJECTED')}>
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
        <Card style={{ marginTop: 12 }}>
          <strong>Prompt canônico · v{artifact.version}</strong>
          <pre className="nx-flow-step-compare" style={{ whiteSpace: 'pre-wrap', marginTop: 8 }}>
            {artifact.content}
          </pre>
          <div className="nx-composer-header-actions">
            <Button onClick={() => void copy()}>
              <ClipboardCopy size={14} /> Copiar
            </Button>
            <Button tone="brand" onClick={() => onTransformFlow(artifact)}>
              <Layers size={14} /> Transformar em Flow
            </Button>
          </div>
        </Card>
      )}
    </div>
  );
};
