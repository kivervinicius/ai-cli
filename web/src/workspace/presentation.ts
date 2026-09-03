import { surfaceViewId, type WorkspaceSurface } from './model';
import { applySharedEdgeDelta, arrangeSmart, scaleTilesToCanvas, type ArrangeBounds } from './arrange';

export type WorkspacePresentationMode = 'TABS' | 'DESKTOP';

export interface DesktopWindowState {
  viewId: string;
  x: number;
  y: number;
  width: number;
  height: number;
  zIndex: number;
  minimized: boolean;
  maximized: boolean;
}

export interface WorkspacePresentationState {
  version: 2;
  mode: WorkspacePresentationMode;
  windows: Record<string, DesktopWindowState>;
  nextZ: number;
  canvas: ArrangeBounds;
  /** When true, next sync should re-run Smart instead of preserving free geometry. */
  tiled: boolean;
}

const DEFAULT_CANVAS: ArrangeBounds = { x: 8, y: 8, width: 1200, height: 720 };
const FLOATING_TYPES = new Set(['terminal', 'project-shell']);

export function isDesktopFloatingSurface(surface: WorkspaceSurface): boolean {
  return FLOATING_TYPES.has(surface.type);
}

export function createPresentationState(mode: WorkspacePresentationMode = 'TABS'): WorkspacePresentationState {
  return { version: 2, mode, windows: {}, nextZ: 1, canvas: { ...DEFAULT_CANVAS }, tiled: true };
}

export function migratePresentationState(raw: unknown): WorkspacePresentationState {
  const fallback = createPresentationState();
  if (!raw || typeof raw !== 'object') return fallback;
  const parsed = raw as Partial<WorkspacePresentationState> & { version?: number };
  const mode = parsed.mode === 'DESKTOP' || parsed.mode === 'TABS' ? parsed.mode : 'TABS';
  const windows =
    parsed.windows && typeof parsed.windows === 'object'
      ? (parsed.windows as Record<string, DesktopWindowState>)
      : {};
  const nextZ = typeof parsed.nextZ === 'number' && parsed.nextZ > 0 ? parsed.nextZ : 1;
  const canvas =
    parsed.canvas &&
    typeof parsed.canvas.width === 'number' &&
    typeof parsed.canvas.height === 'number'
      ? {
          x: Number(parsed.canvas.x) || 8,
          y: Number(parsed.canvas.y) || 8,
          width: Math.max(320, Number(parsed.canvas.width) || DEFAULT_CANVAS.width),
          height: Math.max(240, Number(parsed.canvas.height) || DEFAULT_CANVAS.height),
        }
      : { ...DEFAULT_CANVAS };

  // v1 cascade → re-tile on first Desktop paint.
  const wasV1 = Number(parsed.version) === 1;
  return {
    version: 2,
    mode,
    windows,
    nextZ,
    canvas,
    tiled: wasV1 ? true : parsed.tiled !== false,
  };
}

function applyTiles(
  state: WorkspacePresentationState,
  viewIds: string[],
  preserved: Record<string, DesktopWindowState>
): WorkspacePresentationState {
  const tiles = arrangeSmart(state.canvas, viewIds);
  const windows: Record<string, DesktopWindowState> = {};
  let nextZ = Math.max(1, state.nextZ);
  tiles.forEach((tile, index) => {
    const previous = preserved[tile.viewId];
    const zIndex = previous?.zIndex ?? nextZ++;
    windows[tile.viewId] = {
      viewId: tile.viewId,
      x: tile.x,
      y: tile.y,
      width: tile.width,
      height: tile.height,
      zIndex,
      minimized: previous?.minimized ?? false,
      maximized: false,
    };
    nextZ = Math.max(nextZ, zIndex + 1);
  });
  // Keep minimized-only windows that were not in the tile pack order if they somehow appear.
  Object.values(preserved).forEach((win) => {
    if (!windows[win.viewId] && win.minimized) {
      windows[win.viewId] = win;
    }
  });
  return { ...state, windows, nextZ, tiled: true };
}

export function rearrangeSmart(
  state: WorkspacePresentationState,
  viewIds?: string[]
): WorkspacePresentationState {
  const ids =
    viewIds ??
    Object.values(state.windows)
      .filter((win) => !win.minimized)
      .sort((a, b) => a.zIndex - b.zIndex)
      .map((win) => win.viewId);
  const visible = ids.filter((id) => state.windows[id] && !state.windows[id].minimized);
  const minimized = Object.values(state.windows).filter((win) => win.minimized);
  const next = applyTiles(state, visible.length ? visible : ids, state.windows);
  minimized.forEach((win) => {
    next.windows[win.viewId] = { ...win, maximized: false };
  });
  return next;
}

