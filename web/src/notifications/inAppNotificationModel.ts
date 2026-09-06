import type { RuntimeSession } from '../types';

export type InAppNotification = {
  id: string;
  runtimeId: string;
  projectName: string;
  title: string;
  message: string;
  tone: 'success' | 'danger' | 'warning';
};

export function notificationFromRuntime(runtime: RuntimeSession): InAppNotification | null {
  const reason = runtime.attention_reason;
  if (
    reason !== 'TASK_COMPLETED' &&
    reason !== 'ERROR' &&
    reason !== 'QUESTION' &&
    reason !== 'APPROVAL'
  ) {
    return null;
  }

  const message =
    runtime.attention_context ||
    runtime.last_task_summary ||
    runtime.dynamic_title ||
    (reason === 'TASK_COMPLETED'
      ? 'A tarefa foi concluída.'
      : reason === 'ERROR'
        ? 'O terminal reportou um erro.'
        : 'O agente espera sua resposta.');

  const projectName = runtime.project_name || 'Projeto';
  if (reason === 'TASK_COMPLETED') {
    return {
      id: `${runtime.runtime_id}:${reason}:${message}`,
      runtimeId: runtime.runtime_id,
      projectName,
      title: 'Tarefa concluída',
      message,
      tone: 'success',
    };
  }
  if (reason === 'ERROR') {
    return {
      id: `${runtime.runtime_id}:${reason}:${message}`,
      runtimeId: runtime.runtime_id,
      projectName,
      title: 'Erro no terminal',
      message,
      tone: 'danger',
    };
  }
  return {
    id: `${runtime.runtime_id}:${reason}:${message}`,
    runtimeId: runtime.runtime_id,
    projectName,
    title: reason === 'APPROVAL' ? 'Aprovação necessária' : 'Confirmação pendente',
    message,
    tone: 'warning',
  };
}
