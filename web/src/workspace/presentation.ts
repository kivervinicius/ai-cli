import { surfaceViewId, type WorkspaceSurface } from './model';
import {
  applySharedEdgeDelta,
  arrangeSmart,
  scaleTilesToCanvas,
  type ArrangeBounds,
} from './arrange';

export type WorkspacePresentationMode = 'TABS' | 'DESKTOP' | 'MOSAIC';

export interface DesktopWindowState {
  viewId: string;
  x: number;
  y: number;
  width: number;
  height: number;
  zIndex: number;
  minimized: boolean;
  maximized: boolean;
  /** User-chosen identity label; never overwritten by PTY OSC titles. */
  customTitle?: string;
  /** Accent color token (css color). */
  accent?: string;
  /** Fixed icon key from the mosaic chrome palette. */
  icon?: string;
}

export type MosaicResizeEdge = 'n' | 's' | 'e' | 'w' | 'ne' | 'nw' | 'se' | 'sw';

export interface WorkspacePresentationState {
  version: 2;
  mode: WorkspacePresentationMode;
  windows: Record<string, DesktopWindowState>;
  nextZ: number;
  canvas: ArrangeBounds;
  /**
   * True while MOSAIC keeps Smart-tiled geometry. Free-float DESKTOP keeps this false.
   */
  tiled: boolean;
  /** viewId of the PTY the user is viewing (inner tabs or desktop focus). */
  activePtyViewId: string;
}

const DEFAULT_CANVAS: ArrangeBounds = { x: 8, y: 8, width: 1200, height: 720 };
const DEFAULT_WINDOW_W = 640;
const DEFAULT_WINDOW_H = 420;
const CASCADE_OFFSET = 28;
const MIN_W = 280;
const MIN_H = 180;
const FLOATING_TYPES = new Set(['terminal', 'project-shell']);

export function isDesktopFloatingSurface(surface: WorkspaceSurface): boolean {
  return FLOATING_TYPES.has(surface.type);
}

export function isWindowedPresentationMode(mode: WorkspacePresentationMode): boolean {
  return mode === 'DESKTOP' || mode === 'MOSAIC';
}

/** Default presentation is floating windows (DESKTOP) inside the Terminals tab. */
export function createPresentationState(mode: WorkspacePresentationMode = 'DESKTOP'): WorkspacePresentationState {
  return {
    version: 2,
    mode,
    windows: {},
    nextZ: 1,
    canvas: { ...DEFAULT_CANVAS },
    tiled: mode === 'MOSAIC',
    activePtyViewId: '',
  };
}

export function migratePresentationState(raw: unknown): WorkspacePresentationState {
  const fallback = createPresentationState();
  if (!raw || typeof raw !== 'object') return fallback;
  const parsed = raw as Partial<WorkspacePresentationState> & { version?: number };
  const mode =
    parsed.mode === 'TABS' || parsed.mode === 'DESKTOP' || parsed.mode === 'MOSAIC'
      ? parsed.mode
      : 'DESKTOP';
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

  const activePtyViewId =
    typeof parsed.activePtyViewId === 'string' && parsed.activePtyViewId
      ? parsed.activePtyViewId
      : '';
  const tiled = mode === 'MOSAIC' ? parsed.tiled !== false : false;
  return {
    version: 2,
    mode,
    windows,
    nextZ,
    canvas,
    tiled,
    activePtyViewId,
  };
}

function clampWindow(
  canvas: ArrangeBounds,
  win: Pick<DesktopWindowState, 'x' | 'y' | 'width' | 'height'>
): Pick<DesktopWindowState, 'x' | 'y' | 'width' | 'height'> {
  const width = Math.min(Math.max(MIN_W, Math.round(win.width)), Math.max(MIN_W, canvas.width));
  const height = Math.min(Math.max(MIN_H, Math.round(win.height)), Math.max(MIN_H, canvas.height));
  const maxX = canvas.x + Math.max(0, canvas.width - width);
  const maxY = canvas.y + Math.max(0, canvas.height - height);
  return {
    x: Math.min(Math.max(canvas.x, Math.round(win.x)), maxX),
    y: Math.min(Math.max(canvas.y, Math.round(win.y)), maxY),
    width,
    height,
  };
}

