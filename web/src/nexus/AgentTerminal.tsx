import React, { useEffect, useRef, useState } from 'react';
import { MessageSquare, Play, RefreshCw, Send, ShieldAlert, Sparkles, Trash2, Unplug, X } from 'lucide-react';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import { nexus } from './api';
import {
  TERMINAL_MAX_RECONNECT_ATTEMPTS,
  agentTerminalWebSocketURL,
  isFatalTerminalAttachError,
  normalizeInitialPrompt,
  normalizeTerminalRole,
  isRecoverAlreadyAlive,
  shouldFallbackRecoverToStart,
  terminalAttachFailureMessage,
  terminalReconnectDelay,
  type TerminalRole,
} from './agentTerminalModel';
import { TerminalActionDialog } from './TerminalActionDialog';
import { scrubProtocolOutput } from './terminalProtocol';
import type { RuntimeSession } from '../types';

async function recoverOrStartAgent(agentId: string) {
  try {
    return await nexus.recoverAgent(agentId);
  } catch (recoverErr) {
    const recoverMsg = recoverErr instanceof Error ? recoverErr.message : String(recoverErr);
    // Healthy runtime: soft-success so the caller rebinds/reconnects without killing the session.
    if (isRecoverAlreadyAlive(recoverMsg)) {
      return { runtime: undefined as unknown as RuntimeSession };
    }
    if (!shouldFallbackRecoverToStart(recoverMsg)) {
      throw recoverErr;
    }
    try {
      return await nexus.startAgent(agentId);
    } catch (startErr) {
      const startMsg = startErr instanceof Error ? startErr.message : String(startErr);
      throw new Error(`${recoverMsg} · Start: ${startMsg}`);
    }
  }
}

