export type WorkspaceDirection = 'horizontal' | 'vertical';

export interface WorkspaceSurface {
  id: string;
  /** Stable visual identity. `id` remains as a compatibility alias. */
  viewId?: string;
  /** Pre-v2 identifier retained so live runtime events can address migrated views. */
  legacyId?: string;
  /** Stable semantic identity used to focus an existing view instead of duplicating it. */
  logicalKey?: string;
  type: string;
  title: string;
  titleKey?: string;
  titleParams?: Record<string, string | number>;
  subtitle?: string;
  icon?: string;
  closable?: boolean;
  data?: Record<string, string>;
}

export interface WorkspaceStack {
  kind: 'stack';
  id: string;
  tabs: WorkspaceSurface[];
  activeId: string;
}

export interface WorkspaceSplit {
  kind: 'split';
  id: string;
  direction: WorkspaceDirection;
  ratio: number;
  first: WorkspaceNode;
  second: WorkspaceNode;
}

export type WorkspaceNode = WorkspaceStack | WorkspaceSplit;

export interface WorkspaceModel {
  version: 2;
  root: WorkspaceNode;
  focusedStackId: string;
  maximizedSurfaceId?: string;
  maximizedViewId?: string;
}

let idCounter = 0;
const nextId = (prefix: string) => `${prefix}_${Date.now().toString(36)}_${(++idCounter).toString(36)}`;

export function createStack(surface: WorkspaceSurface): WorkspaceStack {
  const normalized = normalizeSurface(surface);
  return { kind: 'stack', id: nextId('stack'), tabs: [normalized], activeId: normalized.viewId! };
}

export function createWorkspace(surface: WorkspaceSurface): WorkspaceModel {
  const stack = createStack(surface);
  return { version: 2, root: stack, focusedStackId: stack.id };
}

export function createEmptyStack(): WorkspaceStack {
  return { kind: 'stack', id: nextId('stack'), tabs: [], activeId: '' };
}

export function normalizeSurface(surface: WorkspaceSurface): WorkspaceSurface {
  const viewId = surface.viewId || surface.id;
  return { ...surface, id: viewId, viewId, legacyId: surface.legacyId || (surface.viewId && surface.viewId !== surface.id ? surface.id : undefined), logicalKey: surface.logicalKey || surface.id, closable: surface.closable ?? true };
}

export function surfaceViewId(surface: WorkspaceSurface): string { return surface.viewId || surface.id; }
export function surfaceLogicalKey(surface: WorkspaceSurface): string { return surface.logicalKey || surface.id; }

export function clampRatio(ratio: number): number {
  return Math.min(0.8, Math.max(0.2, Number.isFinite(ratio) ? ratio : 0.5));
}

export function isSurfaceMatch(surface: WorkspaceSurface, targetId: string): boolean {
  if (!targetId) return false;
  return surface.id === targetId || surface.viewId === targetId || surface.legacyId === targetId || surface.logicalKey === targetId;
}

export function findStackContaining(node: WorkspaceNode, surfaceId: string): WorkspaceStack | null {
  if (node.kind === 'stack') return node.tabs.some((tab) => isSurfaceMatch(tab, surfaceId)) ? node : null;
  return findStackContaining(node.first, surfaceId) ?? findStackContaining(node.second, surfaceId);
}

export function findStackById(node: WorkspaceNode, stackId: string): WorkspaceStack | null {
  if (node.kind === 'stack') return node.id === stackId ? node : null;
  return findStackById(node.first, stackId) ?? findStackById(node.second, stackId);
}

export function listStacks(node: WorkspaceNode): WorkspaceStack[] {
  if (node.kind === 'stack') return [node];
  return [...listStacks(node.first), ...listStacks(node.second)];
}

export function listSurfaces(node: WorkspaceNode): WorkspaceSurface[] {
  return listStacks(node).flatMap((stack) => stack.tabs);
}

function updateNode(node: WorkspaceNode, fn: (node: WorkspaceNode) => WorkspaceNode): WorkspaceNode {
  const transformed = node.kind === 'split'
    ? { ...node, first: updateNode(node.first, fn), second: updateNode(node.second, fn) }
    : node;
  return fn(transformed);
}

function replaceStack(node: WorkspaceNode, stackId: string, next: WorkspaceNode): WorkspaceNode {
  if (node.kind === 'stack') return node.id === stackId ? next : node;
  return { ...node, first: replaceStack(node.first, stackId, next), second: replaceStack(node.second, stackId, next) };
}

export function setActiveSurface(model: WorkspaceModel, surfaceId: string): WorkspaceModel {
  const root = updateNode(model.root, (node) => {
    if (node.kind !== 'stack') return node;
    const match = node.tabs.find((tab) => isSurfaceMatch(tab, surfaceId));
    if (!match) return node;
    return { ...node, activeId: match.id };
  });
  const stack = findStackContaining(root, surfaceId);
  return { ...model, root, focusedStackId: stack?.id ?? model.focusedStackId };
}

export function openSurface(model: WorkspaceModel, surface: WorkspaceSurface, targetStackId?: string): WorkspaceModel {
  const normalized = normalizeSurface(surface);
  const existing = listSurfaces(model.root).find((candidate) =>
    isSurfaceMatch(candidate, normalized.id) ||
    Boolean(normalized.logicalKey && surfaceLogicalKey(candidate) === surfaceLogicalKey(normalized))
  );
  if (existing) return setActiveSurface(model, existing.id);
  const stacks = listStacks(model.root);
  const target = (targetStackId && findStackById(model.root, targetStackId)) || findStackById(model.root, model.focusedStackId) || stacks[0];
  if (!target) return createWorkspace(normalized);
  const nextStack: WorkspaceStack = {
    ...target,
    tabs: [...target.tabs, normalized],
    activeId: normalized.id,
  };
  return { ...model, root: replaceStack(model.root, target.id, nextStack), focusedStackId: target.id };
}

