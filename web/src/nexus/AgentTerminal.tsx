import React, { useEffect, useRef, useState } from 'react';
import { Eye, Keyboard, Unplug } from 'lucide-react';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import { agentTerminalWebSocketURL, normalizeTerminalRole, type TerminalRole } from './agentTerminalModel';

export const AgentTerminal: React.FC<{ agentId: string; onClose?: () => void }> = ({ agentId, onClose }) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [role, setRole] = useState<TerminalRole>('VIEW_ONLY');
  const [connection, setConnection] = useState<'CONNECTING' | 'CONNECTED' | 'DISCONNECTED' | 'ERROR'>('CONNECTING');
  const [message, setMessage] = useState('');

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;
    const roleRef: { current: TerminalRole } = { current: 'VIEW_ONLY' };
    const term = new Terminal({ cursorBlink: true, fontSize: 13, lineHeight: 1.25, fontFamily: 'var(--nx-font-mono)', theme: { background: '#090b10' }, scrollback: 5000 });
    const fit = new FitAddon();
    term.loadAddon(fit); term.open(container);
    const ws = new WebSocket(agentTerminalWebSocketURL(window.location.protocol, window.location.host, agentId));
    const fitAndResize = () => {
      try { fit.fit(); } catch { return; }
      if (ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify({ type: 'resize', rows: term.rows, cols: term.cols }));
    };
    const dataDisposable = term.onData((data) => { if (ws.readyState === WebSocket.OPEN && roleRef.current === 'CONTROL') ws.send(JSON.stringify({ type: 'input', data })); });
    ws.onopen = () => { setConnection('CONNECTED'); setMessage(''); window.requestAnimationFrame(fitAndResize); };
    ws.onmessage = (event) => {
      try {
        const payload = JSON.parse(event.data);
        if (payload.type === 'output' && payload.data) term.write(payload.data);
        else if (payload.type === 'lease') { const next = normalizeTerminalRole(payload.role); roleRef.current = next; setRole(next); }
        else if (payload.type === 'error') setMessage(String(payload.data ?? 'Terminal error'));
      } catch { term.write(event.data); }
    };
    ws.onerror = () => { setConnection('ERROR'); setMessage('WebSocket connection error'); };
    ws.onclose = () => { setConnection('DISCONNECTED'); setMessage('Agent runtime disconnected'); };
    const observer = new ResizeObserver(() => window.requestAnimationFrame(fitAndResize));
    observer.observe(container);
    const visibility = () => { if (!document.hidden) window.requestAnimationFrame(fitAndResize); };
    document.addEventListener('visibilitychange', visibility);
    return () => { observer.disconnect(); document.removeEventListener('visibilitychange', visibility); dataDisposable.dispose(); ws.close(); term.dispose(); };
  }, [agentId]);

  return <section className="nx-agent-terminal" aria-label={`Terminal for Agent ${agentId}`}>
    <header className="nx-agent-terminal__header"><div className="nx-agent-terminal__identity"><code>{agentId}</code><span data-state={connection}>{connection.toLowerCase()}</span></div><div className="nx-agent-terminal__status"><span data-role={role}>{role === 'CONTROL' ? <Keyboard size={13} /> : <Eye size={13} />}{role}</span>{message && <span className="nx-terminal-message"><Unplug size={12} />{message}</span>}{onClose && <button type="button" onClick={onClose}>Detach</button>}</div></header>
    <div ref={containerRef} className="nx-agent-terminal__xterm" />
  </section>;
};
