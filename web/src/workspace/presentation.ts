import { surfaceViewId, type WorkspaceSurface } from './model';

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
  version: 1;
  mode: WorkspacePresentationMode;
  windows: Record<string, DesktopWindowState>;
  nextZ: number;
}

export function createPresentationState(mode: WorkspacePresentationMode = 'TABS'): WorkspacePresentationState {
  return { version: 1, mode, windows: {}, nextZ: 1 };
}

function defaultWindow(viewId: string, index: number, zIndex: number): DesktopWindowState {
  const offset = (index % 7) * 28;
  return { viewId, x: 32 + offset, y: 28 + offset, width: 720, height: 500, zIndex, minimized: false, maximized: false };
}

export function syncDesktopWindows(state: WorkspacePresentationState, surfaces: WorkspaceSurface[]): WorkspacePresentationState {
  const windows: Record<string, DesktopWindowState> = {};
  let nextZ = Math.max(1, state.nextZ);
  surfaces.forEach((surface, index) => {
    const viewId = surfaceViewId(surface);
    const current = state.windows[viewId];
    if (current) {
      windows[viewId] = current;
      nextZ = Math.max(nextZ, current.zIndex + 1);
    } else {
      windows[viewId] = defaultWindow(viewId, index, nextZ++);
    }
  });
  return { ...state, windows, nextZ };
}

function patchWindow(state: WorkspacePresentationState, viewId: string, patch: Partial<DesktopWindowState>): WorkspacePresentationState {
  const current = state.windows[viewId];
  if (!current) return state;
  return { ...state, windows: { ...state.windows, [viewId]: { ...current, ...patch } } };
}

export function focusDesktopWindow(state: WorkspacePresentationState, viewId: string): WorkspacePresentationState {
  if (!state.windows[viewId]) return state;
  const zIndex = state.nextZ;
  return { ...patchWindow(state, viewId, { zIndex, minimized: false }), nextZ: zIndex + 1 };
}

export function moveDesktopWindow(state: WorkspacePresentationState, viewId: string, x: number, y: number): WorkspacePresentationState {
  return patchWindow(state, viewId, { x: Math.max(0, Math.round(x)), y: Math.max(0, Math.round(y)) });
}

export function resizeDesktopWindow(state: WorkspacePresentationState, viewId: string, width: number, height: number): WorkspacePresentationState {
  return patchWindow(state, viewId, { width: Math.max(360, Math.round(width)), height: Math.max(240, Math.round(height)) });
}

export function toggleDesktopMinimize(state: WorkspacePresentationState, viewId: string): WorkspacePresentationState {
  const current = state.windows[viewId];
  if (!current) return state;
  return patchWindow(state, viewId, { minimized: !current.minimized, maximized: false });
}

export function toggleDesktopMaximize(state: WorkspacePresentationState, viewId: string): WorkspacePresentationState {
  const current = state.windows[viewId];
  if (!current) return state;
  return patchWindow(state, viewId, { maximized: !current.maximized, minimized: false });
}