export const AgentTerminal: React.FC<{
  agentId: string;
  runtimeId?: string;
  initialPrompt?: string;
  provider?: string;
  profile?: string;
  mode?: 'Safe' | 'YOLO';
  agentName?: string;
  /** When "window", identity lives on the Desktop titlebar; actions stay in a compact toolbar. */
  chrome?: 'full' | 'window';
  onRecover?: () => Promise<RuntimeSession | void> | RuntimeSession | void;
  onRestartWithMode?: (mode: 'Safe' | 'YOLO') => Promise<RuntimeSession | void> | RuntimeSession | void;
  onClose?: (stopRuntime: boolean) => void | Promise<void>;
  onDelete?: () => void | Promise<void>;
}> = ({ agentId, runtimeId, initialPrompt, provider, profile, mode = 'Safe', agentName, chrome = 'full', onRecover, onRestartWithMode, onClose, onDelete }) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const [connection, setConnection] = useState<'CONNECTING' | 'CONNECTED' | 'DISCONNECTED' | 'ERROR'>('CONNECTING');
  const [message, setMessage] = useState('');
  const [customTitle, setCustomTitle] = useState('');
  const [recovering, setRecovering] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [boundRuntimeId, setBoundRuntimeId] = useState(runtimeId || '');
  const [connectNonce, setConnectNonce] = useState(0);
  const [askOpen, setAskOpen] = useState(false);
  const [askPrompt, setAskPrompt] = useState('');
  const [availableSkills, setAvailableSkills] = useState<Array<{ id: string; name?: string; description?: string }>>([]);
  const [selectedSkills, setSelectedSkills] = useState<string[]>([]);
  const [asking, setAsking] = useState(false);
  const [askFeedback, setAskFeedback] = useState('');
  const [modeAction, setModeAction] = useState<'Applying' | 'Restarting' | 'Ready' | 'Error' | ''>('');
  const [selectedMode, setSelectedMode] = useState<'Safe' | 'YOLO'>(mode === 'YOLO' ? 'YOLO' : 'Safe');
  const [pendingMode, setPendingMode] = useState<'Safe' | 'YOLO' | null>(null);
  const [closeConfirmOpen, setCloseConfirmOpen] = useState(false);
  const [closing, setClosing] = useState(false);

  useEffect(() => {
    setBoundRuntimeId(runtimeId || '');
  }, [runtimeId]);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    let disposed = false;
    let reconnectAttempt = 0;
    let reconnectTimer: number | undefined;
    let openedOnce = false;
    let stopReconnect = false;
    const roleRef: { current: TerminalRole } = { current: 'VIEW_ONLY' };
    const kickoffRef = { current: normalizeInitialPrompt(initialPrompt), sent: false };

    (container as any).__triggerReconnect = () => {
      stopReconnect = false;
      reconnectAttempt = 0;
      if (reconnectTimer !== undefined) {
        window.clearTimeout(reconnectTimer);
        reconnectTimer = undefined;
      }
      connect();
    };

    const term = new Terminal({
      cursorBlink: true,
      fontSize: 13,
      lineHeight: 1.25,
      fontFamily: 'var(--nx-font-mono)',
      theme: { background: '#090b10' },
      scrollback: 5000,
    });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(container);

    const fitAndResize = () => {
      try {
        if (container.clientWidth > 0 && container.clientHeight > 0) {
          fit.fit();
        }
      } catch {
        return;
      }
      const ws = wsRef.current;
      if (ws?.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'resize', rows: term.rows, cols: term.cols }));
      }
    };

    const maybeSendKickoff = () => {
      const ws = wsRef.current;
      if (
        kickoffRef.current &&
        !kickoffRef.sent &&
        roleRef.current === 'CONTROL' &&
        ws?.readyState === WebSocket.OPEN
      ) {
        kickoffRef.sent = true;
        ws.send(JSON.stringify({ type: 'input', data: kickoffRef.current }));
      }
    };

    const failPermanently = (detail?: string) => {
      stopReconnect = true;
      if (reconnectTimer !== undefined) {
        window.clearTimeout(reconnectTimer);
        reconnectTimer = undefined;
      }
      setConnection('ERROR');
      setMessage(terminalAttachFailureMessage(detail));
    };

    const scheduleReconnect = (detail?: string) => {
      if (disposed || stopReconnect || reconnectTimer !== undefined) return;
      if (detail && isFatalTerminalAttachError(detail)) {
        failPermanently(detail);
        return;
      }
      if (!openedOnce && reconnectAttempt >= TERMINAL_MAX_RECONNECT_ATTEMPTS) {
        failPermanently(detail);
        return;
      }
      if (openedOnce && reconnectAttempt >= TERMINAL_MAX_RECONNECT_ATTEMPTS) {
        failPermanently(detail || 'Conexão com o terminal perdida.');
        return;
      }
      const delay = terminalReconnectDelay(reconnectAttempt++);
      setConnection('CONNECTING');
      setMessage(`Reconnecting Agent terminal in ${delay}ms…`);
      reconnectTimer = window.setTimeout(() => {
        reconnectTimer = undefined;
        connect();
      }, delay);
    };

    const connect = async () => {
      if (disposed || stopReconnect) return;
      setConnection('CONNECTING');
      if (!openedOnce) setMessage('');

      const ws = new WebSocket(
        agentTerminalWebSocketURL(window.location.protocol, window.location.host, agentId, boundRuntimeId)
      );
      wsRef.current = ws;
      roleRef.current = 'VIEW_ONLY';

      ws.onopen = () => {
        openedOnce = true;
        reconnectAttempt = 0;
        setConnection('CONNECTED');
        setModeAction((current) => current === 'Restarting' ? 'Ready' : current);
        setMessage((current) => current === 'Reiniciando runtime…' ? 'Pronto' : '');
        // Lease is acquired by the web handler on Attach for the CONTROL
        // broker seat. Do not spam lease_acquire here — older SessionHosts
        // echoed those RPC frames into the PTY.
        window.requestAnimationFrame(fitAndResize);
      };

      ws.onmessage = (event) => {
        try {
          const payload = JSON.parse(event.data);
          if (payload.type === 'output' && payload.data) {
            const cleaned = scrubProtocolOutput(String(payload.data));
            if (cleaned) term.write(cleaned);
          } else if (payload.type === 'lease') {
            const next = normalizeTerminalRole(payload.role);
            roleRef.current = next;
            setMessage(next === 'CONTROL' ? '' : 'Somente leitura — outro acesso está digitando neste runtime.');
            maybeSendKickoff();
          } else if (payload.type === 'runtime_changed') {
            setMessage('Runtime generation changed — rebinding terminal…');
            ws.close(1012, 'runtime generation changed');
          } else if (payload.type === 'title' && payload.data) {
            setCustomTitle(payload.data);
          } else if (payload.type === 'attention') {
            // Push/sound are owned by the focused-project poll in NexusWorkspaceApp.
            // WS only refreshes local chrome (title) to avoid multi-attach spam.
            if (payload.dynamic_title) setCustomTitle(payload.dynamic_title);
          } else if (payload.type === 'error') {
            const detail = String(payload.data ?? 'Terminal error');
            setMessage(detail);
            if (isFatalTerminalAttachError(detail)) {
              failPermanently(detail);
              ws.close(1011, 'fatal attach error');
            }
          }
        } catch {
          term.write(event.data);
        }
      };

      ws.onerror = () => {
        if (!disposed && !stopReconnect) {
          setConnection('ERROR');
          setMessage('Terminal transport error — reconnecting…');
        }
      };

      ws.onclose = () => {
        if (wsRef.current === ws) wsRef.current = null;
        if (disposed || stopReconnect) return;
        setConnection('DISCONNECTED');
        scheduleReconnect();
      };
    };

    const dataDisposable = term.onData((data) => {
      const ws = wsRef.current;
      if (ws?.readyState === WebSocket.OPEN && roleRef.current === 'CONTROL') {
        ws.send(JSON.stringify({ type: 'input', data }));
      }
    });

    const observer = new ResizeObserver(() => window.requestAnimationFrame(fitAndResize));
    observer.observe(container);
    const visibility = () => {
      if (!document.hidden) window.requestAnimationFrame(fitAndResize);
    };
    document.addEventListener('visibilitychange', visibility);

    connect();

    return () => {
      disposed = true;
      stopReconnect = true;
      if (reconnectTimer !== undefined) window.clearTimeout(reconnectTimer);
      observer.disconnect();
      document.removeEventListener('visibilitychange', visibility);
      dataDisposable.dispose();
      wsRef.current?.close();
      term.dispose();
    };
  }, [agentId, runtimeId, boundRuntimeId, initialPrompt, connectNonce]);

  const handleManualStartOrRecover = async () => {
    setRecovering(true);
    setMessage('Iniciando runtime do agente…');
    try {
      let nextRuntimeId = '';
      if (onRecover) {
        const result = await onRecover();
        nextRuntimeId = result?.runtime_id || '';
      } else {
        const result = await recoverOrStartAgent(agentId);
        nextRuntimeId = result?.runtime?.runtime_id || '';
      }
      if (nextRuntimeId) {
        setBoundRuntimeId(nextRuntimeId);
        setMessage('');
        setConnection('CONNECTING');
        return;
      }
      setConnectNonce((n) => n + 1);
    } catch (e) {
      setMessage(e instanceof Error ? e.message : String(e));
      setConnection('ERROR');
    } finally {
      setRecovering(false);
    }
  };

  const handleDeleteAgent = async () => {
    if (!onDelete || deleting) return;
    const label = agentName || agentId;
    if (!window.confirm(`Remover o agente "${label}"? A identidade e o terminal serão excluídos.`)) return;
    setDeleting(true);
    setMessage('Removendo agente…');
    try {
      await onDelete();
    } catch (e) {
      setMessage(e instanceof Error ? e.message : String(e));
      setConnection('ERROR');
    } finally {
      setDeleting(false);
    }
  };

  const loadSkills = async () => {
    try {
      const status = await nexus.getMaestroStatus();
      if (status?.capabilities?.skills) {
        setAvailableSkills(status.capabilities.skills);
      }
    } catch {
      setAvailableSkills([]);
    }
  };

  const handleToggleSkill = (id: string) => {
    setSelectedSkills((prev) =>
      prev.includes(id) ? prev.filter((s) => s !== id) : prev.length < 3 ? [...prev, id] : prev
    );
  };

  const handleSendPrompt = async () => {
    if (!askPrompt.trim() || asking) return;
    setAsking(true);
    setAskFeedback('');
    try {
      await nexus.askAgent(agentId, askPrompt.trim(), true, selectedSkills.length > 0 ? selectedSkills : undefined);
      setAskPrompt('');
      setSelectedSkills([]);
      setAskOpen(false);
      setAskFeedback('Prompt enviado');
    } catch (err) {
      setAskFeedback(err instanceof Error ? err.message : String(err));
    } finally {
      setAsking(false);
    }
  };

  const requestModeChange = (nextMode: 'Safe' | 'YOLO') => {
    if (!onRestartWithMode || nextMode === selectedMode || modeAction === 'Applying' || modeAction === 'Restarting') return;
    setPendingMode(nextMode);
  };

  const handleModeChange = async () => {
    if (!onRestartWithMode || !pendingMode || modeAction === 'Applying' || modeAction === 'Restarting') return;
    const nextMode = pendingMode;
    setPendingMode(null);
    setModeAction('Applying');
    setMessage('Salvando configuração…');
    try {
      const nextRuntime = await onRestartWithMode(nextMode);
      setModeAction('Restarting');
      setMessage('Reiniciando runtime…');
      if (nextRuntime?.runtime_id) {
        setBoundRuntimeId(nextRuntime.runtime_id);
      } else {
        setConnectNonce((nonce) => nonce + 1);
      }
      setSelectedMode(nextMode);
    } catch (error) {
      setModeAction('Error');
      setMessage(error instanceof Error ? error.message : String(error));
    }
  };

  const confirmClose = async (stopRuntime: boolean) => {
    if (!onClose || closing) return;
    setClosing(true);
    try {
      await onClose(stopRuntime);
    } finally {
      setClosing(false);
      setCloseConfirmOpen(false);
    }
  };

  const displayName = customTitle || agentName || agentId;
  const isStaleOrDisconnected = connection === 'ERROR' || connection === 'DISCONNECTED';
  const windowChrome = chrome === 'window';

  const modeButtons = (
    <div className="nx-agent-terminal__modes" role="group" aria-label="Modo de execução">
      {(['Safe', 'YOLO'] as const).map((m) => (
        <button
          key={m}
          type="button"
          className="nx-agent-terminal__mode"
          data-active={selectedMode === m ? 'true' : 'false'}
          onClick={() => requestModeChange(m)}
          disabled={!onRestartWithMode || modeAction === 'Applying' || modeAction === 'Restarting'}
          title={`Alternar modo para ${m}`}
        >
          {m}
        </button>
      ))}
    </div>
  );

  const terminalActions = (
    <div className="nx-agent-terminal__controls">
      {message && !isStaleOrDisconnected && (
        <span className="nx-terminal-message">
          {modeAction === 'Error' ? <ShieldAlert size={12} /> : <Unplug size={12} />}
          {message}
        </span>
      )}
      <button
        type="button"
        className="nx-agent-terminal__ask-btn"
        onClick={() => {
          const next = !askOpen;
          setAskOpen(next);
          if (next && availableSkills.length === 0) {
            void loadSkills();
          }
        }}
        title="Perguntar ao Agente / Sugerir skills"
      >
        <Sparkles size={13} />
        <span>Perguntar</span>
      </button>
      {onClose && !windowChrome && (
        <button type="button" onClick={() => setCloseConfirmOpen(true)} title="Escolher como fechar este terminal">
          Fechar terminal
        </button>
      )}
    </div>
  );

  return (
    <section className="nx-agent-terminal" data-chrome={chrome} aria-label={`Terminal for Agent ${agentId}`} style={{ position: 'relative' }}>
      {!windowChrome && (
        <header className="nx-agent-terminal__header">
          <div className="nx-agent-terminal__identity">
            <span style={{ fontSize: '10px', padding: '1px 5px', borderRadius: '4px', background: 'var(--nx-accent)', color: 'white', fontWeight: 700 }}>
              AGENT
            </span>
            <strong style={{ fontSize: '12px', color: 'var(--nx-text-primary)' }} title={agentId}>
              {displayName}
            </strong>
            {provider && (
              <span style={{ fontSize: '11px', color: 'var(--nx-text-soft)', border: '1px solid var(--nx-border)', padding: '1px 6px', borderRadius: '4px' }}>
                nexus {provider}{profile && profile !== 'default' ? `:${profile}` : ''}
              </span>
            )}
            <span data-state={connection}>{connection.toLowerCase()}</span>
          </div>
          <div className="nx-agent-terminal__status">
            {modeButtons}
            {terminalActions}
          </div>
        </header>
      )}
      {windowChrome && (
        <div className="nx-agent-terminal__toolbar nx-agent-terminal__toolbar--actions" aria-label="Terminal actions">
          {modeButtons}
          {terminalActions}
        </div>
      )}
      {askOpen && (
        <div className="nx-agent-terminal__composer" style={{ padding: '8px 12px', background: 'var(--nx-surface-2)', borderBottom: '1px solid var(--nx-border)', display: 'grid', gap: '8px' }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <span style={{ fontSize: '12px', fontWeight: 600, display: 'flex', alignItems: 'center', gap: '6px' }}>
              <MessageSquare size={13} /> Enviar instrução ao agente (One-shot)
            </span>
            <button type="button" onClick={() => setAskOpen(false)} style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--nx-muted)' }}>
              <X size={13} />
            </button>
          </div>
          <div style={{ display: 'flex', gap: '8px' }}>
            <input
              type="text"
              className="nx-input"
              style={{ flex: 1, fontSize: '12px', padding: '6px 10px' }}
              placeholder="Digite o objetivo ou comando para o agente..."
              value={askPrompt}
              onChange={(e) => setAskPrompt(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault();
                  void handleSendPrompt();
                }
              }}
            />
            <button
              type="button"
              className="nx-button"
              data-tone="brand"
              data-size="sm"
              disabled={asking || !askPrompt.trim()}
              onClick={() => void handleSendPrompt()}
            >
              <Send size={12} />
              <span>{asking ? 'Enviando...' : 'Enviar'}</span>
            </button>
          </div>
          {availableSkills.length > 0 && (
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px', alignItems: 'center' }}>
              <span style={{ fontSize: '11px', color: 'var(--nx-muted)' }}>Skills Maestro (até 3 no próximo prompt):</span>
              {availableSkills.slice(0, 8).map((s) => {
                const active = selectedSkills.includes(s.id);
                return (
                  <button
                    key={s.id}
                    type="button"
                    onClick={() => handleToggleSkill(s.id)}
                    style={{
                      fontSize: '11px',
                      padding: '2px 8px',
                      borderRadius: '12px',
                      border: '1px solid',
                      borderColor: active ? 'var(--nx-accent)' : 'var(--nx-border)',
                      background: active ? 'var(--nx-accent-soft)' : 'var(--nx-surface-3)',
                      color: active ? 'var(--nx-text)' : 'var(--nx-text-soft)',
                      cursor: 'pointer',
                    }}
                    title={s.description || s.name || s.id}
                  >
                    {active ? '✓ ' : '+ '}{s.name || s.id}
                  </button>
                );
              })}
            </div>
          )}
          {askFeedback && (
            <span style={{ fontSize: '11px', color: askFeedback.includes('erro') || askFeedback.includes('failed') ? 'var(--nx-danger)' : 'var(--nx-accent)' }}>
              {askFeedback}
            </span>
          )}
        </div>
      )}
      {closeConfirmOpen && onClose && (
        <TerminalActionDialog
          close
          busy={closing}
          onCancel={() => setCloseConfirmOpen(false)}
          onCloseTab={() => void confirmClose(false)}
          onStopRuntime={() => void confirmClose(true)}
        />
      )}
      <TerminalActionDialog
        mode={pendingMode || undefined}
        busy={modeAction === 'Applying' || modeAction === 'Restarting'}
        onCancel={() => setPendingMode(null)}
        onConfirmMode={() => void handleModeChange()}
      />
      <div ref={containerRef} className="nx-agent-terminal__xterm" />

      {isStaleOrDisconnected && (
        <div
          style={{
            position: 'absolute',
            inset: windowChrome ? 0 : '39px 0 0 0',
            background: 'rgba(9, 11, 16, 0.88)',
            backdropFilter: 'blur(3px)',
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            gap: '12px',
            padding: '24px',
            zIndex: 10,
            textAlign: 'center',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px', color: 'var(--nx-warning)' }}>
            <Unplug size={18} />
            <strong style={{ fontSize: '14px' }}>Runtime do Agente desconectado</strong>
          </div>
          <p style={{ maxWidth: '420px', fontSize: '12px', color: 'var(--nx-muted)', margin: 0, lineHeight: 1.5 }}>
            O processo do agente não está rodando no momento ou foi finalizado. Inicie o runtime para anexar o terminal e executar comandos.
          </p>
          {message && (
            <p
              style={{
                maxWidth: '440px',
                margin: 0,
                padding: '8px 10px',
                borderRadius: 6,
                border: '1px solid color-mix(in srgb, var(--nx-danger) 40%, var(--nx-border))',
                background: 'var(--nx-danger-soft)',
                color: 'var(--nx-danger)',
                fontSize: '12px',
                lineHeight: 1.45,
              }}
            >
              {message}
            </p>
          )}
          <div style={{ display: 'flex', gap: '8px', marginTop: '4px', flexWrap: 'wrap', justifyContent: 'center' }}>
            <button
              type="button"
              className="nx-button"
              data-tone="brand"
              disabled={recovering}
              onClick={handleManualStartOrRecover}
              style={{ display: 'inline-flex', alignItems: 'center', gap: '6px' }}
            >
              {recovering ? <RefreshCw size={13} className="nx-spin-slow" /> : <Play size={13} />}
              <span>{recovering ? 'Iniciando…' : 'Iniciar / Recuperar Agente'}</span>
            </button>
            <button
              type="button"
              className="nx-button"
              onClick={() => {
                if (containerRef.current && (containerRef.current as any).__triggerReconnect) {
                  (containerRef.current as any).__triggerReconnect();
                } else {
                  setConnectNonce((n) => n + 1);
                }
              }}
              style={{ display: 'inline-flex', alignItems: 'center', gap: '6px' }}
            >
              <RefreshCw size={13} />
              <span>Tentar Conectar</span>
            </button>
            {onDelete && (
              <button
                type="button"
                className="nx-button"
                data-tone="danger"
                disabled={recovering || deleting}
                onClick={() => void handleDeleteAgent()}
                style={{ display: 'inline-flex', alignItems: 'center', gap: '6px' }}
              >
                <Trash2 size={13} />
                <span>{deleting ? 'Removendo…' : 'Remover Agente'}</span>
              </button>
            )}
          </div>
        </div>
      )}
    </section>
  );
};
