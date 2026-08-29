import React, { useEffect, useId, useRef } from 'react';
import { AlertCircle, CheckCircle2, Info, LoaderCircle, Search, TriangleAlert, X } from 'lucide-react';
import { useTranslation } from 'react-i18next';

export type Tone = 'default' | 'brand' | 'success' | 'warning' | 'danger' | 'info' | 'ghost';

export const Button: React.FC<React.ButtonHTMLAttributes<HTMLButtonElement> & {
  tone?: Tone;
  size?: 'sm' | 'md';
}> = ({ tone = 'default', size = 'md', className = '', type = 'button', children, ...props }) => (
  <button type={type} data-tone={tone} data-size={size} className={`nx-button ${className}`.trim()} {...props}>{children}</button>
);

export const IconButton: React.FC<React.ButtonHTMLAttributes<HTMLButtonElement> & { label: string; tone?: Tone }> = ({ label, tone = 'ghost', className = '', children, ...props }) => (
  <button type="button" aria-label={label} title={props.title || label} data-tone={tone} className={`nx-icon-button ${className}`.trim()} {...props}>{children}</button>
);

export const Badge: React.FC<{ tone?: Exclude<Tone, 'ghost'>; children: React.ReactNode; className?: string }> = ({ tone = 'default', children, className = '' }) => (
  <span data-tone={tone} className={`nx-badge ${className}`.trim()}>{children}</span>
);

export const Card: React.FC<React.HTMLAttributes<HTMLDivElement> & { interactive?: boolean }> = ({ interactive, className = '', children, ...props }) => (
  <div data-interactive={interactive ? 'true' : undefined} className={`nx-card ${className}`.trim()} {...props}>{children}</div>
);

export const Input: React.FC<Omit<React.InputHTMLAttributes<HTMLInputElement>, 'onChange'> & { value: string; onChange: (value: string) => void; onEnter?: () => void; mono?: boolean }> = ({ value, onChange, onEnter, mono, className = '', ...props }) => (
  <input data-mono={mono ? 'true' : undefined} className={`nx-input ${className}`.trim()} value={value} onChange={(event) => onChange(event.target.value)} onKeyDown={(event) => { if (event.key === 'Enter') onEnter?.(); }} {...props} />
);

export const Textarea: React.FC<Omit<React.TextareaHTMLAttributes<HTMLTextAreaElement>, 'onChange'> & { value: string; onChange: (value: string) => void }> = ({ value, onChange, className = '', ...props }) => (
  <textarea className={`nx-textarea ${className}`.trim()} value={value} onChange={(event) => onChange(event.target.value)} {...props} />
);

export const Select: React.FC<{ value: string; onChange: (value: string) => void; options: { value: string; label: string }[]; label?: string; className?: string; placeholder?: string }> = ({ value, onChange, options, label, className = '', placeholder }) => (
  <label className={`nx-select-wrap ${className}`.trim()}>{label && <span>{label}</span>}<select className="nx-select" value={value} onChange={(event) => onChange(event.target.value)}>{placeholder && !options.some((option) => option.value === '') && <option value="">{placeholder}</option>}{options.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}</select></label>
);

export const Switch: React.FC<{ checked: boolean; onChange: (checked: boolean) => void; label: string; description?: string }> = ({ checked, onChange, label, description }) => (
  <label className="nx-switch-row"><button type="button" role="switch" aria-checked={checked} className="nx-switch" data-checked={checked ? 'true' : 'false'} onClick={() => onChange(!checked)}><span /></button><span className="nx-switch-copy"><strong>{label}</strong>{description && <small>{description}</small>}</span></label>
);

export const Segmented: React.FC<{ value: string; onChange: (value: string) => void; options: { value: string; label: string }[]; ariaLabel: string }> = ({ value, onChange, options, ariaLabel }) => (
  <div className="nx-segmented" role="radiogroup" aria-label={ariaLabel}>{options.map((option) => <button type="button" role="radio" aria-checked={value === option.value} data-active={value === option.value ? 'true' : 'false'} className="nx-segmented__item" key={option.value} onClick={() => onChange(option.value)}>{option.label}</button>)}</div>
);

export const Progress: React.FC<{ value: number; label: string }> = ({ value, label }) => <div className="nx-progress" role="progressbar" aria-label={label} aria-valuemin={0} aria-valuemax={100} aria-valuenow={Math.round(value)}><span style={{ width: `${Math.min(100, Math.max(0, value))}%` }} /><small>{label}</small></div>;

export const Spinner: React.FC<{ label?: string }> = ({ label }) => { const { t } = useTranslation(); return <div className="nx-spinner" role="status"><LoaderCircle size={16} aria-hidden="true" /><span>{label ?? t('common.loading')}</span></div>; };

export const EmptyState: React.FC<{ icon?: React.ReactNode; title: string; hint?: string; action?: React.ReactNode }> = ({ icon, title, hint, action }) => <div className="nx-empty-state">{icon}<strong>{title}</strong>{hint && <p>{hint}</p>}{action}</div>;

export const InlineAlert: React.FC<{ tone?: 'info' | 'success' | 'warning' | 'danger'; title?: string; children: React.ReactNode }> = ({ tone = 'info', title, children }) => {
  const Icon = tone === 'success' ? CheckCircle2 : tone === 'warning' ? TriangleAlert : tone === 'danger' ? AlertCircle : Info;
  return <div className="nx-inline-alert" data-tone={tone} role={tone === 'danger' ? 'alert' : 'status'}><Icon size={15} aria-hidden="true" /><div>{title && <strong>{title}</strong>}<span>{children}</span></div></div>;
};

export const Dialog: React.FC<{ open: boolean; title: string; onClose: () => void; children: React.ReactNode; wide?: boolean }> = ({ open, title, onClose, children, wide }) => {
  const { t } = useTranslation();
  const titleId = useId();
  const panelRef = useRef<HTMLDivElement>(null);
  const previousFocus = useRef<HTMLElement | null>(null);
  useEffect(() => {
    if (!open) return;
    previousFocus.current = document.activeElement as HTMLElement | null;
    const panel = panelRef.current;
    panel?.focus();
    const onKey = (event: KeyboardEvent) => { if (event.key === 'Escape') onClose(); };
    window.addEventListener('keydown', onKey);
    return () => { window.removeEventListener('keydown', onKey); previousFocus.current?.focus?.(); };
  }, [open, onClose]);
  if (!open) return null;
  return <div className="nx-dialog-backdrop" onMouseDown={(event) => { if (event.target === event.currentTarget) onClose(); }}><div ref={panelRef} tabIndex={-1} role="dialog" aria-modal="true" aria-labelledby={titleId} className="nx-dialog" data-wide={wide ? 'true' : 'false'}><header><h2 id={titleId}>{title}</h2><IconButton label={t('common.closeDialog')} onClick={onClose}><X size={15} /></IconButton></header><div className="nx-dialog__body">{children}</div></div></div>;
};

export const SearchInput: React.FC<{ value: string; onChange: (value: string) => void; placeholder?: string; autoFocus?: boolean }> = ({ value, onChange, placeholder, autoFocus }) => { const { t } = useTranslation(); return <label className="nx-search-input"><Search size={15} aria-hidden="true" /><input value={value} onChange={(e) => onChange(e.target.value)} placeholder={placeholder ?? t('common.search')} autoFocus={autoFocus} /></label>; };