export function setPresentationCanvas(
  state: WorkspacePresentationState,
  canvas: ArrangeBounds
): WorkspacePresentationState {
  const nextCanvas: ArrangeBounds = {
    x: Math.max(0, Math.round(canvas.x)),
    y: Math.max(0, Math.round(canvas.y)),
    width: Math.max(320, Math.round(canvas.width)),
    height: Math.max(240, Math.round(canvas.height)),
  };
  if (
    state.canvas.width === nextCanvas.width &&
    state.canvas.height === nextCanvas.height &&
    state.canvas.x === nextCanvas.x &&
    state.canvas.y === nextCanvas.y
  ) {
    return state;
  }

  const visible = Object.values(state.windows).filter((win) => !win.minimized && !win.maximized);
  if (visible.length === 0) {
    return { ...state, canvas: nextCanvas };
  }

  if (state.tiled) {
    const ids = visible.sort((a, b) => a.zIndex - b.zIndex).map((win) => win.viewId);
    return applyTiles({ ...state, canvas: nextCanvas }, ids, state.windows);
  }

  const scaled = scaleTilesToCanvas(visible, state.canvas, nextCanvas);
  const windows = { ...state.windows };
  scaled.forEach((tile) => {
    const current = windows[tile.viewId];
    if (!current) return;
    windows[tile.viewId] = { ...current, x: tile.x, y: tile.y, width: tile.width, height: tile.height };
  });
  return { ...state, canvas: nextCanvas, windows };
}

export function syncDesktopWindows(
  state: WorkspacePresentationState,
  surfaces: WorkspaceSurface[]
): WorkspacePresentationState {
  const floating = surfaces.filter(isDesktopFloatingSurface);
  const nextIds = floating.map(surfaceViewId);
  const preserved: Record<string, DesktopWindowState> = {};
  let nextZ = Math.max(1, state.nextZ);

  nextIds.forEach((viewId) => {
    const current = state.windows[viewId];
    if (current) {
      preserved[viewId] = current;
      nextZ = Math.max(nextZ, current.zIndex + 1);
    }
  });

  const added = nextIds.filter((id) => !state.windows[id]);
  const removed = Object.keys(state.windows).filter((id) => !nextIds.includes(id));
  if (added.length === 0 && removed.length === 0) {
    // Still drop non-floating leftovers if any.
    if (Object.keys(state.windows).every((id) => nextIds.includes(id))) return state;
  }

  const base: WorkspacePresentationState = { ...state, nextZ, tiled: true };
  const visibleIds = nextIds.filter((id) => !preserved[id]?.minimized);
  const tiled = applyTiles(base, visibleIds, preserved);
  // Re-attach minimized windows that still exist.
  nextIds.forEach((id) => {
    if (preserved[id]?.minimized) {
      tiled.windows[id] = { ...preserved[id], maximized: false };
    }
  });
  return tiled;
}

function patchWindow(
  state: WorkspacePresentationState,
  viewId: string,
  patch: Partial<DesktopWindowState>
): WorkspacePresentationState {
  const current = state.windows[viewId];
  if (!current) return state;
  return { ...state, windows: { ...state.windows, [viewId]: { ...current, ...patch } } };
}

export function focusDesktopWindow(state: WorkspacePresentationState, viewId: string): WorkspacePresentationState {
  if (!state.windows[viewId]) return state;
  const zIndex = state.nextZ;
  return { ...patchWindow(state, viewId, { zIndex, minimized: false }), nextZ: zIndex + 1 };
}

export function moveDesktopWindow(
  state: WorkspacePresentationState,
  viewId: string,
  x: number,
  y: number
): WorkspacePresentationState {
  // Desktop v1 is tiling-only; move is a no-op to avoid free-float regression.
  void x;
  void y;
  return state.windows[viewId] ? { ...state, tiled: true } : state;
}

export function resizeDesktopWindow(
  state: WorkspacePresentationState,
  viewId: string,
  width: number,
  height: number
): WorkspacePresentationState {
  const current = state.windows[viewId];
  if (!current || current.maximized || current.minimized) return state;
  return {
    ...patchWindow(state, viewId, {
      width: Math.max(280, Math.round(width)),
      height: Math.max(180, Math.round(height)),
    }),
    tiled: false,
  };
}

/** Persist a shared-edge drag between two tiled windows (ratios via absolute geometry). */
export function resizeAdjacentDesktopWindows(
  state: WorkspacePresentationState,
  firstId: string,
  secondId: string,
  orientation: 'vertical' | 'horizontal',
  delta: number
): WorkspacePresentationState {
  const first = state.windows[firstId];
  const second = state.windows[secondId];
  if (!first || !second || first.maximized || second.maximized || first.minimized || second.minimized) {
    return state;
  }
  const patched = applySharedEdgeDelta(first, second, orientation, delta);
  if (!patched) return state;
  return {
    ...state,
    tiled: false,
    windows: {
      ...state.windows,
      [firstId]: { ...first, ...patched.first },
      [secondId]: { ...second, ...patched.second },
    },
  };
}

export function toggleDesktopMinimize(state: WorkspacePresentationState, viewId: string): WorkspacePresentationState {
  const current = state.windows[viewId];
  if (!current) return state;
  const minimized = !current.minimized;
  let next = patchWindow(state, viewId, { minimized, maximized: false });
  if (minimized || !minimized) {
    const visible = Object.values(next.windows)
      .filter((win) => !win.minimized)
      .sort((a, b) => a.zIndex - b.zIndex)
      .map((win) => win.viewId);
    next = rearrangeSmart({ ...next, tiled: true }, visible);
  }
  return next;
}

export function toggleDesktopMaximize(state: WorkspacePresentationState, viewId: string): WorkspacePresentationState {
  const current = state.windows[viewId];
  if (!current) return state;
  return patchWindow(state, viewId, { maximized: !current.maximized, minimized: false });
}
