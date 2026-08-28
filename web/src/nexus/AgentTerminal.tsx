import React, { useEffect, useRef, useState } from 'react';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';

/* Agent-scoped terminal: stable across runtime generations (§30-31).
   WS target /api/v1/agents/:id/terminal resolves the current generation. */

export const AgentTerminal: React.FC<{ agentId: string; onClose?: () => void }> = ({ agentId, onClose }) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const termRef = useRef<Terminal | null>(null);
  const fitRef = useRef<FitAddon | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const [role, setRole] = useState<'CONTROL' | 'VIEW_ONLY'>('VIEW_ONLY');
  const [errorMsg, setErrorMsg] = useState<string>('');

  useEffect(() => {
    if (!containerRef.current) return;
    const term = new Terminal({ cursorBlink: true, fontSize: 13, fontFamily: 'var(--nx-font-mono)' });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(containerRef.current);
    fit.fit();
    termRef.current = term;
    fitRef.current = fit;

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const ws = new WebSocket(`${protocol}//${window.location.host}/api/v1/agents/${agentId}/terminal`);
    wsRef.current = ws;

    const sendResize = () => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'resize', rows: term.rows, cols: term.cols }));
      }
    };

    const onData = (data: string) => {
      if (ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify({ type: 'input', data }));
    };

    ws.onopen = () => {
      setErrorMsg('');
      fit.fit();
      sendResize();
      term.onData(onData);
      term.onResize(sendResize);
    };

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        if (msg.type === 'output' && msg.data) {
          term.write(msg.data);
        } else if (msg.type === 'lease') {
          setRole(msg.role === 'CONTROL' ? 'CONTROL' : 'VIEW_ONLY');
        } else if (msg.type === 'error') {
          setErrorMsg(msg.data);
        }
      } catch {
        term.write(event.data);
      }
    };

    ws.onerror = () => setErrorMsg('WebSocket connection error');
    ws.onclose = () => setErrorMsg('Terminal disconnected (agent runtime may have stopped)');

    const onResize = () => fit.fit();
    window.addEventListener('resize', onResize);
    return () => {
      window.removeEventListener('resize', onResize);
      ws.close();
      term.dispose();
    };
  }, [agentId]);

  return (
    <div className="flex flex-col h-full rounded-[var(--nx-radius)] border border-slate-800/80 bg-black/40 overflow-hidden">
      <div className="flex items-center justify-between px-3 py-1.5 border-b border-slate-800/80 text-[11px] font-mono">
        <span className="text-slate-400 truncate">{agentId}</span>
        <div className="flex items-center gap-2">
          <span
            className={`inline-flex items-center gap-1 ${
              role === 'CONTROL' ? 'text-emerald-400' : 'text-slate-500'
            }`}
          >
            <span className="w-1.5 h-1.5 rounded-full bg-current" />
            {role}
          </span>
          {errorMsg && <span className="text-amber-400">{errorMsg}</span>}
          {onClose && (
            <button onClick={onClose} className="text-slate-500 hover:text-slate-200 px-1" aria-label="Detach">
              ×
            </button>
          )}
        </div>
      </div>
      <div ref={containerRef} className="flex-1 min-h-0" />
    </div>
  );
};