function cascadeOrigin(canvas: ArrangeBounds, index: number): { x: number; y: number } {
  const maxStepsX = Math.max(0, Math.floor((canvas.width - DEFAULT_WINDOW_W) / CASCADE_OFFSET));
  const maxStepsY = Math.max(0, Math.floor((canvas.height - DEFAULT_WINDOW_H) / CASCADE_OFFSET));
  const step = maxStepsX === 0 && maxStepsY === 0 ? 0 : index % (Math.max(1, Math.min(maxStepsX, maxStepsY)) + 1);
  return {
    x: canvas.x + step * CASCADE_OFFSET,
    y: canvas.y + step * CASCADE_OFFSET,
  };
}

function createCascadedWindow(
  state: WorkspacePresentationState,
  viewId: string,
  cascadeIndex: number,
  zIndex: number
): DesktopWindowState {
  const origin = cascadeOrigin(state.canvas, cascadeIndex);
  const geometry = clampWindow(state.canvas, {
    x: origin.x,
    y: origin.y,
    width: DEFAULT_WINDOW_W,
    height: DEFAULT_WINDOW_H,
  });
  return {
    viewId,
    ...geometry,
    zIndex,
    minimized: false,
    maximized: false,
  };
}

function preserveChrome(previous: DesktopWindowState | undefined, next: DesktopWindowState): DesktopWindowState {
  if (!previous) return next;
  return {
    ...next,
    customTitle: previous.customTitle,
    accent: previous.accent,
    icon: previous.icon,
  };
}

function visibleViewIds(state: WorkspacePresentationState, preferred?: string[]): string[] {
  const ids =
    preferred ??
    Object.values(state.windows)
      .filter((win) => !win.minimized)
      .sort((a, b) => a.zIndex - b.zIndex)
      .map((win) => win.viewId);
  return ids.filter((id) => state.windows[id] && !state.windows[id].minimized);
}

/** Pack visible windows with Smart mosaic geometry. */
export function tileMosaicWindows(
  state: WorkspacePresentationState,
  viewIds?: string[]
): WorkspacePresentationState {
  const visible = visibleViewIds(state, viewIds);
  const tiles = arrangeSmart(state.canvas, visible);
  const windows = { ...state.windows };
  let nextZ = Math.max(1, state.nextZ);
  tiles.forEach((tile) => {
    const previous = windows[tile.viewId];
    const zIndex = previous?.zIndex ?? nextZ++;
    const geometry = clampWindow(state.canvas, tile);
    windows[tile.viewId] = preserveChrome(previous, {
      viewId: tile.viewId,
      ...geometry,
      zIndex,
      minimized: false,
      maximized: false,
    });
    nextZ = Math.max(nextZ, zIndex + 1);
  });
  return { ...state, windows, nextZ, tiled: true };
}

