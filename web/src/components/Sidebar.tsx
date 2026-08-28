import React from 'react';
import { Workspace } from '../types';
import { LayoutDashboard, Terminal, Cpu, History, FolderGit2, ShieldCheck } from 'lucide-react';

interface SidebarProps {
  currentTab: string;
  onSelectTab: (tab: string) => void;
  workspaces: Workspace[];
  activeWorkspace: string;
  onSelectWorkspace: (path: string) => void;
  runtimeCount: number;
}

export const Sidebar: React.FC<SidebarProps> = ({
  currentTab,
  onSelectTab,
  workspaces,
  activeWorkspace,
  onSelectWorkspace,
  runtimeCount,
}) => {
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
          <FolderGit2 className="w-3.5 h-3.5 text-slate-500" />
        </div>
        <div className="mt-1 space-y-1">
          {workspaces.map((ws) => {
            const isSelected = activeWorkspace === ws.path;
            return (
              <button
                key={ws.path}
                onClick={() => onSelectWorkspace(ws.path)}
                className={`w-full text-left px-3 py-2 rounded-md text-xs transition ${
                  isSelected
                    ? 'bg-slate-800 text-white font-semibold border-l-2 border-sky-500'
                    : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900/40'
                }`}
              >
                <div className="truncate font-medium">{ws.name}</div>
                <div className="truncate text-[10px] font-mono text-slate-400">{ws.path}</div>
              </button>
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
