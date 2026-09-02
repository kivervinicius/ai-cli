import { normalizeSurface, surfaceLogicalKey, type WorkspaceModel, type WorkspaceNode } from './model';

function validNode(node: unknown): node is WorkspaceNode {
  if (!node || typeof node !== 'object') return false;
  const candidate = node as Partial<WorkspaceNode> & Record<string, unknown>;
  if (candidate.kind === 'stack') return typeof candidate.id === 'string' && Array.isArray(candidate.tabs) && typeof candidate.activeId === 'string';
  if (candidate.kind === 'split') return typeof candidate.id === 'string' && (candidate.direction === 'horizontal' || candidate.direction === 'vertical') && typeof candidate.ratio === 'number' && validNode(candidate.first) && validNode(candidate.second);
  return false;
}

export const workspaceStorageKey = (projectId: string) => `iapro:nexus:workspace:${projectId}:v2`;

export function serializeWorkspace(model: WorkspaceModel): string { return JSON.stringify(model); }

function normalizeNode(node: WorkspaceNode, seen: Set<string>): WorkspaceNode {
  if (node.kind === 'stack') {
    const tabs = node.tabs.map(normalizeSurface).filter((tab) => {
      const key = surfaceLogicalKey(tab);
      if (seen.has(key)) return false;
      seen.add(key);
      return true;
    });
    const active = tabs.find((tab) => tab.id === node.activeId || tab.viewId === node.activeId || tab.legacyId === node.activeId || tab.logicalKey === node.activeId);
    return { ...node, tabs, activeId: active?.id || tabs[0]?.id || '' };
  }
  return { ...node, first: normalizeNode(node.first, seen), second: normalizeNode(node.second, seen) };
}

function migrate(root: WorkspaceNode, maximizedSurfaceId?: string): WorkspaceModel {
  return { version: 2, root: normalizeNode(root, new Set()), maximizedSurfaceId };
}

export function deserializeWorkspace(raw: string | null | undefined, fallback: WorkspaceModel): WorkspaceModel {
  if (!raw) return fallback;
  try {
    const parsed = JSON.parse(raw) as { version?: number; root?: WorkspaceNode; maximizedSurfaceId?: string };
    if ((parsed.version === 1 || parsed.version === 2) && validNode(parsed.root)) return migrate(parsed.root, parsed.maximizedSurfaceId);
    return fallback;
  } catch { return fallback; }
}
