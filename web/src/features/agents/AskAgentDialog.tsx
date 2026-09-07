import React from 'react';
import { TaskPreparationDialog } from '../../components/TaskPreparationDialog';
import { nexus } from '../../nexus/api';
import type { Agent } from '../../types';
import { askActionForStatus } from './askAgentModel';

export const AskAgentDialog: React.FC<{
  agent: Agent | null;
  onClose: () => void;
  onSent?: (agent: Agent) => void | Promise<void>;
}> = ({ agent, onClose, onSent }) => {
  const action = askActionForStatus(agent?.status || 'STOPPED');
  return (
    <TaskPreparationDialog
      open={Boolean(agent)}
      agentId={agent?.id || ''}
      agentName={agent?.name}
      projectId={agent?.project_id}
      startIfNeeded={action.startIfNeeded}
      onClose={onClose}
      onSubmit={async (prompt, skills, startIfNeeded, contextFingerprintId) => {
        if (!agent) return;
        await nexus.askAgent(agent.id, prompt, startIfNeeded, skills, {
          projectId: agent.project_id,
          contextFingerprintId: contextFingerprintId || '',
        });
        await onSent?.(agent);
      }}
    />
  );
};
