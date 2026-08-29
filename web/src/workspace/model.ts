export type WorkspaceDirection = 'horizontal' | 'vertical';

export interface WorkspaceSurface {
  id: string;
  type: string;
  title: string;
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
  version: 1;
  root: WorkspaceNode;
  maximizedSurfaceId?: string;
}

let idCounter = 0;
const nextId = (prefix: string) => `${prefix}_${Date.now().toString(36)}_${(++idCounter).toString(36)}`;

export function createStack(surface: WorkspaceSurface): WorkspaceStack {
  return { kind: 'stack', id: nextId('stack'), tabs: [{ ...surface, closable: surface.closable ?? true }], activeId: surface.id };
}

export function createWorkspace(surface: WorkspaceSurface): WorkspaceModel {
  return { version: 1, root: createStack(surface) };
}

export function clampRatio(ratio: number): number {
  return Math.min(0.8, Math.max(0.2, Number.isFinite(ratio) ? ratio : 0.5));
}

export function findStackContaining(node: WorkspaceNode, surfaceId: string): WorkspaceStack | null {
  if (node.kind === 'stack') return node.tabs.some((tab) => tab.id === surfaceId) ? node : null;
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
    if (node.kind !== 'stack' || !node.tabs.some((tab) => tab.id === surfaceId)) return node;
    return { ...node, activeId: surfaceId };
  });
  return { ...model, root };
}

export function openSurface(model: WorkspaceModel, surface: WorkspaceSurface, targetStackId?: string): WorkspaceModel {
  if (findStackContaining(model.root, surface.id)) return setActiveSurface(model, surface.id);
  const stacks = listStacks(model.root);
  const target = (targetStackId && findStackById(model.root, targetStackId)) || stacks[0];
  if (!target) return createWorkspace(surface);
  const nextStack: WorkspaceStack = {
    ...target,
    tabs: [...target.tabs, { ...surface, closable: surface.closable ?? true }],
    activeId: surface.id,
  };
  return { ...model, root: replaceStack(model.root, target.id, nextStack) };
}

function removeSurface(node: WorkspaceNode, surfaceId: string): { node: WorkspaceNode | null; removed?: WorkspaceSurface } {
  if (node.kind === 'stack') {
    const removed = node.tabs.find((tab) => tab.id === surfaceId);
    if (!removed || removed.closable === false) return { node };
    const tabs = node.tabs.filter((tab) => tab.id !== surfaceId);
    if (tabs.length === 0) return { node: null, removed };
    const activeId = node.activeId === surfaceId ? tabs[Math.max(0, node.tabs.findIndex((tab) => tab.id === surfaceId) - 1)]?.id ?? tabs[0].id : node.activeId;
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
  return { ...model, maximizedSurfaceId: model.maximizedSurfaceId === surfaceId ? undefined : surfaceId };
}
