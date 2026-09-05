import React from 'react';
import { AlertTriangle, CheckCircle2, Circle, Loader2, Radio } from 'lucide-react';
import type { RuntimeSession } from '../types';
import {
  buildAttentionRadar,
  type RadarBadge,
  type RadarRuntimeItem,
} from '../app/attentionRadarModel';
import { sanitizeAttentionText } from './attentionText';

const badgeIcon: Record<RadarBadge, React.ReactNode> = {
  needs_user: <AlertTriangle size={12} />,
  working: <Loader2 size={12} className="nx-spin-slow" />,
  completed: <CheckCircle2 size={12} />,
  error: <AlertTriangle size={12} />,
  idle: <Circle size={12} />,
};

const badgeLabel: Record<RadarBadge, string> = {
  needs_user: 'precisa de você',
  working: 'trabalhando',
  completed: 'concluído',
  error: 'erro',
  idle: 'ocioso',
};

export const GlobalAttentionRadar: React.FC<{
  runtimes: RuntimeSession[];
  currentProjectId?: string;
  onFocus: (item: RadarRuntimeItem) => void;
}> = ({ runtimes, currentProjectId, onFocus }) => {
  const groups = buildAttentionRadar(runtimes);
  const totalNeeds = groups.reduce((sum, group) => sum + group.needsUserCount, 0);

  return (
    <details className="nx-attention-radar" data-has-attention={totalNeeds > 0 ? 'true' : 'false'}>
      <summary className="nx-attention-radar__summary" title="Radar global de terminais">
        <Radio size={13} />
        <span>Radar</span>
        {totalNeeds > 0 ? (
          <span className="nx-attention-radar__count">{totalNeeds}</span>
        ) : (
          <span className="nx-attention-radar__idle">ok</span>
        )}
      </summary>
      <div className="nx-attention-radar__panel" role="list">
        {groups.length === 0 ? (
          <p className="nx-attention-radar__empty">Nenhum terminal ativo.</p>
        ) : (
          groups.map((group) => (
            <section key={group.projectId} className="nx-attention-radar__group">
              <header>
                <strong>{group.projectName}</strong>
                {group.projectId === currentProjectId ? <span>atual</span> : null}
              </header>
              <ul>
                {group.items.map((item) => {
                  const label = sanitizeAttentionText(item.context || item.title, item.title);
                  return (
                    <li key={item.runtimeId}>
                      <button
                        type="button"
                        className="nx-attention-radar__item"
                        data-badge={item.badge}
                        onClick={() => onFocus(item)}
                        title={label}
                      >
                        <span className="nx-attention-radar__badge" data-badge={item.badge}>
                          {badgeIcon[item.badge]}
                          {badgeLabel[item.badge]}
                        </span>
                        <span className="nx-attention-radar__title">{label}</span>
                      </button>
                    </li>
                  );
                })}
              </ul>
            </section>
          ))
        )}
      </div>
    </details>
  );
};
