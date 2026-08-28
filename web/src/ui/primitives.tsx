import React from 'react';

/* IAPro Nexus UI primitives (§71-74). Component-first: no ad hoc UI for core features. */

type Tone = 'default' | 'brand' | 'success' | 'warning' | 'danger' | 'info';

const toneClasses: Record<Tone, string> = {
  default: 'bg-slate-800/70 text-slate-200 hover:bg-slate-700/70 border border-slate-700/60',
  brand: 'bg-indigo-600 text-white hover:bg-indigo-500 border border-indigo-500',
  success: 'bg-emerald-600 text-white hover:bg-emerald-500 border border-emerald-500',
  warning: 'bg-amber-500 text-slate-900 hover:bg-amber-400 border border-amber-500',
  danger: 'bg-red-600 text-white hover:bg-red-500 border border-red-500',
  info: 'bg-cyan-600 text-white hover:bg-cyan-500 border border-cyan-500',
};

export const Button: React.FC<{
  tone?: Tone;
  size?: 'sm' | 'md';
  disabled?: boolean;
  onClick?: () => void;
  children: React.ReactNode;
  className?: string;
  title?: string;
}> = ({ tone = 'default', size = 'md', disabled, onClick, children, className = '', title }) => (
  <button
    type="button"
    title={title}
    disabled={disabled}
    onClick={onClick}
    className={`inline-flex items-center gap-1.5 rounded-[var(--nx-radius-sm)] font-medium transition disabled:opacity-40 disabled:cursor-not-allowed ${
      size === 'sm' ? 'px-2 py-1 text-xs' : 'px-3 py-1.5 text-sm'
    } ${toneClasses[tone]} ${className}`}
  >
    {children}
  </button>
);

export const Badge: React.FC<{ tone?: Tone; children: React.ReactNode }> = ({ tone = 'default', children }) => {
  const map: Record<Tone, string> = {
    default: 'bg-slate-800 text-slate-300',
    brand: 'bg-indigo-950 text-indigo-300 border border-indigo-800/60',
    success: 'bg-emerald-950 text-emerald-300 border border-emerald-800/60',
    warning: 'bg-amber-950 text-amber-300 border border-amber-800/60',
    danger: 'bg-red-950 text-red-300 border border-red-800/60',
    info: 'bg-cyan-950 text-cyan-300 border border-cyan-800/60',
  };
  return (
    <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-medium ${map[tone]}`}>
      {children}
    </span>
  );
};

export const Card: React.FC<{ children: React.ReactNode; className?: string; onClick?: () => void }> = ({
  children,
  className = '',
  onClick,
}) => (
  <div
    onClick={onClick}
    className={`rounded-[var(--nx-radius)] border border-slate-800/80 bg-slate-900/60 p-4 ${onClick ? 'cursor-pointer hover:border-indigo-700/70 transition' : ''} ${className}`}
  >
    {children}
  </div>
);

export const Input: React.FC<{
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  onEnter?: () => void;
  mono?: boolean;
}> = ({ value, onChange, placeholder, onEnter, mono }) => (
  <input
    value={value}
    placeholder={placeholder}
    onChange={(e) => onChange(e.target.value)}
    onKeyDown={(e) => {
      if (e.key === 'Enter' && onEnter) onEnter();
    }}
    className={`w-full rounded-[var(--nx-radius-sm)] border border-slate-700/70 bg-slate-950/70 px-3 py-1.5 text-sm text-slate-100 placeholder:text-slate-600 outline-none focus:border-indigo-500 ${
      mono ? 'font-mono' : ''
    }`}
  />
);

export const Spinner: React.FC<{ label?: string }> = ({ label }) => (
  <div className="flex items-center gap-2 text-slate-400 text-sm">
    <span className="w-3.5 h-3.5 rounded-full border-2 border-slate-600 border-t-indigo-500 animate-spin" />
    {label && <span>{label}</span>}
  </div>
);

export const EmptyState: React.FC<{ title: string; hint?: string; children?: React.ReactNode }> = ({
  title,
  hint,
  children,
}) => (
  <div className="flex flex-col items-center justify-center gap-2 py-14 text-center">
    <span className="text-3xl">✦</span>
    <h3 className="text-slate-200 font-medium">{title}</h3>
    {hint && <p className="text-sm text-slate-500 max-w-sm">{hint}</p>}
    {children && <div className="mt-2">{children}</div>}
  </div>
);
