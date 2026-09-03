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

  it('collapses legacy and duplicate overview / resources tabs into a single canonical view', () => {
    const fallback = createWorkspace({
      id: 'project:prj_1:overview',
      logicalKey: 'project:prj_1:overview',
      type: 'overview',
      title: 'Overview',
      closable: false,
      data: { projectId: 'prj_1' },
    });
    const raw = JSON.stringify({
      version: 2,
      root: {
        kind: 'stack',
        id: 'stack-main',
        activeId: 'project-overview',
        tabs: [
          { id: 'project-overview', type: 'overview', title: 'Overview' },
          { id: 'project:prj_1:overview', type: 'overview', title: 'Visão Geral', logicalKey: 'project:prj_1:overview' },
          { id: 'resources', type: 'resources', title: 'Uso' },
          { id: 'project:prj_1:resources', type: 'resources', title: 'Resources', logicalKey: 'project:prj_1:resources' },
          { id: 'projects', type: 'projects', title: 'Desktops' },
          { id: 'project:prj_1:projects', type: 'projects', title: 'Workspace Desktops' },
        ],
      },
    });
    const migrated = deserializeWorkspace(raw, fallback, 'prj_1');
    if (migrated.root.kind !== 'stack') throw new Error('expected stack');
    // Must have exactly 1 overview, 1 resources, 1 projects
    expect(migrated.root.tabs).toHaveLength(3);
    const keys = migrated.root.tabs.map((t) => t.logicalKey);
    expect(keys).toEqual([
      'project:prj_1:overview',
      'project:prj_1:resources',
      'project:prj_1:projects',
    ]);
  });

  it('filters out project surfaces belonging to another project ID', () => {
    const fallback = createWorkspace({
      id: 'project:prj_target:overview',
      logicalKey: 'project:prj_target:overview',
      type: 'overview',
      title: 'Overview',
      closable: false,
      data: { projectId: 'prj_target' },
    });
    const raw = JSON.stringify({
      version: 2,
      root: {
        kind: 'stack',
        id: 'stack-main',
        activeId: 'project:prj_target:overview',
        tabs: [
          { id: 'project:prj_target:overview', type: 'overview', title: 'Target Overview', data: { projectId: 'prj_target' } },
          { id: 'project:prj_old:overview', type: 'overview', title: 'Old Overview', data: { projectId: 'prj_old' } },
          { id: 'project:prj_old:resources', type: 'resources', title: 'Old Resources', data: { projectId: 'prj_old' } },
          { id: 'agent:agt_1:terminal', type: 'terminal', title: 'Agent', data: { agentId: 'agt_1' } },
        ],
      },
    });
    const migrated = deserializeWorkspace(raw, fallback, 'prj_target');
    if (migrated.root.kind !== 'stack') throw new Error('expected stack');
    expect(migrated.root.tabs).toHaveLength(2);
    expect(migrated.root.tabs[0].logicalKey).toBe('project:prj_target:overview');
    expect(migrated.root.tabs[1].logicalKey).toBe('agent:agt_1:terminal');
  });

  it('scopes local layout keys by project', () => {
    expect(workspaceStorageKey('prj_1')).toBe('iapro:nexus:workspace:prj_1:v2');
  });
});
