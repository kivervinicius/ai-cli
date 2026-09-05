import { describe, expect, it, beforeEach } from 'vitest';
import { WorkspaceLayoutService, WORKSPACE_LAYOUT_VERSION } from './WorkspaceLayoutService';
import { createWorkspace } from '../workspace/model';

// In Node environment vitest tests, provide mock localStorage on global/window
const store = new Map<string, string>();
const mockLocalStorage = {
  getItem: (key: string) => store.get(key) ?? null,
  setItem: (key: string, val: string) => { store.set(key, String(val)); },
  removeItem: (key: string) => { store.delete(key); },
  clear: () => { store.clear(); },
};

(globalThis as any).window = globalThis;
(globalThis as any).localStorage = mockLocalStorage;

describe('WorkspaceLayoutService', () => {
  const projectId = 'test-proj';
  let service: WorkspaceLayoutService;
  const overviewId = `project:${projectId}:overview`;
  const fallback = createWorkspace({
    id: overviewId,
    logicalKey: overviewId,
    viewId: `view:${overviewId}`,
    type: 'overview',
    title: 'Overview',
    data: { projectId },
  });

  beforeEach(() => {
    mockLocalStorage.clear();
    service = new WorkspaceLayoutService(projectId);
  });

  it('loads fallback when storage is empty', () => {
    const layout = service.load(fallback);
    expect(layout.version).toBe(WORKSPACE_LAYOUT_VERSION);
    expect(layout.workspaceId).toBe(projectId);
    expect(layout.model.root.kind).toBe('stack');
  });

  it('saves and reloads atomically with versioning', () => {
    const initial = service.load(fallback);
    service.save(initial);

    const reloaded = service.load(fallback);
    expect(reloaded.version).toBe(WORKSPACE_LAYOUT_VERSION);
    expect(reloaded.activeSurfaceId).toBe(overviewId);
    expect(reloaded.openSurfaceIds).toContain(`view:${overviewId}`);
  });

  it('migrates from legacy v1/v2 storage keys smoothly', () => {
    mockLocalStorage.setItem(service.legacyModelKey, JSON.stringify(fallback));
    mockLocalStorage.setItem(
      service.legacyPresentationKey,
      JSON.stringify({ mode: 'MOSAIC', windows: {}, nextZ: 2 })
    );

    const migrated = service.load(fallback);
    expect(migrated.version).toBe(WORKSPACE_LAYOUT_VERSION);
    expect(migrated.presentation.mode).toBe('MOSAIC');
  });

  it('exports and imports valid workbench configurations', () => {
    const initial = service.load(fallback);
    const exported = service.export(initial);
    expect(typeof exported).toBe('string');

    const imported = service.import(exported, fallback);
    expect(imported.version).toBe(WORKSPACE_LAYOUT_VERSION);
    expect(imported.workspaceId).toBe(projectId);
  });

  it('resets layout to fallback defaults', () => {
    const initial = service.load(fallback);
    service.save(initial);

    const reset = service.reset(fallback);
    expect(reset.version).toBe(WORKSPACE_LAYOUT_VERSION);
    expect(reset.model.root.kind).toBe('stack');
  });
});
