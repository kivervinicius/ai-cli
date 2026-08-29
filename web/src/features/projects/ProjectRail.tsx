import React, { useState } from 'react';
import { Bot, BrainCircuit, FolderGit2, Gauge, Home, Plus, Settings, X } from 'lucide-react';
import { Button, Dialog, IconButton, Input } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { Project } from '../../types';
import { useTranslation } from 'react-i18next';

export const ProjectRail: React.FC<{
  projects: Project[];
  selected?: Project | null;
  open: boolean;
  onClose: () => void;
  onSelect: (project: Project) => void;
  onCreated: (project: Project) => void;
  onOpenGlobal: (kind: 'overview'|'agents'|'resources'|'maestro'|'sessions'|'settings') => void;
}> = ({ projects, selected, open, onClose, onSelect, onCreated, onOpenGlobal }) => {
  const { t } = useTranslation();
  const [addOpen,setAddOpen]=useState(false); const [path,setPath]=useState(''); const [name,setName]=useState(''); const [busy,setBusy]=useState(false); const [error,setError]=useState('');
  const create = async () => { if(!path.trim())return; setBusy(true); setError(''); try{const project=await nexus.createProject(path.trim(),name.trim()||undefined);onCreated(project);setPath('');setName('');setAddOpen(false);}catch(e){setError(e instanceof Error?e.message:String(e));}finally{setBusy(false);} };
  const global=[{id:'overview',label:t('nav.overview'),icon:Home},{id:'agents',label:t('nav.agents'),icon:Bot},{id:'resources',label:t('nav.resources'),icon:Gauge},{id:'maestro',label:'Maestro',icon:BrainCircuit},{id:'settings',label:t('nav.settings'),icon:Settings}] as const;
  return <><aside className="nx-project-rail" data-open={open?'true':'false'} data-tour="projects" aria-label={t('rail.aria')}><div className="nx-project-rail__brand"><span className="nx-brand-mark">N</span><span><strong>IAPro Nexus</strong><small>Powered by Maestro</small></span><IconButton className="nx-project-rail__mobile-close" label={t('rail.close')} onClick={onClose}><X size={15}/></IconButton></div><div className="nx-project-rail__global">{global.map((item)=><button type="button" key={item.id} onClick={()=>{onOpenGlobal(item.id);onClose();}}><item.icon size={15}/><span>{item.label}</span></button>)}</div><div className="nx-project-rail__heading"><span>{t('rail.projects')}</span><IconButton label={t('rail.add')} onClick={()=>setAddOpen(true)}><Plus size={14}/></IconButton></div><div className="nx-project-list">{projects.map((project)=><button type="button" key={project.id} data-active={selected?.id===project.id?'true':'false'} onClick={()=>{onSelect(project);onClose();}} title={project.canonical_path}><span className="nx-project-avatar">{project.name.slice(0,2).toUpperCase()}</span><span><strong>{project.name}</strong><small>{project.default_branch||'main'}</small></span>{selected?.id===project.id&&<span className="nx-project-active-dot"/>}</button>)}{projects.length===0&&<p>{t('rail.empty')}</p>}</div><div className="nx-project-rail__footer"><Button size="sm" tone="ghost" onClick={()=>setAddOpen(true)}><FolderGit2 size={13}/> {t('rail.addLocal')}</Button></div></aside><div className="nx-project-rail-overlay" data-open={open?'true':'false'} onClick={onClose}/><Dialog open={addOpen} onClose={()=>setAddOpen(false)} title={t('rail.add')}><div className="nx-form-stack"><label>{t('rail.path')}<Input value={path} onChange={setPath} onEnter={create} placeholder="/path/to/repository" mono/></label><label>{t('rail.name')}<Input value={name} onChange={setName} placeholder={t('rail.namePlaceholder')}/></label>{error&&<p className="nx-error-copy">{error}</p>}<div className="nx-dialog-actions"><Button onClick={()=>setAddOpen(false)}>{t('rail.cancel')}</Button><Button tone="brand" disabled={!path.trim()||busy} onClick={create}>{busy?t('rail.adding'):t('rail.add')}</Button></div></div></Dialog></>;
};
