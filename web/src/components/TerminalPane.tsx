import React, { useEffect, useRef, useState } from 'react';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import { Shield, ShieldAlert, Maximize2, XSquare } from 'lucide-react';

interface TerminalPaneProps {
  runtimeId: string;
  provider: string;
  profile: string;
  onClose?: () => void;
}

export const TerminalPane: React.FC<TerminalPaneProps> = ({
  runtimeId,
  provider,
  profile,
  onClose,
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const termRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const wsRef = useRef<WebSocket | null>(null);

  const [role, setRole] = useState<'CONTROL' | 'VIEW_ONLY'>('VIEW_ONLY');
  const [connected, setConnected] = useState<boolean>(false);
  const [errorMsg, setErrorMsg] = useState<string>('');

  useEffect(() => {
    if (!containerRef.current) return;

    // Initialize xterm
    const term = new Terminal({
      theme: {
        background: '#090d16',
        foreground: '#e2e8f0',
        cursor: '#38bdf8',
        selectionBackground: 'rgba(56, 189, 248, 0.3)',
      },
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
      fontSize: 13,
      lineHeight: 1.2,
      cursorBlink: true,
      cursorStyle: 'block',
    });

    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.open(containerRef.current);
    fitAddon.fit();

    termRef.current = term;
    fitAddonRef.current = fitAddon;

    // Connect WebSocket
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/v1/runtimes/${runtimeId}/terminal`;
    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      setConnected(true);
      setErrorMsg('');
      fitAddon.fit();
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'resize', rows: term.rows, cols: term.cols }));
      }
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
        // Raw bytes fallback
        term.write(event.data);
      }
    };

    ws.onclose = () => {
      setConnected(false);
    };

    ws.onerror = () => {
      setConnected(false);
      setErrorMsg('WebSocket connection error');
    };

    // Forward terminal input to WebSocket
    const dataListener = term.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'input', data }));
      }
    });

    // Resize handling
    const resizeListener = term.onResize(({ rows, cols }) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'resize', rows, cols }));
      }
    });

    const handleWindowResize = () => {
      fitAddon.fit();
    };
    window.addEventListener('resize', handleWindowResize);

    return () => {
      dataListener.dispose();
      resizeListener.dispose();
      window.removeEventListener('resize', handleWindowResize);
      ws.close();
      term.dispose();
    };
  }, [runtimeId]);

  const requestControl = () => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({ type: 'lease_acquire' }));
    }
  };

  const releaseControl = () => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({ type: 'lease_release' }));
    }
  };

  return (
    <div className="flex flex-col h-full bg-[#090d16] border border-slate-800 rounded-lg overflow-hidden shadow-xl">
      {/* Terminal Toolbar */}
      <div className="flex items-center justify-between px-3 py-2 bg-slate-900 border-b border-slate-800 text-xs font-mono select-none">
        <div className="flex items-center space-x-2">
          <span className="w-2.5 h-2.5 rounded-full bg-emerald-500 animate-pulse"></span>
          <span className="text-slate-200 font-bold uppercase">{provider}</span>
          <span className="text-slate-500">({profile})</span>
          <span className="text-slate-600">|</span>
          <span className="text-slate-400">{runtimeId}</span>
        </div>

        <div className="flex items-center space-x-3">
          {errorMsg && <span className="text-rose-400 font-sans text-xs">{errorMsg}</span>}

          {/* Lease Status Badge */}
          {role === 'CONTROL' ? (
            <span className="inline-flex items-center px-2 py-0.5 rounded text-[11px] font-semibold bg-emerald-950 text-emerald-300 border border-emerald-800">
              <Shield className="w-3 h-3 mr-1" />
              CONTROL
            </span>
          ) : (
            <span className="inline-flex items-center px-2 py-0.5 rounded text-[11px] font-semibold bg-amber-950 text-amber-300 border border-amber-800">
              <ShieldAlert className="w-3 h-3 mr-1" />
              VIEW ONLY
            </span>
          )}

          {/* Lease Actions */}
          {role === 'VIEW_ONLY' ? (
            <button
              onClick={requestControl}
              className="px-2 py-0.5 rounded bg-sky-600 hover:bg-sky-500 text-white font-sans text-xs transition"
            >
              Take Control
            </button>
          ) : (
            <button
              onClick={releaseControl}
              className="px-2 py-0.5 rounded bg-slate-800 hover:bg-slate-700 text-slate-300 font-sans text-xs transition"
            >
              Release
            </button>
          )}

          {onClose && (
            <button
              onClick={onClose}
              className="p-1 rounded text-slate-400 hover:text-white hover:bg-slate-800 transition"
              title="Close Pane"
            >
              <XSquare className="w-4 h-4" />
            </button>
          )}
        </div>
      </div>

      {/* xterm container */}
      <div className="flex-1 w-full p-2 overflow-hidden" ref={containerRef} />
    </div>
  );
};
