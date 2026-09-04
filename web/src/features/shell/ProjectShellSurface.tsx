import React from 'react';
import { TerminalPane } from '../../components/TerminalPane';
import { api } from '../../api';
import { useWorkspacePresentation } from '../../workspace/WorkspacePresentationProvider';

export const ProjectShellSurface: React.FC<{
  runtimeId: string;
  title?: string;
  onRuntimeChanged?: () => void | Promise<void>;
}> = ({ runtimeId, title = 'Project Shell', onRuntimeChanged }) => {
  const presentation = useWorkspacePresentation();
  const hideHeader = presentation.state.mode === 'DESKTOP';
  return (
    <div className="nx-agent-terminal-surface nx-project-shell-surface" data-runtime-id={runtimeId} data-chrome={hideHeader ? 'window' : 'full'}>
      <TerminalPane
        runtimeId={runtimeId}
        title={title}
        provider="shell"
        profile="local"
        hideHeader={hideHeader}
        onUpdateTitle={async (id, nextTitle) => {
          await api.updateRuntimeTitle(id, nextTitle);
          await onRuntimeChanged?.();
        }}
      />
    </div>
  );
};
