import React from 'react';
import { TerminalPane } from '../../components/TerminalPane';
import { api } from '../../api';

export const ProjectShellSurface: React.FC<{ runtimeId: string; title?: string; onRuntimeChanged?: () => void | Promise<void> }> = ({ runtimeId, title = 'Project Shell', onRuntimeChanged }) => (
  <div className="nx-agent-terminal-surface nx-project-shell-surface" data-runtime-id={runtimeId}>
    <TerminalPane
      runtimeId={runtimeId}
      title={title}
      provider="shell"
      profile="local"
      onUpdateTitle={async (id, nextTitle) => { await api.updateRuntimeTitle(id, nextTitle); await onRuntimeChanged?.(); }}
    />
  </div>
);
