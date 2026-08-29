import React from 'react';
import i18n from '../i18n';
import { EventRecord } from '../types';

interface EventsViewProps {
  events: EventRecord[];
}

export const EventsView: React.FC<EventsViewProps> = ({ events }) => {
  return (
    <div className="bg-slate-900 border border-slate-800 rounded-lg overflow-hidden">
      <div className="px-4 py-3 border-b border-slate-800 flex items-center justify-between">
        <h2 className="text-xs font-bold font-mono text-slate-200 uppercase tracking-wider">
          Runtime Audit Event Log
        </h2>
        <span className="text-xs font-mono text-slate-500">{events.length} events recorded</span>
      </div>

      {events.length === 0 ? (
        <div className="p-8 text-center text-slate-500 text-xs font-mono">
          No audit events recorded yet.
        </div>
      ) : (
        <div className="divide-y divide-slate-800 font-mono text-xs max-h-[600px] overflow-y-auto">
          {events.map((ev) => (
            <div key={ev.id} className="p-3 hover:bg-slate-800/40 transition">
              <div className="flex items-center justify-between text-slate-400">
                <div className="flex items-center space-x-2">
                  <span className="text-[10px] px-1.5 py-0.5 rounded bg-slate-800 text-sky-300 font-bold">
                    {ev.type}
                  </span>
                  <span className="text-slate-200 font-bold uppercase">{ev.provider_id || ev.provider || 'system'}</span>
                  <span className="text-slate-500">[{ev.runtime_id}]</span>
                </div>
                <span className="text-[10px] text-slate-500">
                  {new Date(ev.timestamp).toLocaleTimeString(i18n.language)}
                </span>
              </div>
              <div className="mt-1 text-slate-300">{ev.summary}</div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
