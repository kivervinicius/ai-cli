import type { WorkspaceModel, WorkspaceNode } from './model';

function validNode(node: unknown): node is WorkspaceNode {
  if (!node || typeof node !== 'object') return false;
  const candidate = node as Partial<WorkspaceNode> & Record<string, unknown>;
  if (candidate.kind === 'stack') {
    return typeof candidate.id === 'string' && Array.isArray(candidate.tabs) && typeof candidate.activeId === 'string';
  }
  if (candidate.kind === 'split') {
    return typeof candidate.id === 'string' && (candidate.direction === 'horizontal' || candidate.direction === 'vertical') && typeof candidate.ratio === 'number' && validNode(candidate.first) && validNode(candidate.second);
  }
  return false;
}

export const workspaceStorageKey = (projectId: string) => `iapro:nexus:workspace:${projectId}:v1`;

export function serializeWorkspace(model: WorkspaceModel): string {
  return JSON.stringify(model);
}

export function deserializeWorkspace(raw: string | null | undefined, fallback: WorkspaceModel): WorkspaceModel {
  if (!raw) return fallback;
  try {
    const parsed = JSON.parse(raw) as WorkspaceModel;
    return parsed?.version === 1 && validNode(parsed.root) ? parsed : fallback;
  } catch {
    return fallback;
  }
}
