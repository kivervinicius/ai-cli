import { describe, expect, it } from 'vitest';
import {
  commitMosaicMove,
  createPresentationState,
  focusDesktopWindow,
  migratePresentationState,
  moveDesktopWindow,
  rearrangeSmart,
  resizeAdjacentDesktopWindows,
  resizeDesktopWindow,
  setPresentationMode,
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

  it('cascades new windows instead of Smart tiling', () => {
    let state = createPresentationState('DESKTOP');
    state = { ...state, canvas: { x: 8, y: 8, width: 1000, height: 600 } };
    state = syncDesktopWindows(state, [surface('a'), surface('b')]);
    expect(state.windows['view:a'].x).toBe(8);
    expect(state.windows['view:a'].y).toBe(8);
    expect(state.windows['view:b'].x).toBeGreaterThan(state.windows['view:a'].x);
    expect(state.windows['view:b'].y).toBeGreaterThan(state.windows['view:a'].y);
    expect(state.tiled).toBe(false);
  });

  it('preserves existing geometry when syncing additional PTYs', () => {
    let state = syncDesktopWindows(
      { ...createPresentationState('DESKTOP'), canvas: { x: 8, y: 8, width: 1000, height: 600 } },
      [surface('a')]
    );
    state = moveDesktopWindow(state, 'view:a', 200, 140);
    const before = { ...state.windows['view:a'] };
    state = syncDesktopWindows(state, [surface('a'), surface('b')]);
    expect(state.windows['view:a'].x).toBe(before.x);
    expect(state.windows['view:a'].y).toBe(before.y);
    expect(state.windows['view:a'].width).toBe(before.width);
    expect(state.windows['view:b']).toBeDefined();
    expect(state.windows['view:b'].x).not.toBe(before.x);
  });

  it('focuses without changing logical view identity', () => {
    let state = syncDesktopWindows(createPresentationState('DESKTOP'), [surface('a')]);
    const before = state.windows['view:a'].viewId;
    state = focusDesktopWindow(state, 'view:a');
    expect(state.windows['view:a'].viewId).toBe(before);
    expect(state.windows['view:a'].zIndex).toBeGreaterThan(0);
    expect(state.activePtyViewId).toBe('view:a');
  });

  it('promotes newly synced PTYs to activePtyViewId', () => {
    let state = syncDesktopWindows(createPresentationState('DESKTOP'), [surface('a')]);
    expect(state.activePtyViewId).toBe('view:a');
    state = syncDesktopWindows(state, [surface('a'), surface('b')]);
    expect(state.activePtyViewId).toBe('view:b');
  });

  it('persists free-float moves', () => {
    let state = syncDesktopWindows(createPresentationState('DESKTOP'), [surface('a')]);
    state = moveDesktopWindow(state, 'view:a', 120, 80);
    expect(state.windows['view:a'].x).toBe(120);
    expect(state.windows['view:a'].y).toBe(80);
    expect(state.tiled).toBe(false);
  });

  it('records custom resize as free-float geometry', () => {
    let state = syncDesktopWindows(createPresentationState('DESKTOP'), [surface('a')]);
    state = resizeDesktopWindow(state, 'view:a', 900, 640);
    expect(state.windows['view:a']).toMatchObject({ width: 900, height: 640 });
    expect(state.tiled).toBe(false);
  });

  it('minimizes without rearranging siblings', () => {
    let state = syncDesktopWindows(
      { ...createPresentationState('DESKTOP'), canvas: { x: 8, y: 8, width: 1000, height: 600 } },
      [surface('a'), surface('b')]
    );
    state = moveDesktopWindow(state, 'view:b', 300, 200);
    const beforeB = { ...state.windows['view:b'] };
    state = toggleDesktopMinimize(state, 'view:a');
    expect(state.windows['view:a'].minimized).toBe(true);
    expect(state.windows['view:b'].x).toBe(beforeB.x);
    expect(state.windows['view:b'].y).toBe(beforeB.y);
    state = toggleDesktopMaximize(state, 'view:b');
    expect(state.windows['view:b'].maximized).toBe(true);
  });

  it('rearranges visible windows with cascade helper', () => {
    let state = syncDesktopWindows(
      { ...createPresentationState('DESKTOP'), canvas: { x: 0, y: 0, width: 800, height: 600 } },
      [surface('a'), surface('b'), surface('c')]
    );
    state = rearrangeSmart(state);
    expect(Object.keys(state.windows)).toHaveLength(3);
    expect(state.tiled).toBe(false);
  });

  it('preserves DESKTOP mode on migrate and disables tiled mosaic', () => {
    const migrated = migratePresentationState({
      version: 1,
      mode: 'DESKTOP',
      windows: {
        'view:a': { viewId: 'view:a', x: 32, y: 28, width: 720, height: 500, zIndex: 1, minimized: false, maximized: false },
      },
      nextZ: 2,
    });
    expect(migrated.version).toBe(2);
    expect(migrated.mode).toBe('DESKTOP');
    expect(migrated.tiled).toBe(false);
    expect(migrated.canvas.width).toBeGreaterThan(0);
  });

  it('drops window chrome only after the logical surface is removed', () => {
    let state = syncDesktopWindows(createPresentationState('DESKTOP'), [surface('a'), surface('b')]);
    state = syncDesktopWindows(state, [surface('a')]);
    expect(state.windows['view:a']).toBeDefined();
    expect(state.windows['view:b']).toBeUndefined();
  });

  it('tiles visible windows in MOSAIC and retile on minimize', () => {
    let state = syncDesktopWindows(
      { ...createPresentationState('MOSAIC'), canvas: { x: 8, y: 8, width: 1000, height: 600 } },
      [surface('a'), surface('b')]
    );
    state = setPresentationMode(state, 'MOSAIC');
    expect(state.mode).toBe('MOSAIC');
    expect(state.tiled).toBe(true);
    expect(state.windows['view:a'].width + state.windows['view:b'].width).toBeGreaterThan(500);
    const beforeA = { ...state.windows['view:a'] };
    state = toggleDesktopMinimize(state, 'view:b');
    expect(state.windows['view:b'].minimized).toBe(true);
    expect(state.windows['view:a'].width).toBeGreaterThan(beforeA.width);
  });

  it('resizes adjacent mosaic tiles and supports free mosaic move/swap', () => {
    let state = setPresentationMode(
      syncDesktopWindows(
        { ...createPresentationState('DESKTOP'), canvas: { x: 0, y: 0, width: 800, height: 400 } },
        [surface('a'), surface('b')]
      ),
      'MOSAIC'
    );
    const left = state.windows['view:a'];
    const right = state.windows['view:b'];
    state = resizeAdjacentDesktopWindows(state, 'view:a', 'view:b', 'vertical', 40);
    expect(state.windows['view:a'].width).toBe(left.width + 40);
    expect(state.windows['view:b'].width).toBe(right.width - 40);

    const origin = {
      x: state.windows['view:a'].x,
      y: state.windows['view:a'].y,
      width: state.windows['view:a'].width,
      height: state.windows['view:a'].height,
    };
    const targetGeom = {
      x: state.windows['view:b'].x,
      y: state.windows['view:b'].y,
      width: state.windows['view:b'].width,
      height: state.windows['view:b'].height,
    };
    state = moveDesktopWindow(state, 'view:a', targetGeom.x + 10, targetGeom.y + 10);
    state = commitMosaicMove(state, 'view:a', origin);
    expect(state.windows['view:a'].x).toBe(targetGeom.x);
    expect(state.windows['view:b'].x).toBe(origin.x);
  });

  it('persists mosaic chrome across migrate and pack', () => {
    const migrated = migratePresentationState({
      version: 2,
      mode: 'MOSAIC',
      windows: {
        'view:a': {
          viewId: 'view:a',
          x: 0,
          y: 0,
          width: 400,
          height: 400,
          zIndex: 1,
          minimized: false,
          maximized: false,
          customTitle: 'Ops',
          accent: '#38bdf8',
          icon: '⚡',
        },
      },
      nextZ: 2,
      canvas: { x: 0, y: 0, width: 800, height: 400 },
      tiled: true,
    });
    expect(migrated.windows['view:a'].customTitle).toBe('Ops');
    expect(migrated.windows['view:a'].accent).toBe('#38bdf8');
    expect(migrated.windows['view:a'].icon).toBe('⚡');
  });

  it('migrates MOSAIC mode', () => {
    const migrated = migratePresentationState({ version: 2, mode: 'MOSAIC', windows: {}, nextZ: 1 });
    expect(migrated.mode).toBe('MOSAIC');
  });
});
