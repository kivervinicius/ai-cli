import React, { useEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { ChevronDown, Plus, Sparkles, TerminalSquare } from 'lucide-react';
import { useTranslation } from 'react-i18next';

export type ProjectCreateMenuProps = {
  onNewAgent?: () => void;
  onNewAISession?: () => void;
  onProjectShell?: () => void;
  size?: 'sm' | 'xs';
  variant?: 'topbar' | 'compact' | 'ghost';
  label?: string;
  className?: string;
};

export const ProjectCreateMenu: React.FC<ProjectCreateMenuProps> = ({
  onNewAgent,
  onNewAISession,
  onProjectShell,
  size = 'sm',
  variant = 'topbar',
  label,
  className,
}) => {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const [menuPos, setMenuPos] = useState<{ top: number; left?: number; right?: number } | null>(
    null,
  );

  useEffect(() => {
    if (!open) {
      setMenuPos(null);
      return;
    }

    const compute = () => {
      const el = rootRef.current;
      if (!el) return;
      const rect = el.getBoundingClientRect();
      const fitsRight = rect.left + 240 < window.innerWidth;
      if (fitsRight) {
        setMenuPos({ top: rect.bottom + 4, left: Math.max(8, rect.left) });
      } else {
        setMenuPos({ top: rect.bottom + 4, right: Math.max(8, window.innerWidth - rect.right) });
      }
    };

    compute();
    const onResize = () => compute();
    const onPointerDown = (event: PointerEvent) => {
      if (!rootRef.current?.contains(event.target as Node)) {
        setOpen(false);
      }
    };
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setOpen(false);
    };

    window.addEventListener('resize', onResize);
    window.addEventListener('scroll', onResize, true);
    window.addEventListener('pointerdown', onPointerDown, true);
    window.addEventListener('keydown', onKey);

    return () => {
      window.removeEventListener('resize', onResize);
      window.removeEventListener('scroll', onResize, true);
      window.removeEventListener('pointerdown', onPointerDown, true);
      window.removeEventListener('keydown', onKey);
    };
  }, [open]);

  if (!onNewAgent && !onNewAISession && !onProjectShell) return null;

  const defaultLabel = label ?? t('overview.create', 'Criar');

  const panel =
    open && menuPos
      ? createPortal(
          <div
            className="nx-create-menu__panel"
            role="menu"
            aria-label={t('overview.create', 'Criar no Projeto')}
            style={{
              top: menuPos.top,
              ...(menuPos.left !== undefined ? { left: menuPos.left } : { right: menuPos.right }),
            }}
            onPointerDown={(e) => e.stopPropagation()}
          >
            {onNewAgent && (
              <button
                type="button"
                role="menuitem"
                className="nx-create-menu__item"
                onClick={() => {
                  setOpen(false);
                  onNewAgent();
                }}
              >
                <div className="nx-create-menu__icon nx-create-menu__icon--agent">
                  <Plus size={13} />
                </div>
                <div className="nx-create-menu__text">
                  <strong>{t('overview.newAgent', 'Novo Agente')}</strong>
                  <small>{t('overview.newAgentHint', 'Agente autônomo com workspace')}</small>
                </div>
              </button>
            )}

            {onNewAISession && (
              <button
                type="button"
                role="menuitem"
                className="nx-create-menu__item"
                onClick={() => {
                  setOpen(false);
                  onNewAISession();
                }}
              >
                <div className="nx-create-menu__icon nx-create-menu__icon--session">
                  <Sparkles size={13} />
                </div>
                <div className="nx-create-menu__text">
                  <strong>{t('overview.newAISession', 'Nova Sessão IA')}</strong>
                  <small>{t('overview.newAISessionHint', 'Prompt direto com Codex/Claude')}</small>
                </div>
              </button>
            )}

            {onProjectShell && (
              <button
                type="button"
                role="menuitem"
                className="nx-create-menu__item"
                onClick={() => {
                  setOpen(false);
                  onProjectShell();
                }}
              >
                <div className="nx-create-menu__icon nx-create-menu__icon--terminal">
                  <TerminalSquare size={13} />
                </div>
                <div className="nx-create-menu__text">
                  <strong>{t('overview.projectShell', 'Terminal do Projeto')}</strong>
                  <small>{t('overview.projectShellHint', 'Shell interativo bash/zsh')}</small>
                </div>
              </button>
            )}
          </div>,
          document.body,
        )
      : null;

  return (
    <div
      ref={rootRef}
      className={`nx-create-menu ${className ? className : ''}`}
      data-variant={variant}
      data-size={size}
    >
      <button
        type="button"
        data-testid="topbar-create-menu-btn"
        className="nx-create-menu__trigger nx-button"
        data-size={size}
        data-tone={variant === 'ghost' ? undefined : 'brand'}
        aria-expanded={open}
        aria-haspopup="menu"
        title="Criar no projeto (Agente, Sessão IA ou Terminal)"
        onClick={() => setOpen((val) => !val)}
      >
        <Plus size={size === 'xs' ? 12 : 13} />
        <span className="nx-create-menu__label">{defaultLabel}</span>
        <ChevronDown size={11} className="nx-create-menu__chevron" />
      </button>
      {panel}
    </div>
  );
};