/** Keep mosaic tiles covering the canvas without discarding relative sizes. */
export function packMosaic(state: WorkspacePresentationState): WorkspacePresentationState {
  if (state.mode !== 'MOSAIC') return state;
  const visible = Object.values(state.windows).filter((win) => !win.minimized);
  if (visible.length === 0) return { ...state, tiled: true };
  if (visible.length === 1) {
    const only = visible[0];
    return {
      ...state,
      windows: {
        ...state.windows,
        [only.viewId]: preserveChrome(only, {
          ...only,
          x: state.canvas.x,
          y: state.canvas.y,
          width: state.canvas.width,
          height: state.canvas.height,
          maximized: false,
        }),
      },
      tiled: true,
    };
  }
  const scaled = scaleTilesToCanvas(visible, state.canvas, state.canvas);
  // scale with identical canvas is a no-op geometrically; re-normalize against gaps via arrange when coverage is broken.
  const area = visible.reduce((sum, win) => sum + win.width * win.height, 0);
  const canvasArea = state.canvas.width * state.canvas.height;
  if (area < canvasArea * 0.85) {
    return tileMosaicWindows(state);
  }
  const windows = { ...state.windows };
  scaled.forEach((tile) => {
    const current = windows[tile.viewId];
    if (!current) return;
    windows[tile.viewId] = preserveChrome(current, {
      ...current,
      ...clampWindow(state.canvas, tile),
      maximized: false,
    });
  });
  return { ...state, windows, tiled: true };
}

/** Cascade (DESKTOP) or Smart tile (MOSAIC). */
export function rearrangeSmart(
  state: WorkspacePresentationState,
  viewIds?: string[]
): WorkspacePresentationState {
  if (state.mode === 'MOSAIC') {
    return tileMosaicWindows(state, viewIds);
  }
  const visible = visibleViewIds(state, viewIds);
  const windows = { ...state.windows };
  let nextZ = Math.max(1, state.nextZ);
  visible.forEach((viewId, index) => {
    const previous = windows[viewId];
    const zIndex = previous?.zIndex ?? nextZ++;
    windows[viewId] = createCascadedWindow(state, viewId, index, zIndex);
    nextZ = Math.max(nextZ, zIndex + 1);
  });
  return { ...state, windows, nextZ, tiled: false };
}

export function setPresentationMode(
  state: WorkspacePresentationState,
  mode: WorkspacePresentationMode
): WorkspacePresentationState {
  if (state.mode === mode) return state;
  const next = { ...state, mode };
  if (mode === 'MOSAIC') return tileMosaicWindows(next);
  if (mode === 'DESKTOP') return { ...next, tiled: false };
  return { ...next, tiled: false };
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

  const withCanvas = { ...state, canvas: nextCanvas };
  if (state.mode === 'MOSAIC') {
    const visible = Object.values(state.windows).filter((win) => !win.minimized);
    if (visible.length === 0) return { ...withCanvas, tiled: true };
    const scaled = scaleTilesToCanvas(visible, state.canvas, nextCanvas);
    const windows = { ...state.windows };
    scaled.forEach((tile) => {
      const current = windows[tile.viewId];
      if (!current) return;
      windows[tile.viewId] = preserveChrome(current, {
        ...current,
        ...clampWindow(nextCanvas, tile),
        maximized: false,
      });
    });
    return packMosaic({ ...withCanvas, windows, tiled: true });
  }

  const visible = Object.values(state.windows).filter((win) => !win.minimized && !win.maximized);
  if (visible.length === 0) {
    return withCanvas;
  }

  const scaled = scaleTilesToCanvas(visible, state.canvas, nextCanvas);
  const windows = { ...state.windows };
  scaled.forEach((tile) => {
    const current = windows[tile.viewId];
    if (!current) return;
    const geometry = clampWindow(nextCanvas, {
      x: tile.x,
      y: tile.y,
      width: tile.width,
      height: tile.height,
    });
    windows[tile.viewId] = { ...current, ...geometry };
  });
  return { ...withCanvas, windows, tiled: false };
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
    if (Object.keys(state.windows).every((id) => nextIds.includes(id))) {
      if (state.activePtyViewId && nextIds.includes(state.activePtyViewId)) return state;
      return { ...state, activePtyViewId: nextIds[0] || '' };
    }
  }

  const windows: Record<string, DesktopWindowState> = {};
  const existingCount = Object.keys(preserved).length;
  nextIds.forEach((viewId, index) => {
    const previous = preserved[viewId];
    if (previous) {
      windows[viewId] = previous;
      return;
    }
    const cascadeIndex = existingCount + added.indexOf(viewId);
    const zIndex = nextZ++;
    windows[viewId] = createCascadedWindow(state, viewId, cascadeIndex >= 0 ? cascadeIndex : index, zIndex);
  });

  const activePtyViewId = added.length
    ? added[added.length - 1]
    : nextIds.includes(state.activePtyViewId)
      ? state.activePtyViewId
      : nextIds[0] || '';

  const synced: WorkspacePresentationState = {
    ...state,
    windows,
    nextZ,
    tiled: state.mode === 'MOSAIC',
    activePtyViewId,
  };
  if (state.mode === 'MOSAIC' && (added.length > 0 || removed.length > 0)) {
    return tileMosaicWindows(synced);
  }
  return { ...synced, tiled: false };
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

