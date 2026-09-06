export function notificationTitle(
  projectName: string | undefined,
  reason: string,
  dynamicTitle?: string,
): string {
  const project = projectName?.trim() || 'Projeto';
  const dynamic = dynamicTitle?.trim();
  if (dynamic) return `[Nexus | ${project}] ${dynamic}`;
  if (reason === 'QUESTION') return `[Nexus | ${project}] ❓ Pergunta Requer Atenção`;
  if (reason === 'APPROVAL') return `[Nexus | ${project}] ⚠️ Aprovação Necessária`;
  if (reason === 'TASK_COMPLETED') return `[Nexus | ${project}] ✅ Tarefa Concluída!`;
  return `[Nexus | ${project}] Notificação`;
}
