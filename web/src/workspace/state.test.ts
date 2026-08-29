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
    expect(workspaceStorageKey('prj_1')).toBe('iapro:nexus:workspace:prj_1:v1');
  });
});