export function setActivePtyView(state: WorkspacePresentationState, viewId: string): WorkspacePresentationState {
  const next = (viewId || '').trim();
  if (state.activePtyViewId === next) return state;
  return { ...state, activePtyViewId: next };
}

export function focusDesktopWindow(state: WorkspacePresentationState, viewId: string): WorkspacePresentationState {
  if (!state.windows[viewId]) {
    return setActivePtyView(state, viewId);
  }
  const zIndex = state.nextZ;
  return {
    ...patchWindow(state, viewId, { zIndex, minimized: false }),
    nextZ: zIndex + 1,
    activePtyViewId: viewId,
  };
}

export function moveDesktopWindow(
  state: WorkspacePresentationState,
  viewId: string,
  x: number,
  y: number
): WorkspacePresentationState {
  const current = state.windows[viewId];
  if (!current || current.maximized || current.minimized) return state;
  if (state.mode === 'MOSAIC') {
    // Preview only — commit via commitMosaicMove on pointer up.
    return {
      ...patchWindow(state, viewId, { x: Math.round(x), y: Math.round(y) }),
      tiled: true,
    };
  }
  const geometry = clampWindow(state.canvas, {
    x,
    y,
    width: current.width,
    height: current.height,
  });
  return {
    ...patchWindow(state, viewId, { x: geometry.x, y: geometry.y }),
    tiled: false,
  };
}

/** Finish a mosaic drag: swap with the tile under the center, else retile nearest slot. */
export function commitMosaicMove(
  state: WorkspacePresentationState,
  viewId: string,
  origin: Pick<DesktopWindowState, 'x' | 'y' | 'width' | 'height'>
): WorkspacePresentationState {
  if (state.mode !== 'MOSAIC') return state;
  const current = state.windows[viewId];
  if (!current || current.minimized) return tileMosaicWindows(state);
  const cx = current.x + current.width / 2;
  const cy = current.y + current.height / 2;
  const target = Object.values(state.windows).find(
    (win) =>
      win.viewId !== viewId &&
      !win.minimized &&
      cx >= win.x &&
      cx <= win.x + win.width &&
      cy >= win.y &&
      cy <= win.y + win.height
  );
  if (target) {
    const windows = {
      ...state.windows,
      [viewId]: preserveChrome(current, {
        ...current,
        x: target.x,
        y: target.y,
        width: target.width,
        height: target.height,
        maximized: false,
      }),
      [target.viewId]: preserveChrome(target, {
        ...target,
        x: origin.x,
        y: origin.y,
        width: origin.width,
        height: origin.height,
        maximized: false,
      }),
    };
    return packMosaic({ ...state, windows, tiled: true, activePtyViewId: viewId });
  }
  return tileMosaicWindows({ ...state, activePtyViewId: viewId });
}

export function resizeDesktopWindow(
  state: WorkspacePresentationState,
  viewId: string,
  width: number,
  height: number
): WorkspacePresentationState {
  if (state.mode === 'MOSAIC') return state;
  const current = state.windows[viewId];
  if (!current || current.maximized || current.minimized) return state;
  const geometry = clampWindow(state.canvas, {
    x: current.x,
    y: current.y,
    width,
    height,
  });
  return {
    ...patchWindow(state, viewId, {
      x: geometry.x,
      y: geometry.y,
      width: geometry.width,
      height: geometry.height,
    }),
    tiled: false,
  };
}

