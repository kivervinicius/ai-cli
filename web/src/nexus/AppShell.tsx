import React, { useEffect, useRef, useState } from 'react';
import { EmptyState } from '../ui/primitives';

export interface NavItem {
  id: string;
  label: string;
  section: 'nexus' | 'legacy';
  hint: string;
}

export const NEXUS_NAV: NavItem[] = [
  { id: 'overview', label: 'Overview', section: 'nexus', hint: 'Projects, agents and activity at a glance' },
  { id: 'projects', label: 'Projects', section: 'nexus', hint: 'Project-first workspace' },
  { id: 'agents', label: 'Agents', section: 'nexus', hint: 'Persistent agents across runtimes' },
  { id: 'resources', label: 'Resources', section: 'nexus', hint: 'Providers, accounts, quotas' },
  { id: 'maestro', label: 'Maestro', section: 'nexus', hint: 'Process, skills and verification recommendations' },
  { id: 'missions', label: 'Missions', section: 'nexus', hint: 'Mission planning and task tracking (Beta)' },
  { id: 'sessions', label: 'Sessions', section: 'nexus', hint: 'Sessions and continuity' },
  { id: 'settings', label: 'Settings', section: 'nexus', hint: 'Nexus settings' },
];

export const LEGACY_NAV: NavItem[] = [
  { id: 'runtimes', label: 'Runtimes', section: 'legacy', hint: 'Legacy runtime control' },
  { id: 'providers', label: 'Providers', section: 'legacy', hint: 'Provider detection and capabilities' },
  { id: 'events', label: 'Events', section: 'legacy', hint: 'Operational events' },
];

export const AppShell: React.FC<{
  current: string;
  onNavigate: (id: string) => void;
  onCommandPalette: () => void;
  children: React.ReactNode;
}> = ({ current, onNavigate, onCommandPalette, children }) => {
  const [sidebarOpen, setSidebarOpen] = useState(false);

  // Close sidebar on escape key
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && sidebarOpen) setSidebarOpen(false);
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [sidebarOpen]);

  return (
    <div className="flex h-screen w-screen bg-[var(--nx-bg)] text-[var(--nx-text)] overflow-hidden">
      {/* Mobile overlay */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 lg:hidden"
          onClick={() => setSidebarOpen(false)}
          aria-hidden="true"
        />
      )}

      {/* Sidebar */}
      <aside
        className={`fixed inset-y-0 left-0 z-50 w-56 border-r border-slate-800/70 bg-slate-950/60 flex flex-col shrink-0 transform transition-transform duration-200 lg:relative lg:translate-x-0 ${
          sidebarOpen ? 'translate-x-0' : '-translate-x-full'
        }`}
        role="navigation"
        aria-label="Main navigation"
      >
        <div className="px-4 py-4">
          <div className="text-sm font-semibold tracking-tight">
            <span className="iapro-gradient-text">IAPro Nexus</span>
          </div>
          <div className="text-[10px] text-slate-500 mt-0.5">Powered by Maestro</div>
        </div>

        <nav className="flex-1 overflow-y-auto px-2 pb-4">
          <div className="px-2 py-1 text-[10px] uppercase tracking-wider text-slate-600">Nexus</div>
          {NEXUS_NAV.map((item) => (
            <button
              key={item.id}
              onClick={() => {
                onNavigate(item.id);
                setSidebarOpen(false);
              }}
              className={`w-full text-left px-2 py-1.5 rounded-[var(--nx-radius-sm)] text-sm transition ${
                current === item.id
                  ? 'bg-indigo-950/50 text-indigo-200 border border-indigo-800/50'
                  : 'text-slate-400 hover:text-slate-100 hover:bg-slate-800/50'
              }`}
              title={item.hint}
              aria-current={current === item.id ? 'page' : undefined}
            >
              {item.label}
            </button>
          ))}
          <div className="px-2 py-1 mt-3 text-[10px] uppercase tracking-wider text-slate-600">Legacy</div>
          {LEGACY_NAV.map((item) => (
            <button
              key={item.id}
              onClick={() => {
                onNavigate(item.id);
                setSidebarOpen(false);
              }}
              className={`w-full text-left px-2 py-1.5 rounded-[var(--nx-radius-sm)] text-sm transition ${
                current === item.id
                  ? 'bg-slate-800 text-slate-100 border border-slate-700'
                  : 'text-slate-500 hover:text-slate-300 hover:bg-slate-800/40'
              }`}
              aria-current={current === item.id ? 'page' : undefined}
            >
              {item.label}
            </button>
          ))}
        </nav>
      </aside>

      <main className="flex-1 flex flex-col min-w-0">
        <header className="h-12 border-b border-slate-800/70 px-4 flex items-center justify-between text-xs shrink-0">
          <div className="flex items-center gap-2">
            <button
              onClick={() => setSidebarOpen(!sidebarOpen)}
              className="lg:hidden p-1.5 rounded hover:bg-slate-800 text-slate-400"
              aria-label="Toggle navigation menu"
            >
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
              </svg>
            </button>
            <span className="font-mono text-slate-500">
              {current} · <span className="text-slate-300">{window.location.hostname}</span>
            </span>
          </div>
          <button
            onClick={onCommandPalette}
            className="flex items-center gap-2 px-2.5 py-1 rounded-md border border-slate-800 bg-slate-900/60 text-slate-400 hover:text-slate-200"
            aria-label="Open command palette"
          >
            <span>Search &amp; commands</span>
            <kbd className="text-[10px] border border-slate-700 rounded px-1">Ctrl K</kbd>
          </button>
        </header>
        <div className="flex-1 p-4 overflow-y-auto min-h-0">{children}</div>
      </main>
    </div>
  );
};

export const PlaceholderPage: React.FC<{ title: string; hint: string }> = ({ title, hint }) => (
  <EmptyState title={title} hint={hint} />
);

/* Minimal command palette (§83): Ctrl/Cmd+K to jump to any navigation item. */
export const CommandPalette: React.FC<{
  onClose: () => void;
  onNavigate: (id: string) => void;
}> = ({ onClose, onNavigate }) => {
  const [query, setQuery] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);
  const all = [...NEXUS_NAV, ...LEGACY_NAV].filter((i) => i.label.toLowerCase().includes(query.toLowerCase()));

  useEffect(() => {
    inputRef.current?.focus();
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose]);

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-24 bg-black/60" onClick={onClose}>
      <div
        className="w-full max-w-lg rounded-[var(--nx-radius)] border border-slate-700 bg-slate-900 shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <input
          ref={inputRef}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Open Project · Start Agent · Configure Agent · Open Maestro…"
          className="w-full px-4 py-3 bg-transparent outline-none text-sm placeholder:text-slate-600 border-b border-slate-800"
          onKeyDown={(e) => {
            if (e.key === 'Enter' && all[0]) {
              onNavigate(all[0].id);
              onClose();
            }
          }}
        />
        <div className="p-1 max-h-72 overflow-y-auto">
          {all.map((item) => (
            <button
              key={item.id}
              onClick={() => {
                onNavigate(item.id);
                onClose();
              }}
              className="w-full text-left px-3 py-2 rounded-[var(--nx-radius-sm)] hover:bg-slate-800 text-sm flex items-center justify-between"
            >
              <span>{item.label}</span>
              <span className="text-[10px] text-slate-600">{item.section}</span>
            </button>
          ))}
        </div>
      </div>
    </div>
  );
};
