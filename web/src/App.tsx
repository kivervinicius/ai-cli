import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { NexusWorkspaceApp } from './app/NexusWorkspaceApp';
import { NexusDemoApp } from './app/NexusDemoApp';
import type { WorkspaceSurface } from './workspace/model';

function parseLegacyPopoutSurface(): WorkspaceSurface | undefined {
  if (typeof window === 'undefined') return undefined;
  const raw = new URLSearchParams(window.location.search).get('popout');
  if (!raw) return undefined;
  try {
    return JSON.parse(raw) as WorkspaceSurface;
  } catch {
    try {
      return JSON.parse(decodeURIComponent(raw)) as WorkspaceSurface;
    } catch {
      return undefined;
    }
  }
}

export const AppRoutes: React.FC = () => {
  const legacyPopout = parseLegacyPopoutSurface();
  if (legacyPopout) {
    return <NexusWorkspaceApp popoutSurface={legacyPopout} />;
  }

  return (
    <Routes>
      <Route path="/demo" element={<NexusDemoApp />} />
      <Route path="/p/:projectId/popout/:surface" element={<NexusWorkspaceApp />} />
      <Route path="/p/:projectId/*" element={<NexusWorkspaceApp />} />
      <Route path="/projects" element={<NexusWorkspaceApp initialGlobalSurface="projects" />} />
      <Route path="/settings" element={<NexusWorkspaceApp initialGlobalSurface="settings" />} />
      <Route path="/updates" element={<NexusWorkspaceApp initialGlobalSurface="updates" />} />
      <Route path="/welcome" element={<NexusWorkspaceApp initialGlobalSurface="welcome" />} />
      <Route path="/" element={<NexusWorkspaceApp />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
};

export const App: React.FC = () => {
  if (
    typeof window !== 'undefined' &&
    new URLSearchParams(window.location.search).get('demo') === '1'
  ) {
    return <NexusDemoApp />;
  }

  return (
    <BrowserRouter>
      <AppRoutes />
    </BrowserRouter>
  );
};