function neighborsOnEdge(
  state: WorkspacePresentationState,
  viewId: string,
  side: 'n' | 's' | 'e' | 'w'
): DesktopWindowState[] {
  const self = state.windows[viewId];
  if (!self) return [];
  const gap = 14;
  return Object.values(state.windows).filter((win) => {
    if (win.viewId === viewId || win.minimized) return false;
    if (side === 'e') {
      return (
        Math.abs(self.x + self.width + 8 - win.x) <= gap &&
        rangesOverlapY(self, win)
      );
    }
    if (side === 'w') {
      return (
        Math.abs(win.x + win.width + 8 - self.x) <= gap &&
        rangesOverlapY(self, win)
      );
    }
    if (side === 's') {
      return (
        Math.abs(self.y + self.height + 8 - win.y) <= gap &&
        rangesOverlapX(self, win)
      );
    }
    return (
      Math.abs(win.y + win.height + 8 - self.y) <= gap &&
      rangesOverlapX(self, win)
    );
  });
}

function rangesOverlapX(a: DesktopWindowState, b: DesktopWindowState): boolean {
  return a.x < b.x + b.width - 4 && b.x < a.x + a.width - 4;
}

function rangesOverlapY(a: DesktopWindowState, b: DesktopWindowState): boolean {
  return a.y < b.y + b.height - 4 && b.y < a.y + a.height - 4;
}

function applyEdgeDelta(
  state: WorkspacePresentationState,
  viewId: string,
  side: 'n' | 's' | 'e' | 'w',
  delta: number
): WorkspacePresentationState {
  if (delta === 0) return state;
  const self = state.windows[viewId];
  if (!self) return state;
  const neighbors = neighborsOnEdge(state, viewId, side);
  let next = state;
  if (neighbors.length === 0) {
    // Grow/shrink against canvas edge only.
    let patch: Partial<DesktopWindowState> = {};
    if (side === 'e') {
      const width = Math.max(MIN_W, self.width + delta);
      patch = { width: Math.min(width, state.canvas.x + state.canvas.width - self.x) };
    } else if (side === 'w') {
      const nextX = Math.min(self.x + delta, self.x + self.width - MIN_W);
      const clampedX = Math.max(state.canvas.x, nextX);
      patch = { x: clampedX, width: self.width - (clampedX - self.x) };
    } else if (side === 's') {
      const height = Math.max(MIN_H, self.height + delta);
      patch = { height: Math.min(height, state.canvas.y + state.canvas.height - self.y) };
    } else {
      const nextY = Math.min(self.y + delta, self.y + self.height - MIN_H);
      const clampedY = Math.max(state.canvas.y, nextY);
      patch = { y: clampedY, height: self.height - (clampedY - self.y) };
    }
    return { ...patchWindow(next, viewId, patch), tiled: true };
  }
  for (const neighbor of neighbors) {
    const first = side === 'e' || side === 's' ? next.windows[viewId]! : neighbor;
    const second = side === 'e' || side === 's' ? neighbor : next.windows[viewId]!;
    const orientation = side === 'e' || side === 'w' ? 'vertical' : 'horizontal';
    const signed = side === 'w' || side === 'n' ? -delta : delta;
    const patched = applySharedEdgeDelta(first, second, orientation, signed);
    if (!patched) continue;
    const windows = { ...next.windows };
    windows[first.viewId] = preserveChrome(windows[first.viewId], {
      ...windows[first.viewId]!,
      x: patched.first.x,
      y: patched.first.y,
      width: patched.first.width,
      height: patched.first.height,
      maximized: false,
    });
    windows[second.viewId] = preserveChrome(windows[second.viewId], {
      ...windows[second.viewId]!,
      x: patched.second.x,
      y: patched.second.y,
      width: patched.second.width,
      height: patched.second.height,
      maximized: false,
    });
    next = { ...next, windows, tiled: true };
  }
  return next;
}

