import { normalizeSurface, type WorkspaceModel, type WorkspaceNode } from './model';

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

export const workspaceStorageKey = (projectId: string) => `iapro:nexus:workspace:${projectId}:v2`;

export function serializeWorkspace(model: WorkspaceModel): string {
  return JSON.stringify(model);
}

export function deserializeWorkspace(raw: string | null | undefined, fallback: WorkspaceModel): WorkspaceModel {
  if (!raw) return fallback;
  try {
    const parsed = JSON.parse(raw) as { version?: number; root?: WorkspaceNode; focusedStackId?: string; maximizedSurfaceId?: string };
    if (parsed?.version === 2 && validNode(parsed.root) && typeof parsed.focusedStackId === 'string') return normalizeWorkspace(parsed as WorkspaceModel);
    if (parsed?.version === 1 && validNode(parsed.root)) return migrateV1(parsed.root, parsed.maximizedSurfaceId);
    return fallback;
  } catch {
    return fallback;
  }
}

function normalizeWorkspace(model: WorkspaceModel): WorkspaceModel {
  const normalizeNode = (node: WorkspaceNode): WorkspaceNode => node.kind === 'stack'
    ? { ...node, tabs: node.tabs.map(normalizeSurface), activeId: node.tabs.some((tab) => (tab.viewId || tab.id) === node.activeId) ? node.activeId : (node.tabs[0]?.viewId || node.tabs[0]?.id || '') }
    : { ...node, first: normalizeNode(node.first), second: normalizeNode(node.second) };
  const root = normalizeNode(model.root);
  const stacks = listStacksForState(root);
  return { ...model, root, focusedStackId: stacks.some((stack) => stack.id === model.focusedStackId) ? model.focusedStackId : stacks[0]?.id || '' };
}

function migrateV1(root: WorkspaceNode, maximizedSurfaceId?: string): WorkspaceModel {
  const model: WorkspaceModel = { version: 2, root, focusedStackId: '', maximizedSurfaceId, maximizedViewId: maximizedSurfaceId };
  const normalized = normalizeWorkspace(model);
  const seen = new Set<string>();
  const dedup = (node: WorkspaceNode): WorkspaceNode => node.kind === 'stack'
    ? { ...node, tabs: node.tabs.filter((tab) => { const key = tab.logicalKey || tab.id; if (seen.has(key)) return false; seen.add(key); return true; }), activeId: node.activeId }
    : { ...node, first: dedup(node.first), second: dedup(node.second) };
  return normalizeWorkspace({ ...normalized, root: dedup(normalized.root) });
}

function listStacksForState(node: WorkspaceNode): Array<{ id: string }> {
  return node.kind === 'stack' ? [node] : [...listStacksForState(node.first), ...listStacksForState(node.second)];
}