function removeSurface(node: WorkspaceNode, surfaceId: string): { node: WorkspaceNode | null; removed?: WorkspaceSurface } {
  if (node.kind === 'stack') {
    const removedIndex = node.tabs.findIndex((tab) => isSurfaceMatch(tab, surfaceId));
    if (removedIndex === -1) return { node };
    const removed = node.tabs[removedIndex];
    if (removed.closable === false) return { node };
    const tabs = node.tabs.filter((_, idx) => idx !== removedIndex);
    if (tabs.length === 0) return { node: null, removed };
    const wasActive = isSurfaceMatch(removed, node.activeId);
    const activeId = wasActive ? (tabs[Math.max(0, removedIndex - 1)]?.id ?? tabs[0].id) : node.activeId;
    return { node: { ...node, tabs, activeId }, removed };
  }
  const first = removeSurface(node.first, surfaceId);
  if (first.removed) {
    if (!first.node) return { node: node.second, removed: first.removed };
    return { node: { ...node, first: first.node }, removed: first.removed };
  }
  const second = removeSurface(node.second, surfaceId);
  if (second.removed) {
    if (!second.node) return { node: node.first, removed: second.removed };
    return { node: { ...node, second: second.node }, removed: second.removed };
  }
  return { node };
}

export function closeSurface(model: WorkspaceModel, surfaceId: string): WorkspaceModel {
  const result = removeSurface(model.root, surfaceId);
  if (!result.removed || !result.node) return model;
  return {
    ...model,
    root: result.node,
    maximizedSurfaceId: model.maximizedSurfaceId === surfaceId ? undefined : model.maximizedSurfaceId,
    maximizedViewId: model.maximizedViewId === surfaceId ? undefined : model.maximizedViewId,
  };
}

export function splitWithSurface(
  model: WorkspaceModel,
  relativeSurfaceId: string,
  surface: WorkspaceSurface,
  direction: WorkspaceDirection,
  before = false,
): WorkspaceModel {
  const source = findStackContaining(model.root, relativeSurfaceId) ?? listStacks(model.root)[0];
  if (!source) return createWorkspace(surface);
  if (findStackContaining(model.root, surface.id)) return model;
  const newStack = createStack(surface);
  const split: WorkspaceSplit = {
    kind: 'split',
    id: nextId('split'),
    direction,
    ratio: 0.5,
    first: before ? newStack : source,
    second: before ? source : newStack,
  };
  return { ...model, root: replaceStack(model.root, source.id, split) };
}

export function splitEmpty(model: WorkspaceModel, relativeSurfaceId: string, direction: WorkspaceDirection, before = false): WorkspaceModel {
  const source = findStackContaining(model.root, relativeSurfaceId) ?? findStackById(model.root, model.focusedStackId);
  if (!source) return model;
  const empty = createEmptyStack();
  const split: WorkspaceSplit = { kind: 'split', id: nextId('split'), direction, ratio: 0.5, first: before ? empty : source, second: before ? source : empty };
  return { ...model, root: replaceStack(model.root, source.id, split), focusedStackId: empty.id };
}

export function moveSurface(model: WorkspaceModel, surfaceId: string, targetStackId: string): WorkspaceModel {
  const source = findStackContaining(model.root, surfaceId);
  const target = findStackById(model.root, targetStackId);
  const surface = source?.tabs.find((tab) => tab.id === surfaceId);
  if (!surface || !target || source?.id === target.id) return setActiveSurface(model, surfaceId);
  const removed = removeSurface(model.root, surfaceId);
  if (!removed.node) return model;
  const targetAfter = findStackById(removed.node, targetStackId);
  if (!targetAfter) return model;
  const nextTarget: WorkspaceStack = { ...targetAfter, tabs: [...targetAfter.tabs, surface], activeId: surfaceId };
  return { ...model, root: replaceStack(removed.node, targetStackId, nextTarget) };
}

export function setSplitRatio(model: WorkspaceModel, splitId: string, ratio: number): WorkspaceModel {
  const root = updateNode(model.root, (node) => node.kind === 'split' && node.id === splitId ? { ...node, ratio: clampRatio(ratio) } : node);
  return { ...model, root };
}

export function toggleMaximize(model: WorkspaceModel, surfaceId: string): WorkspaceModel {
  return { ...model, maximizedSurfaceId: model.maximizedSurfaceId === surfaceId ? undefined : surfaceId, maximizedViewId: model.maximizedViewId === surfaceId ? undefined : surfaceId };
}

export function updateSurface(
  model: WorkspaceModel,
  surfaceId: string,
  patch: Partial<WorkspaceSurface>,
): WorkspaceModel {
  const root = updateNode(model.root, (node) => {
    if (node.kind !== 'stack') return node;
    const tabIdx = node.tabs.findIndex((tab) => tab.id === surfaceId || tab.legacyId === surfaceId || tab.logicalKey === surfaceId);
    if (tabIdx === -1) return node;
    const nextTabs = [...node.tabs];
    nextTabs[tabIdx] = { ...nextTabs[tabIdx], ...patch };
    return { ...node, tabs: nextTabs };
  });
  return { ...model, root };
}
