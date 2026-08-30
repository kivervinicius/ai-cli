import { describe, expect, it } from 'vitest';
import {
  clampRatio,
  closeSurface,
  createWorkspace,
  findStackContaining,
  moveSurface,
  openSurface,
  setActiveSurface,
  splitWithSurface,
  updateSurface,
  type WorkspaceSurface,
} from './model';

const surface = (id: string): WorkspaceSurface => ({ id, type: 'test', title: id });

describe('workspace model', () => {
  it('opens a surface once and focuses it when reopened', () => {
    let ws = createWorkspace(surface('home'));
    ws = openSurface(ws, surface('terminal:a'));
    ws = openSurface(ws, surface('terminal:a'));
    const stack = findStackContaining(ws.root, 'terminal:a');
    expect(stack?.tabs.map((tab) => tab.id)).toEqual(['home', 'terminal:a']);
    expect(stack?.activeId).toBe('terminal:a');
  });

  it('deduplicates by logical key while allowing separate visual views', () => {
    let ws = createWorkspace({ ...surface('session-view-1'), logicalKey: 'session:1' });
    ws = openSurface(ws, { ...surface('session-view-2'), logicalKey: 'session:1' });
    expect(ws.root.kind).toBe('stack');
    if (ws.root.kind !== 'stack') return;
    expect(ws.root.tabs).toHaveLength(1);
    expect(ws.root.activeId).toBe('session-view-1');
  });

  it('accepts legacy runtime event identifiers for v2 views', () => {
    const ws = createWorkspace({ id: 'agent:a:terminal', viewId: 'view:agent:a:terminal', logicalKey: 'session:a', type: 'terminal', title: 'A' });
    const updated = updateSurface(ws, 'agent:a:terminal', { title: 'Updated' });
    expect(updated.root.kind === 'stack' && updated.root.tabs[0].title).toBe('Updated');
  });

  it('opens new empty focused pane for a split', async () => {
    const { splitEmpty, listStacks } = await import('./model');
    const ws = splitEmpty(createWorkspace(surface('home')), 'home', 'horizontal');
    expect(listStacks(ws.root)).toHaveLength(2);
    expect(ws.focusedStackId).toBe(listStacks(ws.root)[1].id);
    expect(listStacks(ws.root)[1].tabs).toHaveLength(0);
  });

  it('splits an existing stack with a new surface', () => {
    const ws = splitWithSurface(createWorkspace(surface('home')), 'home', surface('terminal:a'), 'horizontal');
    expect(ws.root.kind).toBe('split');
    if (ws.root.kind !== 'split') return;
    expect(ws.root.direction).toBe('horizontal');
    expect(findStackContaining(ws.root, 'terminal:a')?.activeId).toBe('terminal:a');
  });

  it('moves a surface between stacks without duplication', () => {
    let ws = splitWithSurface(createWorkspace(surface('home')), 'home', surface('terminal:a'), 'horizontal');
    if (ws.root.kind !== 'split' || ws.root.second.kind !== 'stack') throw new Error('unexpected model');
    ws = openSurface(ws, surface('plan'), ws.root.second.id);
    const targetStack = ws.root.kind === 'split' && ws.root.first.kind === 'stack' ? ws.root.first.id : '';
    ws = moveSurface(ws, 'plan', targetStack);
    expect(findStackContaining(ws.root, 'plan')?.id).toBe(targetStack);
    const all = JSON.stringify(ws.root).match(/"id":"plan"/g) ?? [];
    expect(all).toHaveLength(1);
  });

  it('closes a tab and collapses an empty split', () => {
    let ws = splitWithSurface(createWorkspace(surface('home')), 'home', surface('terminal:a'), 'vertical');
    ws = closeSurface(ws, 'terminal:a');
    expect(ws.root.kind).toBe('stack');
    expect(findStackContaining(ws.root, 'home')?.activeId).toBe('home');
  });

  it('keeps non-closable surfaces', () => {
    const ws = closeSurface(createWorkspace({ ...surface('home'), closable: false }), 'home');
    expect(findStackContaining(ws.root, 'home')).not.toBeNull();
  });

  it('activates a surface without changing the tree', () => {
    let ws = openSurface(createWorkspace(surface('home')), surface('plan'));
    ws = setActiveSurface(ws, 'home');
    expect(findStackContaining(ws.root, 'home')?.activeId).toBe('home');
  });

  it('clamps split ratios to usable bounds', () => {
    expect(clampRatio(0.01)).toBe(0.2);
    expect(clampRatio(0.5)).toBe(0.5);
    expect(clampRatio(0.99)).toBe(0.8);
  });
});
