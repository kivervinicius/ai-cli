import React, { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Check, ChevronDown } from 'lucide-react';

const LANGUAGES = [
  { code: 'pt-BR', label: 'Português', flag: '🇧🇷', short: 'PT' },
  { code: 'en', label: 'English', flag: '🇬🇧', short: 'EN' },
  { code: 'es', label: 'Español', flag: '🇪🇸', short: 'ES' },
] as const;

export const LanguagePicker: React.FC = () => {
  const { i18n } = useTranslation();
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const currentLang = LANGUAGES.find((l) => l.code === i18n.language) ||
    LANGUAGES.find((l) => i18n.language?.startsWith(l.code.slice(0, 2))) ||
    LANGUAGES[0];

  const handleSelect = (code: string) => {
    i18n.changeLanguage(code);
    window.localStorage.setItem('iapro:nexus:lang:v1', code);
    setOpen(false);
  };

  useEffect(() => {
    const onClickOutside = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    if (open) {
      window.addEventListener('mousedown', onClickOutside);
    }
    return () => window.removeEventListener('mousedown', onClickOutside);
  }, [open]);

  return (
    <div className="nx-lang-picker" ref={containerRef} style={{ position: 'relative' }}>
      <button
        type="button"
        className="nx-lang-btn"
        onClick={() => setOpen((prev) => !prev)}
        aria-label={`Language: ${currentLang.label}`}
        aria-expanded={open}
        title="Change Language / Mudar Idioma"
      >
        <span className="nx-lang-flag">{currentLang.flag}</span>
        <span className="nx-lang-short">{currentLang.short}</span>
        <ChevronDown size={11} style={{ opacity: 0.7 }} />
      </button>

      {open && (
        <div className="nx-lang-dropdown" role="menu">
          {LANGUAGES.map((lang) => (
            <button
              key={lang.code}
              type="button"
              className={`nx-lang-option ${lang.code === currentLang.code ? 'nx-lang-option--active' : ''}`}
              role="menuitem"
              onClick={() => handleSelect(lang.code)}
            >
              <span className="nx-lang-flag">{lang.flag}</span>
              <span className="nx-lang-name">{lang.label}</span>
              {lang.code === currentLang.code && <Check size={13} className="nx-lang-check" />}
            </button>
          ))}
        </div>
      )}
    </div>
  );
};