/** Grow/shrink a mosaic tile; adjacent tiles on that edge absorb the delta. */
export function resizeMosaicWindow(
  state: WorkspacePresentationState,
  viewId: string,
  edge: MosaicResizeEdge,
  deltaX: number,
  deltaY: number
): WorkspacePresentationState {
  if (state.mode !== 'MOSAIC') return state;
  const current = state.windows[viewId];
  if (!current || current.minimized) return state;
  let next = state;
  if (edge.includes('e')) next = applyEdgeDelta(next, viewId, 'e', deltaX);
  if (edge.includes('w')) next = applyEdgeDelta(next, viewId, 'w', deltaX);
  if (edge.includes('s')) next = applyEdgeDelta(next, viewId, 's', deltaY);
  if (edge.includes('n')) next = applyEdgeDelta(next, viewId, 'n', deltaY);
  return packMosaic(next);
}

export function patchDesktopWindowChrome(
  state: WorkspacePresentationState,
  viewId: string,
  chrome: Pick<DesktopWindowState, 'customTitle' | 'accent' | 'icon'>
): WorkspacePresentationState {
  if (!state.windows[viewId]) return state;
  return patchWindow(state, viewId, chrome);
}

/** Shared-edge resize for MOSAIC splitters. No-op in DESKTOP/TABS. */
export function resizeAdjacentDesktopWindows(
  state: WorkspacePresentationState,
  firstId: string,
  secondId: string,
  orientation: 'vertical' | 'horizontal',
  delta: number
): WorkspacePresentationState {
  if (state.mode !== 'MOSAIC') return state;
  const first = state.windows[firstId];
  const second = state.windows[secondId];
  if (!first || !second || first.minimized || second.minimized) return state;
  const patched = applySharedEdgeDelta(first, second, orientation, delta);
  if (!patched) return state;
  const windows = {
    ...state.windows,
    [firstId]: {
      ...first,
      x: patched.first.x,
      y: patched.first.y,
      width: patched.first.width,
      height: patched.first.height,
      maximized: false,
    },
    [secondId]: {
      ...second,
      x: patched.second.x,
      y: patched.second.y,
      width: patched.second.width,
      height: patched.second.height,
      maximized: false,
    },
  };
  return packMosaic({ ...state, windows, tiled: true });
}

export function toggleDesktopMinimize(state: WorkspacePresentationState, viewId: string): WorkspacePresentationState {
  const current = state.windows[viewId];
  if (!current) return state;
  const minimized = !current.minimized;
  const next: WorkspacePresentationState = {
    ...patchWindow(state, viewId, { minimized, maximized: false }),
    tiled: state.mode === 'MOSAIC',
    activePtyViewId: minimized
      ? state.activePtyViewId === viewId
        ? Object.values(state.windows)
            .filter((win) => win.viewId !== viewId && !win.minimized)
            .sort((a, b) => b.zIndex - a.zIndex)[0]?.viewId || ''
        : state.activePtyViewId
      : viewId,
  };
  if (state.mode === 'MOSAIC') {
    return tileMosaicWindows(next);
  }
  return { ...next, tiled: false };
}

export function toggleDesktopMaximize(state: WorkspacePresentationState, viewId: string): WorkspacePresentationState {
  const current = state.windows[viewId];
  if (!current) return state;
  // Mosaic uses splitters, not maximize chrome.
  if (state.mode === 'MOSAIC') {
    return focusDesktopWindow(state, viewId);
  }
  return {
    ...patchWindow(state, viewId, { maximized: !current.maximized, minimized: false }),
    tiled: false,
    activePtyViewId: viewId,
  };
}
