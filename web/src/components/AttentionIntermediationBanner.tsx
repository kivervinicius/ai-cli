import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { AlertCircle, CheckCircle2, HelpCircle, Send, Terminal, X } from 'lucide-react';
import { api } from '../api';
import type { RuntimeSession } from '../types';

export const AttentionIntermediationBanner: React.FC<{
  runtimes: RuntimeSession[];
  onFocusRuntime: (runtimeId: string) => void;
}> = ({ runtimes, onFocusRuntime }) => {
  const { t } = useTranslation();
  const [inputText, setInputText] = useState('');
  const [sending, setSending] = useState(false);
  const [dismissedIds, setDismissedIds] = useState<string[]>([]);

  // Find active runtimes needing attention that are not dismissed
  const attentionRuntimes = runtimes.filter((r) => {
    if (dismissedIds.includes(r.runtime_id)) return false;
    return r.attention_reason === 'QUESTION' || r.attention_reason === 'APPROVAL' || r.attention_reason === 'TASK_COMPLETED' || r.state === 'WAITING' || r.state === 'APPROVAL';
  });

  if (attentionRuntimes.length === 0) return null;

  const current = attentionRuntimes[0];
  const projName = current.project_name || t('missions.project');
  const reason = current.attention_reason || (current.state === 'APPROVAL' ? 'APPROVAL' : 'QUESTION');
  const contextText = current.attention_context || current.dynamic_title || t('attentionBanner.defaultContext');

  const handleSend = async (text: string) => {
    if (!text.trim() || sending) return;
    setSending(true);
    try {
      await api.respondRuntime(current.runtime_id, text);
      setInputText('');
      // Temporarily dismiss to avoid flashing
      setDismissedIds((prev) => [...prev, current.runtime_id]);
    } catch (err) {
      console.error('[AttentionBanner] Failed to send input:', err);
    } finally {
      setSending(false);
    }
  };

  const handleDismiss = () => {
    setDismissedIds((prev) => [...prev, current.runtime_id]);
  };

  const getReasonLabel = () => {
    switch (reason) {
      case 'QUESTION':
        return t('attentionBanner.question');
      case 'APPROVAL':
        return t('attentionBanner.approval');
      case 'TASK_COMPLETED':
        return t('attentionBanner.taskCompleted');
      default:
        return t('attentionBanner.attentionNeeded');
    }
  };

  return (
    <div className="nx-attention-banner" role="alert">
      <div className="nx-attention-banner__content">
        <div className="nx-attention-banner__icon">
          {reason === 'QUESTION' && <HelpCircle className="nx-text-amber-400" size={18} />}
          {reason === 'APPROVAL' && <AlertCircle className="nx-text-rose-400" size={18} />}
          {reason === 'TASK_COMPLETED' && <CheckCircle2 className="nx-text-emerald-400" size={18} />}
          {reason !== 'QUESTION' && reason !== 'APPROVAL' && reason !== 'TASK_COMPLETED' && <Terminal className="nx-text-sky-400" size={18} />}
        </div>

        <div className="nx-attention-banner__body">
          <div className="nx-attention-banner__meta">
            <span className="nx-badge nx-badge--brand">{projName}</span>
            <span className="nx-attention-banner__label">
              {getReasonLabel()}
            </span>
            <span className="nx-attention-banner__id">[{current.runtime_id}]</span>
          </div>
          <div className="nx-attention-banner__text" title={contextText}>
            {contextText}
          </div>
        </div>

        {/* Quick Intermediation Controls */}
        <div className="nx-attention-banner__actions">
          {reason !== 'TASK_COMPLETED' && (
            <>
              <div className="nx-attention-banner__quick-btns">
                <button
                  type="button"
                  className="nx-btn nx-btn--sm nx-btn--ghost"
                  onClick={() => handleSend('y')}
                  disabled={sending}
                >
                  {t('attentionBanner.yes')}
                </button>
                <button
                  type="button"
                  className="nx-btn nx-btn--sm nx-btn--ghost"
                  onClick={() => handleSend('n')}
                  disabled={sending}
                >
                  {t('attentionBanner.no')}
                </button>
              </div>

              <form
                className="nx-attention-banner__form"
                onSubmit={(e) => {
                  e.preventDefault();
                  void handleSend(inputText);
                }}
              >
                <input
                  type="text"
                  className="nx-input nx-input--sm"
                  placeholder={t('attentionBanner.inputPlaceholder')}
                  value={inputText}
                  onChange={(e) => setInputText(e.target.value)}
                  disabled={sending}
                />
                <button type="submit" className="nx-btn nx-btn--sm nx-btn--brand" disabled={!inputText.trim() || sending}>
                  <Send size={12} />
                </button>
              </form>
            </>
          )}

          <button
            type="button"
            className="nx-btn nx-btn--sm nx-btn--secondary"
            onClick={() => onFocusRuntime(current.runtime_id)}
          >
            <Terminal size={12} />
            <span>{t('attentionBanner.focusTerminal')}</span>
          </button>

          <button
            type="button"
            className="nx-icon-btn nx-icon-btn--sm"
            onClick={handleDismiss}
            title={t('attentionBanner.dismiss')}
          >
            <X size={14} />
          </button>
        </div>
      </div>
    </div>
  );
};
