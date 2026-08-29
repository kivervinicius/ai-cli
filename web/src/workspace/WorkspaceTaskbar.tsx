import React from 'react';
import { AppWindow, TerminalSquare } from 'lucide-react';
import { listSurfaces } from './model';
import { useWorkspace } from './WorkspaceProvider';
import { useTranslation } from 'react-i18next';

export const WorkspaceTaskbar: React.FC = () => {
  const { t } = useTranslation();
  const { model, activate } = useWorkspace();
  const surfaces = listSurfaces(model.root);
  return <div className="nx-workspace-taskbar" data-tour="taskbar" role="toolbar" aria-label={t('shell.openSurfaces')}>
    {surfaces.map((surface) => <button key={surface.id} type="button" data-active={surface.id === model.maximizedSurfaceId ? 'true' : undefined} onClick={() => activate(surface.id)} title={surface.subtitle || surface.title}>{surface.type === 'terminal' ? <TerminalSquare size={13} /> : <AppWindow size={13} />}<span>{surface.title}</span></button>)}
  </div>;
};
