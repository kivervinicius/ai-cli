import { describe, expect, it } from 'vitest';
import { createWorkspace } from './model';
import { deserializeWorkspace, serializeWorkspace, workspaceStorageKey } from './state';

describe('workspace persistence', () => {
  it('round-trips valid models', () => {
    const model = createWorkspace({ id: 'overview', type: 'overview', title: 'Overview', closable: false });
    expect(deserializeWorkspace(serializeWorkspace(model), model)).toEqual(model);
  });

  it('falls back for invalid JSON', () => {
    const fallback = createWorkspace({ id: 'home', type: 'home', title: 'Home' });
    expect(deserializeWorkspace('{broken', fallback)).toEqual(fallback);
  });

  it('falls back for unsupported versions', () => {
    const fallback = createWorkspace({ id: 'home', type: 'home', title: 'Home' });
    expect(deserializeWorkspace(JSON.stringify({ version: 99, root: {} }), fallback)).toEqual(fallback);
  });

  it('scopes local layout keys by project', () => {
    expect(workspaceStorageKey('prj_1')).toBe('iapro:nexus:workspace:prj_1:v2');
  });

  it('migrates v1 layouts while preserving useful surfaces and assigning focus', () => {
    const fallback = createWorkspace({ id: 'fallback', type: 'home', title: 'Fallback' });
    const v1 = { version: 1, root: fallback.root, maximizedSurfaceId: undefined };
    const migrated = deserializeWorkspace(JSON.stringify(v1), fallback);
    expect(migrated.version).toBe(2);
    expect(migrated.focusedStackId).toBeTruthy();
    expect(migrated.root).toEqual(fallback.root);
  });
});
