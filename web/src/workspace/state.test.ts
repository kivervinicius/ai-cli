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

  it('migrates v1 layouts into v2 and deduplicates the same logical view', () => {
    const fallback = createWorkspace({ id: 'fallback', type: 'overview', title: 'Fallback' });
    const raw = JSON.stringify({
      version: 1,
      root: {
        kind: 'stack',
        id: 'stack-old',
        activeId: 'agent-copy',
        tabs: [
          { id: 'agent-old', type: 'terminal', title: 'Agent', logicalKey: 'agent:1:terminal' },
          { id: 'agent-copy', type: 'terminal', title: 'Agent copy', logicalKey: 'agent:1:terminal' },
        ],
      },
    });
    const migrated = deserializeWorkspace(raw, fallback);
    expect(migrated.version).toBe(2);
    if (migrated.root.kind !== 'stack') throw new Error('expected stack');
    expect(migrated.root.tabs).toHaveLength(1);
    expect(migrated.root.tabs[0].logicalKey).toBe('agent:1:terminal');
  });

  it('scopes local layout keys by project', () => {
    expect(workspaceStorageKey('prj_1')).toBe('iapro:nexus:workspace:prj_1:v2');
  });
});
