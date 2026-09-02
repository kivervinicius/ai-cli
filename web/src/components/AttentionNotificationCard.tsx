import React, { useState } from 'react';
import { AlertCircle, HelpCircle, Send, Terminal, X, Check } from 'lucide-react';
import { api } from '../api';
import type { RuntimeSession } from '../types';
import { sanitizeAttentionText } from './attentionText';

export interface AttentionNotificationCardProps {
  runtime: RuntimeSession;
  onFocusRuntime: (runtimeId: string) => void;
  onDismiss: (runtimeId: string) => void;
}

export const AttentionNotificationCard: React.FC<AttentionNotificationCardProps> = ({
  runtime,
  onFocusRuntime,
  onDismiss,
}) => {
  const [inputText, setInputText] = useState('');
  const [sending, setSending] = useState(false);

  const projName = sanitizeAttentionText(runtime.project_name, 'Projeto');
  const reason = runtime.attention_reason || (runtime.state === 'APPROVAL' ? 'APPROVAL' : 'QUESTION');
  const rawContext = runtime.attention_context || runtime.dynamic_title;
  const sanitizedContext = sanitizeAttentionText(rawContext, 'O agente requer atenção no terminal.');
  const contextText = /[\p{L}\p{N}]/u.test(sanitizedContext) ? sanitizedContext : 'O agente requer atenção no terminal.';
  const runtimeLabel = runtime.runtime_id.length > 16 ? `${runtime.runtime_id.slice(0, 16)}…` : runtime.runtime_id;

  const isApproval = reason === 'APPROVAL';
  const isQuestion = reason === 'QUESTION';

  const handleSend = async (text: string) => {
    if (!text.trim() || sending) return;
    setSending(true);
    try {
      await api.respondRuntime(runtime.runtime_id, text);
      setInputText('');
      onDismiss(runtime.runtime_id);
    } catch (err) {
      console.error('[AttentionNotification] Falha ao enviar resposta:', err);
    } finally {
      setSending(false);
    }
  };

  return (
    <article
      className="nx-uui-notification"
      role="alertdialog"
      aria-live="assertive"
      aria-label={`${isApproval ? 'Aprovação Requerida' : 'Pergunta do Agente'} em ${projName}`}
    >
      <div className="nx-uui-notification__container">
        {/* Left Icon with Untitled UI colored circular background */}
        <div className={`nx-uui-notification__icon-wrap nx-uui-notification__icon-wrap--${isApproval ? 'rose' : isQuestion ? 'amber' : 'brand'}`}>
          {isApproval && <AlertCircle className="nx-uui-notification__icon text-rose-400" size={20} />}
          {isQuestion && <HelpCircle className="nx-uui-notification__icon text-amber-400" size={20} />}
          {!isApproval && !isQuestion && <Terminal className="nx-uui-notification__icon text-indigo-400" size={20} />}
        </div>

        {/* Center Content */}
        <div className="nx-uui-notification__content">
          {/* Header Row: Badge, Title & Close */}
          <div className="nx-uui-notification__header">
            <div className="nx-uui-notification__badges">
              <span className="nx-uui-badge nx-uui-badge--brand">{projName}</span>
              <span className={`nx-uui-badge nx-uui-badge--${isApproval ? 'rose' : 'amber'}`}>
                {isApproval ? 'Aprovação Requerida' : isQuestion ? 'Pergunta do Agente' : 'Atenção'}
              </span>
              <span className="nx-uui-notification__id" title={runtime.runtime_id}>
                {runtimeLabel}
              </span>
            </div>
            <button
              type="button"
              className="nx-uui-notification__close"
              onClick={() => onDismiss(runtime.runtime_id)}
              aria-label="Dispensar notificação"
              title="Dispensar"
            >
              <X size={15} />
            </button>
          </div>

          {/* Message / Question text */}
          <p className="nx-uui-notification__text">
            {contextText}
          </p>

          {/* Quick Action Buttons & Input Form in Untitled UI style */}
          <div className="nx-uui-notification__footer">
            <div className="nx-uui-notification__quick-actions">
              <button
                type="button"
                className="nx-uui-btn nx-uui-btn--primary"
                onClick={() => handleSend('y')}
                disabled={sending}
              >
                <Check size={13} />
                <span>Sim (y)</span>
              </button>
              <button
                type="button"
                className="nx-uui-btn nx-uui-btn--secondary"
                onClick={() => handleSend('n')}
                disabled={sending}
              >
                <span>Não (n)</span>
              </button>
              <button
                type="button"
                className="nx-uui-btn nx-uui-btn--tertiary"
                onClick={() => onFocusRuntime(runtime.runtime_id)}
              >
                <Terminal size={13} />
                <span>Ver Terminal</span>
              </button>
            </div>

            {/* Response Input */}
            <form
              className="nx-uui-notification__form"
              onSubmit={(e) => {
                e.preventDefault();
                void handleSend(inputText);
              }}
            >
              <input
                type="text"
                className="nx-uui-input"
                placeholder="Digitar resposta personalizada..."
                value={inputText}
                onChange={(e) => setInputText(e.target.value)}
                disabled={sending}
              />
              <button
                type="submit"
                className="nx-uui-btn nx-uui-btn--icon-submit"
                disabled={!inputText.trim() || sending}
                aria-label="Enviar resposta"
                title="Enviar resposta"
              >
                <Send size={13} />
              </button>
            </form>
          </div>
        </div>
      </div>
    </article>
  );
};
