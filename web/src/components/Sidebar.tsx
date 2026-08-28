import React, { useState } from 'react';
import { Workspace } from '../types';
import { LayoutDashboard, Terminal, Cpu, History, FolderGit2, ShieldCheck, Plus, Trash2, Check, X } from 'lucide-react';

interface SidebarProps {
  currentTab: string;
  onSelectTab: (tab: string) => void;
  workspaces: Workspace[];
  activeWorkspace: string;
  onSelectWorkspace: (path: string) => void;
  runtimeCount: number;
  onAddWorkspace?: (path: string, name?: string) => void;
  onRemoveWorkspace?: (path: string) => void;
}

export const Sidebar: React.FC<SidebarProps> = ({
  currentTab,
  onSelectTab,
  workspaces,
  activeWorkspace,
  onSelectWorkspace,
  runtimeCount,
  onAddWorkspace,
  onRemoveWorkspace,
}) => {
  const [showAdd, setShowAdd] = useState(false);
  const [newPath, setNewPath] = useState('');
  const [newName, setNewName] = useState('');

  const handleAddSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (newPath.trim() && onAddWorkspace) {
      onAddWorkspace(newPath.trim(), newName.trim() || undefined);
      setNewPath('');
      setNewName('');
      setShowAdd(false);
    }
  };
  const navItems = [
    { id: 'dashboard', label: 'Dashboard', icon: LayoutDashboard },
    { id: 'terminals', label: 'Terminals', icon: Terminal, badge: runtimeCount > 0 ? runtimeCount : undefined },
    { id: 'runtimes', label: 'Runtimes', icon: Cpu },
    { id: 'providers', label: 'Providers', icon: ShieldCheck },
    { id: 'events', label: 'Event Log', icon: History },
  ];

  return (
    <aside className="w-64 bg-slate-950 border-r border-slate-800 flex flex-col h-full select-none">
      {/* Brand Header */}
      <div className="p-4 border-b border-slate-800 flex items-center space-x-3">
        <div className="w-8 h-8 rounded-lg bg-gradient-to-tr from-sky-600 to-indigo-600 flex items-center justify-center font-black text-white text-sm shadow-lg shadow-sky-900/30">
          AI
        </div>
        <div>
          <h1 className="text-sm font-bold text-slate-100 tracking-wide">AI Control Center</h1>
          <p className="text-[11px] font-mono text-slate-400">Local Control Plane v0.4.0</p>
        </div>
      </div>

      {/* Navigation Links */}
      <div className="p-3 space-y-1">
        <div className="px-3 py-1.5 text-[10px] font-semibold font-mono tracking-wider text-slate-500 uppercase">
          Navigation
        </div>
        {navItems.map((item) => {
          const Icon = item.icon;
          const isActive = currentTab === item.id;
          return (
            <button
              key={item.id}
              onClick={() => onSelectTab(item.id)}
              className={`w-full flex items-center justify-between px-3 py-2 rounded-md text-xs font-medium transition ${
                isActive
                  ? 'bg-sky-600/15 text-sky-400 border border-sky-500/30 shadow-sm'
                  : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900/60'
              }`}
            >
              <div className="flex items-center space-x-2.5">
                <Icon className={`w-4 h-4 ${isActive ? 'text-sky-400' : 'text-slate-400'}`} />
                <span>{item.label}</span>
              </div>
              {item.badge !== undefined && (
                <span className="px-1.5 py-0.5 text-[10px] font-mono font-bold bg-sky-500/20 text-sky-300 rounded-full">
                  {item.badge}
                </span>
              )}
            </button>
          );
        })}
      </div>

      {/* Projects & Workspaces */}
      <div className="p-3 flex-1 overflow-y-auto border-t border-slate-900">
        <div className="px-3 py-1.5 text-[10px] font-semibold font-mono tracking-wider text-slate-500 uppercase flex items-center justify-between">
          <span>Projects</span>
          <div className="flex items-center space-x-1">
            <button
              onClick={() => setShowAdd(!showAdd)}
              className="p-1 text-slate-400 hover:text-sky-400 hover:bg-slate-800 rounded transition"
              title="Add New Project"
            >
              <Plus className="w-3.5 h-3.5" />
            </button>
            <FolderGit2 className="w-3.5 h-3.5 text-slate-500" />
          </div>
        </div>

        {showAdd && (
          <form onSubmit={handleAddSubmit} className="mb-2 p-2 bg-slate-900 border border-slate-800 rounded text-xs space-y-1.5 font-mono">
            <input
              type="text"
              placeholder="/path/to/project"
              value={newPath}
              onChange={(e) => setNewPath(e.target.value)}
              className="w-full bg-slate-950 border border-slate-700 rounded px-2 py-1 text-[11px] text-slate-100 placeholder-slate-600 focus:outline-none focus:border-sky-500"
              autoFocus
            />
            <input
              type="text"
              placeholder="Project Name (optional)"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              className="w-full bg-slate-950 border border-slate-700 rounded px-2 py-1 text-[11px] text-slate-100 placeholder-slate-600 focus:outline-none focus:border-sky-500"
            />
            <div className="flex justify-end space-x-1 pt-1">
              <button
                type="button"
                onClick={() => setShowAdd(false)}
                className="px-2 py-0.5 rounded text-[10px] text-slate-400 hover:text-slate-200"
              >
                Cancel
              </button>
              <button
                type="submit"
                className="px-2.5 py-0.5 rounded text-[10px] bg-sky-600 hover:bg-sky-500 text-white font-medium"
              >
                Add
              </button>
            </div>
          </form>
        )}

        <div className="mt-1 space-y-1">
          {workspaces.map((ws) => {
            const isSelected = activeWorkspace === ws.path;
            return (
              <div
                key={ws.path}
                className={`group flex items-center justify-between px-3 py-2 rounded-md text-xs transition ${
                  isSelected
                    ? 'bg-slate-800 text-white font-semibold border-l-2 border-sky-500'
                    : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900/40'
                }`}
              >
                <button
                  onClick={() => onSelectWorkspace(ws.path)}
                  className="flex-1 text-left truncate min-w-0"
                >
                  <div className="truncate font-medium">{ws.name}</div>
                  <div className="truncate text-[10px] font-mono text-slate-500">{ws.path}</div>
                </button>
                {onRemoveWorkspace && workspaces.length > 1 && (
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      onRemoveWorkspace(ws.path);
                    }}
                    className="opacity-0 group-hover:opacity-100 p-1 text-slate-500 hover:text-rose-400 hover:bg-slate-800 rounded transition ml-1"
                    title="Remove Project"
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                )}
              </div>
            );
          })}
        </div>
      </div>

      {/* Footer Info */}
      <div className="p-3 border-t border-slate-800/80 bg-slate-950/60 text-[11px] font-mono text-slate-400 flex items-center justify-between">
        <span>Loopback Mode</span>
        <span className="text-emerald-400">● 127.0.0.1</span>
      </div>
    </aside>
  );
};
