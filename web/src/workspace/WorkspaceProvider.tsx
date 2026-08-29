import React, { createContext, useContext, useEffect, useMemo, useReducer } from 'react';
import {
  closeSurface,
  createWorkspace,
  moveSurface,
  openSurface,
  setActiveSurface,
  setSplitRatio,
  splitWithSurface,
  toggleMaximize,
  type WorkspaceDirection,
  type WorkspaceModel,
  type WorkspaceSurface,
} from './model';
import { deserializeWorkspace, serializeWorkspace, workspaceStorageKey } from './state';

interface WorkspaceContextValue {
  model: WorkspaceModel;
  open: (surface: WorkspaceSurface, targetStackId?: string) => void;
  activate: (surfaceId: string) => void;
  close: (surfaceId: string) => void;
  split: (relativeSurfaceId: string, surface: WorkspaceSurface, direction: WorkspaceDirection) => void;
  move: (surfaceId: string, targetStackId: string) => void;
  resize: (splitId: string, ratio: number) => void;
  maximize: (surfaceId: string) => void;
  reset: () => void;
}

type Action =
  | { type: 'replace'; model: WorkspaceModel }
  | { type: 'open'; surface: WorkspaceSurface; stackId?: string }
  | { type: 'activate'; surfaceId: string }
  | { type: 'close'; surfaceId: string }
  | { type: 'split'; relativeSurfaceId: string; surface: WorkspaceSurface; direction: WorkspaceDirection }
  | { type: 'move'; surfaceId: string; targetStackId: string }
  | { type: 'resize'; splitId: string; ratio: number }
  | { type: 'maximize'; surfaceId: string };

function reducer(model: WorkspaceModel, action: Action): WorkspaceModel {
  switch (action.type) {
    case 'replace': return action.model;
    case 'open': return openSurface(model, action.surface, action.stackId);
    case 'activate': return setActiveSurface(model, action.surfaceId);
    case 'close': return closeSurface(model, action.surfaceId);
    case 'split': return splitWithSurface(model, action.relativeSurfaceId, action.surface, action.direction);
    case 'move': return moveSurface(model, action.surfaceId, action.targetStackId);
    case 'resize': return setSplitRatio(model, action.splitId, action.ratio);
    case 'maximize': return toggleMaximize(model, action.surfaceId);
  }
}

const WorkspaceContext = createContext<WorkspaceContextValue | null>(null);

const defaultSurface = (): WorkspaceSurface => ({ id: 'project-overview', type: 'overview', title: 'Overview', closable: false });

export const WorkspaceProvider: React.FC<{
  projectId: string;
  initialLayout?: string;
  saveLayout?: (layout: string) => Promise<unknown>;
  children: React.ReactNode;
}> = ({ projectId, initialLayout, saveLayout, children }) => {
  const fallback = useMemo(() => createWorkspace(defaultSurface()), []);
  const [model, dispatch] = useReducer(reducer, fallback);

  useEffect(() => {
    const local = window.localStorage.getItem(workspaceStorageKey(projectId));
    dispatch({ type: 'replace', model: deserializeWorkspace(initialLayout || local, fallback) });
  }, [projectId, initialLayout, fallback]);

  useEffect(() => {
    const serialized = serializeWorkspace(model);
    window.localStorage.setItem(workspaceStorageKey(projectId), serialized);
    if (!saveLayout) return;
    const timer = window.setTimeout(() => { void saveLayout(serialized).catch(() => undefined); }, 500);
    return () => window.clearTimeout(timer);
  }, [model, projectId, saveLayout]);

  const value = useMemo<WorkspaceContextValue>(() => ({
    model,
    open: (surface, stackId) => dispatch({ type: 'open', surface, stackId }),
    activate: (surfaceId) => dispatch({ type: 'activate', surfaceId }),
    close: (surfaceId) => dispatch({ type: 'close', surfaceId }),
    split: (relativeSurfaceId, surface, direction) => dispatch({ type: 'split', relativeSurfaceId, surface, direction }),
    move: (surfaceId, targetStackId) => dispatch({ type: 'move', surfaceId, targetStackId }),
    resize: (splitId, ratio) => dispatch({ type: 'resize', splitId, ratio }),
    maximize: (surfaceId) => dispatch({ type: 'maximize', surfaceId }),
    reset: () => dispatch({ type: 'replace', model: fallback }),
  }), [model, fallback]);

  return <WorkspaceContext.Provider value={value}>{children}</WorkspaceContext.Provider>;
};

export function useWorkspace(): WorkspaceContextValue {
  const context = useContext(WorkspaceContext);
  if (!context) throw new Error('useWorkspace must be used inside WorkspaceProvider');
  return context;
}
