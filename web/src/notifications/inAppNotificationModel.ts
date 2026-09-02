import type { RuntimeSession } from '../types';

export type InAppNotification = {
  id: string;
  runtimeId: string;
  projectName: string;
  title: string;
  message: string;
  tone: 'success' | 'danger';
};

export function notificationFromRuntime(runtime: RuntimeSession): InAppNotification | null {
  const reason = runtime.attention_reason;
  if (reason !== 'TASK_COMPLETED' && reason !== 'ERROR') return null;

  const completed = reason === 'TASK_COMPLETED';
  const message = runtime.attention_context || runtime.last_task_summary || runtime.dynamic_title ||
    (completed ? 'A tarefa foi concluída.' : 'O terminal reportou um erro.');

  return {
    id: `${runtime.runtime_id}:${reason}:${message}`,
    runtimeId: runtime.runtime_id,
    projectName: runtime.project_name || 'Projeto',
    title: completed ? 'Tarefa concluída' : 'Erro no terminal',
    message,
    tone: completed ? 'success' : 'danger',
  };
}
