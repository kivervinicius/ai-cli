import React from 'react';
import { NexusWorkspaceApp } from './app/NexusWorkspaceApp';
import { NexusDemoApp } from './app/NexusDemoApp';
import type { WorkspaceSurface } from './workspace/model';

function parsePopoutSurface(): WorkspaceSurface | undefined {
  const raw=new URLSearchParams(window.location.search).get('popout');
  if(!raw)return undefined;
  try{return JSON.parse(raw) as WorkspaceSurface;}catch{try{return JSON.parse(decodeURIComponent(raw)) as WorkspaceSurface;}catch{return undefined;}}
}
export const App:React.FC=()=> new URLSearchParams(window.location.search).get('demo') === '1' ? <NexusDemoApp/> : <NexusWorkspaceApp popoutSurface={parsePopoutSurface()}/>;
