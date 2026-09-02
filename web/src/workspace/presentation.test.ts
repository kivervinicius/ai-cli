import { describe, expect, it } from 'vitest';
import {
  createPresentationState,
  focusDesktopWindow,
  moveDesktopWindow,
  resizeDesktopWindow,
  syncDesktopWindows,
  toggleDesktopMaximize,
  toggleDesktopMinimize,
} from './presentation';
import type { WorkspaceSurface } from './model';

const surface = (id: string, logicalKey = id): WorkspaceSurface => ({ id, viewId: `view:${id}`, logicalKey, type: 'test', title: id });

describe('workspace presentation', () => {
  it('keys Desktop windows by stable view identity', () => {
    let state = createPresentationState('DESKTOP');
    state = syncDesktopWindows(state, [surface('a'), surface('b')]);
    expect(Object.keys(state.windows).sort()).toEqual(['view:a', 'view:b']);
    const again = syncDesktopWindows(state, [surface('a'), surface('b')]);
    expect(Object.keys(again.windows)).toHaveLength(2);
  });

  it('moves, resizes and focuses without changing logical view identity', () => {
    let state = syncDesktopWindows(createPresentationState('DESKTOP'), [surface('a')]);
    const before = state.windows['view:a'].viewId;
    state = moveDesktopWindow(state, 'view:a', 120, 80);
    state = resizeDesktopWindow(state, 'view:a', 900, 640);
    state = focusDesktopWindow(state, 'view:a');
    expect(state.windows['view:a']).toMatchObject({ viewId: before, x: 120, y: 80, width: 900, height: 640 });
  });

  it('minimizes and maximizes as presentation-only state', () => {
    let state = syncDesktopWindows(createPresentationState('DESKTOP'), [surface('a')]);
    state = toggleDesktopMinimize(state, 'view:a');
    expect(state.windows['view:a'].minimized).toBe(true);
    state = toggleDesktopMaximize(state, 'view:a');
    expect(state.windows['view:a'].minimized).toBe(false);
    expect(state.windows['view:a'].maximized).toBe(true);
  });

  it('drops window chrome only after the logical surface is removed', () => {
    let state = syncDesktopWindows(createPresentationState('DESKTOP'), [surface('a'), surface('b')]);
    state = syncDesktopWindows(state, [surface('a')]);
    expect(state.windows['view:a']).toBeDefined();
    expect(state.windows['view:b']).toBeUndefined();
  });
});
