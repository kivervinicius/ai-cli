import { describe, expect, it } from 'vitest';
import {
  createPresentationState,
  focusDesktopWindow,
  migratePresentationState,
  moveDesktopWindow,
  rearrangeSmart,
  resizeAdjacentDesktopWindows,
  resizeDesktopWindow,
  syncDesktopWindows,
  toggleDesktopMaximize,
  toggleDesktopMinimize,
} from './presentation';
import type { WorkspaceSurface } from './model';

const surface = (
  id: string,
  type: string = 'terminal',
  logicalKey = id
): WorkspaceSurface => ({ id, viewId: `view:${id}`, logicalKey, type, title: id });

describe('workspace presentation', () => {
  it('keys Desktop windows by stable view identity', () => {
    let state = createPresentationState('DESKTOP');
    state = syncDesktopWindows(state, [surface('a'), surface('b')]);
    expect(Object.keys(state.windows).sort()).toEqual(['view:a', 'view:b']);
    const again = syncDesktopWindows(state, [surface('a'), surface('b')]);
    expect(Object.keys(again.windows)).toHaveLength(2);
  });

  it('only syncs terminal and project-shell surfaces', () => {
    let state = createPresentationState('DESKTOP');
    state = syncDesktopWindows(state, [
      surface('a', 'terminal'),
      surface('overview', 'overview'),
      surface('shell', 'project-shell'),
      surface('work', 'work'),
    ]);
    expect(Object.keys(state.windows).sort()).toEqual(['view:a', 'view:shell']);
  });

  it('tiles new windows with Smart layout instead of cascade', () => {
    let state = createPresentationState('DESKTOP');
    state = { ...state, canvas: { x: 8, y: 8, width: 1000, height: 600 } };
    state = syncDesktopWindows(state, [surface('a'), surface('b')]);
    expect(state.windows['view:a'].x).toBe(8);
    expect(state.windows['view:b'].x).toBeGreaterThan(state.windows['view:a'].x);
    expect(state.tiled).toBe(true);
  });

  it('focuses without changing logical view identity', () => {
    let state = syncDesktopWindows(createPresentationState('DESKTOP'), [surface('a')]);
    const before = state.windows['view:a'].viewId;
    state = focusDesktopWindow(state, 'view:a');
    expect(state.windows['view:a'].viewId).toBe(before);
    expect(state.windows['view:a'].zIndex).toBeGreaterThan(0);
  });

  it('ignores free-float moves in tiling v1', () => {
    let state = syncDesktopWindows(createPresentationState('DESKTOP'), [surface('a')]);
    const before = { ...state.windows['view:a'] };
    state = moveDesktopWindow(state, 'view:a', 120, 80);
    expect(state.windows['view:a'].x).toBe(before.x);
    expect(state.windows['view:a'].y).toBe(before.y);
  });

  it('records custom resize as non-tiled geometry', () => {
    let state = syncDesktopWindows(createPresentationState('DESKTOP'), [surface('a')]);
    state = resizeDesktopWindow(state, 'view:a', 900, 640);
    expect(state.windows['view:a']).toMatchObject({ width: 900, height: 640 });
    expect(state.tiled).toBe(false);
  });

  it('persists adjacent splitter deltas', () => {
    let state = {
      ...createPresentationState('DESKTOP'),
      canvas: { x: 8, y: 8, width: 1000, height: 600 },
    };
    state = syncDesktopWindows(state, [surface('a'), surface('b')]);
    const beforeA = state.windows['view:a'].width;
    const beforeB = state.windows['view:b'].width;
    state = resizeAdjacentDesktopWindows(state, 'view:a', 'view:b', 'vertical', 50);
    expect(state.windows['view:a'].width).toBe(beforeA + 50);
    expect(state.windows['view:b'].width).toBe(beforeB - 50);
    expect(state.tiled).toBe(false);
  });

  it('minimizes and maximizes as presentation-only state', () => {
    let state = syncDesktopWindows(createPresentationState('DESKTOP'), [surface('a'), surface('b')]);
    state = toggleDesktopMinimize(state, 'view:a');
    expect(state.windows['view:a'].minimized).toBe(true);
    expect(state.windows['view:b'].minimized).toBe(false);
    state = toggleDesktopMaximize(state, 'view:b');
    expect(state.windows['view:b'].maximized).toBe(true);
  });

  it('rearranges visible windows with Smart', () => {
    let state = syncDesktopWindows(
      { ...createPresentationState('DESKTOP'), canvas: { x: 0, y: 0, width: 800, height: 600 } },
      [surface('a'), surface('b'), surface('c')]
    );
    state = rearrangeSmart(state);
    expect(Object.keys(state.windows)).toHaveLength(3);
    expect(state.tiled).toBe(true);
  });

  it('migrates v1 cascade state to tiled v2', () => {
    const migrated = migratePresentationState({
      version: 1,
      mode: 'DESKTOP',
      windows: {
        'view:a': { viewId: 'view:a', x: 32, y: 28, width: 720, height: 500, zIndex: 1, minimized: false, maximized: false },
      },
      nextZ: 2,
    });
    expect(migrated.version).toBe(2);
    expect(migrated.tiled).toBe(true);
    expect(migrated.canvas.width).toBeGreaterThan(0);
  });

  it('drops window chrome only after the logical surface is removed', () => {
    let state = syncDesktopWindows(createPresentationState('DESKTOP'), [surface('a'), surface('b')]);
    state = syncDesktopWindows(state, [surface('a')]);
    expect(state.windows['view:a']).toBeDefined();
    expect(state.windows['view:b']).toBeUndefined();
  });
});
