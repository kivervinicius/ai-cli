import React from 'react';
import { AlertTriangle, RotateCcw, Square } from 'lucide-react';
import { Button, Dialog } from '../design-system';

export const TerminalActionDialog: React.FC<{
  mode?: 'Safe' | 'YOLO';
  close?: boolean;
  shell?: boolean;
  busy?: boolean;
  onCancel: () => void;
  onConfirmMode?: () => void;
  onCloseTab?: () => void;
  onStopRuntime?: () => void;
}> = ({ mode, close, shell, busy, onCancel, onConfirmMode, onCloseTab, onStopRuntime }) => {
  if (!mode && !close) return null;
  if (mode) {
    return (
      <Dialog open title={`Aplicar modo ${mode}`} onClose={onCancel}>
        <div className="nx-terminal-action-dialog">
          <div className="nx-terminal-action-dialog__icon" data-tone={mode === 'YOLO' ? 'danger' : 'warning'}><RotateCcw size={18} /></div>
          <div>
            <strong>O runtime será reiniciado</strong>
            <p>A configuração atual será preservada e o Agente será reconectado no modo <b>{mode}</b>. Durante o reinício, o terminal ficará indisponível por alguns instantes.</p>
          </div>
          <div className="nx-dialog-actions">
            <Button onClick={onCancel} disabled={busy}>Cancelar</Button>
            <Button tone={mode === 'YOLO' ? 'danger' : 'brand'} onClick={onConfirmMode} disabled={busy}>{busy ? 'Aplicando…' : `Aplicar ${mode} e reiniciar`}</Button>
          </div>
        </div>
      </Dialog>
    );
  }
  return (
    <Dialog open title={shell ? 'Fechar Project Shell' : 'Fechar terminal'} onClose={onCancel}>
      <div className="nx-terminal-action-dialog">
        <div className="nx-terminal-action-dialog__icon" data-tone="warning"><AlertTriangle size={18} /></div>
        <div>
          <strong>{shell ? 'O processo da shell será encerrado' : 'O que deseja fechar?'}</strong>
          <p>{shell ? 'Fechar esta Project Shell encerra o processo correspondente.' : 'Fechar a aba visual não precisa parar o Agente persistente.'}</p>
        </div>
        <div className="nx-terminal-action-dialog__choices">
          {!shell && <Button onClick={onCloseTab} disabled={busy}>Fechar somente a aba</Button>}
          <Button tone="danger" onClick={onStopRuntime} disabled={busy}><Square size={13} />{busy ? 'Encerrando…' : shell ? 'Fechar e encerrar processo' : 'Fechar aba e parar runtime'}</Button>
          <Button onClick={onCancel} disabled={busy}>Cancelar</Button>
        </div>
      </div>
    </Dialog>
  );
};
