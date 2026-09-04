import React, { useEffect, useMemo, useState } from 'react';
import { ClipboardCopy, Layers, MessageCircle, Plus, Send, Sparkles } from 'lucide-react';
import { Badge, Button, Card, Input } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { ComposerSession, ComposerSessionView, PromptArtifact, Project } from '../../types';
import { selectResumableComposerSession } from './composerSessionModel';
import { asStringArray } from '../../lib/safeArray';

export const ComposerSurface: React.FC<{ project: Project; onTransformFlow: (prompt: string) => void }> = ({ project, onTransformFlow }) => {
  const [sessions, setSessions] = useState<ComposerSession[]>([]);
  const [view, setView] = useState<ComposerSessionView | null>(null);
  const [draft, setDraft] = useState('');
  const [message, setMessage] = useState('');
  const [artifact, setArtifact] = useState<PromptArtifact | null>(null);
  const [error, setError] = useState('');
  const [confirmGaps, setConfirmGaps] = useState(false);
  const [busy, setBusy] = useState(false);

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
  const finalize = async (confirmed = false) => { if (!view) return; setBusy(true); setError(''); try { setArtifact(await nexus.finalizeComposerSession(view.session.id, [], confirmed)); setView(await nexus.getComposerSession(view.session.id)); await refreshSessions(); } catch (err) { const detail = err instanceof Error ? err.message : String(err); setError(detail); setConfirmGaps(detail.includes('open questions')); } finally { setBusy(false); } };
  const copy = async () => { if (!artifact) return; try { await navigator.clipboard.writeText(artifact.content); } catch { setError('Não foi possível copiar o prompt neste navegador.'); } };

  if (!view) return <Card className="nx-composer-goal-bar"><div className="nx-composer-goal-bar__label"><Sparkles size={16} /><span>Comece uma elaboração</span></div><div className="nx-composer-goal-bar__row"><Input value={draft} onChange={setDraft} placeholder="Qual ideia você quer transformar em um ótimo prompt?" style={{ flex: 1 }} /><Button tone="brand" disabled={!draft.trim() || busy} onClick={() => void create()}><MessageCircle size={14} /> Conversar</Button></div>{sessions.length > 0 && <small>Há {sessions.length} elaborações anteriores neste Project.</small>}{error && <div className="nx-inline-error">{error}</div>}</Card>;

  return <div className="nx-composer-deliberative">
    <div className="nx-composer-deliberative__header"><div><span className="nx-eyebrow"><Sparkles size={13} /> COMPOSER</span><h2>{view.session.title || 'Elaboração'}</h2></div><div style={{ display: 'flex', gap: 8 }}><Badge tone={view.session.state === 'FINALIZED' ? 'success' : 'brand'}>{view.session.state}</Badge><Button size="sm" onClick={() => { setView(null); setArtifact(null); }}><Plus size={13} /> Nova</Button></div></div>
    <div className="nx-composer-deliberative__grid">
      <Card><strong>Conversa de elaboração</strong><div className="nx-composer-turns">{view.turns.length === 0 ? <p className="nx-muted-copy">Descreva contexto, resultado desejado e limitações. O Composer preserva esta elaboração.</p> : view.turns.map((turn) => <div key={turn.id} className={`nx-composer-turn nx-composer-turn--${turn.role.toLowerCase()}`}><small>{turn.role === 'USER' ? 'Você' : 'Composer'}</small><p>{turn.content}</p></div>)}</div>{view.session.state !== 'FINALIZED' && <div className="nx-composer-goal-bar__row"><Input value={message} onChange={setMessage} placeholder="Adicione requisito, decisão ou pergunta…" style={{ flex: 1 }} /><Button tone="brand" disabled={!message.trim() || busy} onClick={() => void send()}><Send size={14} /> Enviar</Button></div>}</Card>
      <Card><strong>Briefing vivo</strong><div style={{ display: 'grid', gap: 10, marginTop: 12 }}>{briefItems.map(([label, value]) => <div key={label}><small style={{ color: 'var(--nx-muted)' }}>{label}</small><p style={{ margin: '2px 0 0' }}>{value}</p></div>) || <p className="nx-muted-copy">O briefing será preenchido durante a conversa.</p>}</div><div style={{ marginTop: 16 }}><Badge tone="default">Maestro: sugestões reais ao refinar</Badge></div></Card>
    </div>
    {error && <div className="nx-inline-error">{error}</div>}
    {!artifact && view.session.state !== 'FINALIZED' && <div className="nx-composer-header-actions"><Button tone="brand" disabled={busy} onClick={() => void finalize()}><Sparkles size={14} /> Concluir elaboração</Button>{confirmGaps && <Button tone="warning" disabled={busy} onClick={() => void finalize(true)}>Concluir com lacunas confirmadas</Button>}</div>}
    {artifact && <Card style={{ marginTop: 12 }}><strong>Prompt canônico · v{artifact.version}</strong><pre className="nx-flow-step-compare" style={{ whiteSpace: 'pre-wrap', marginTop: 8 }}>{artifact.content}</pre><div className="nx-composer-header-actions"><Button onClick={() => void copy()}><ClipboardCopy size={14} /> Copiar</Button><Button tone="brand" onClick={() => onTransformFlow(artifact.content)}><Layers size={14} /> Transformar em Flow</Button></div></Card>}
  </div>;
};
