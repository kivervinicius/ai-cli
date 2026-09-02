import { describe, expect, it } from 'vitest';
import { notificationFromRuntime } from './inAppNotificationModel';

describe('notificationFromRuntime', () => {
  const runtime = {
    runtime_id: 'runtime-1',
    workspace: '/tmp/project',
    pid: 1,
    host_pid: 1,
    state: 'RUNNING' as const,
    control_level: 'FULL',
    control_endpoint: '',
    started_at: '',
    project_name: 'Projeto',
  };

  it('creates a transient notification for completion', () => {
    expect(notificationFromRuntime({ ...runtime, attention_reason: 'TASK_COMPLETED', attention_context: 'Build finalizado.' }))
      .toMatchObject({ title: 'Tarefa concluída', tone: 'success', message: 'Build finalizado.' });
  });

  it('keeps interactive attention out of transient notifications', () => {
    expect(notificationFromRuntime({ ...runtime, attention_reason: 'QUESTION' })).toBeNull();
  });
});
