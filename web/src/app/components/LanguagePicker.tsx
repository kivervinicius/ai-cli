import React from 'react';
import { useTranslation } from 'react-i18next';
import { Check, ChevronDown } from 'lucide-react';
import * as DropdownMenu from '@radix-ui/react-dropdown-menu';

const LANGUAGES = [
  { code: 'pt-BR', label: 'Português', flag: '🇧🇷', short: 'PT' },
  { code: 'en', label: 'English', flag: '🇬🇧', short: 'EN' },
  { code: 'es', label: 'Español', flag: '🇪🇸', short: 'ES' },
] as const;

export const LanguagePicker: React.FC = () => {
  const { i18n } = useTranslation();

  const currentLang = LANGUAGES.find((l) => l.code === i18n.language) ||
    LANGUAGES.find((l) => i18n.language?.startsWith(l.code.slice(0, 2))) ||
    LANGUAGES[0];

  const handleSelect = (code: string) => {
    i18n.changeLanguage(code);
    window.localStorage.setItem('iapro:nexus:language:v1', code);
    window.localStorage.setItem('iapro:nexus:lang:v1', code);
  };

  return (
    <div className="nx-lang-picker">
      <DropdownMenu.Root>
        <DropdownMenu.Trigger asChild>
          <button
            type="button"
            className="nx-lang-btn"
            aria-label={`Language: ${currentLang.label}`}
            title="Change Language / Mudar Idioma"
          >
            <span className="nx-lang-flag">{currentLang.flag}</span>
            <span className="nx-lang-short">{currentLang.short}</span>
            <ChevronDown size={11} style={{ opacity: 0.7 }} />
          </button>
        </DropdownMenu.Trigger>

        <DropdownMenu.Portal>
          <DropdownMenu.Content className="nx-lang-dropdown" sideOffset={4} align="end">
            {LANGUAGES.map((lang) => (
              <DropdownMenu.Item
                key={lang.code}
                asChild
                onSelect={() => handleSelect(lang.code)}
              >
                <button
                  type="button"
                  className={`nx-lang-option ${lang.code === currentLang.code ? 'nx-lang-option--active' : ''}`}
                >
                  <span className="nx-lang-flag">{lang.flag}</span>
                  <span className="nx-lang-name">{lang.label}</span>
                  {lang.code === currentLang.code && <Check size={13} className="nx-lang-check" />}
                </button>
              </DropdownMenu.Item>
            ))}
          </DropdownMenu.Content>
        </DropdownMenu.Portal>
      </DropdownMenu.Root>
    </div>
  );
};
