import React from 'react';
import { ProviderInfo } from '../types';
import { CheckCircle2, XCircle } from 'lucide-react';

interface ProvidersViewProps {
  providers: ProviderInfo[];
}

export const ProvidersView: React.FC<ProvidersViewProps> = ({ providers }) => {
  return (
    <div className="bg-slate-900 border border-slate-800 rounded-lg overflow-hidden">
      <div className="px-4 py-3 border-b border-slate-800 flex items-center justify-between">
        <h2 className="text-xs font-bold font-mono text-slate-200 uppercase tracking-wider">
          Truthful Provider Capabilities
        </h2>
        <span className="text-xs font-mono text-slate-500">Live Evidence Engine</span>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-left text-xs font-mono">
          <thead className="bg-slate-950 text-slate-400 border-b border-slate-800">
            <tr>
              <th className="px-4 py-3">Provider</th>
              <th className="px-4 py-3">Installed</th>
              <th className="px-4 py-3">Version</th>
              <th className="px-4 py-3">Control Level</th>
              <th className="px-4 py-3">Terminal</th>
              <th className="px-4 py-3">Events</th>
              <th className="px-4 py-3">Resume</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-800/80">
            {providers.map((p) => {
              return (
                <tr key={p.id} className="hover:bg-slate-800/30">
                  <td className="px-4 py-3 font-bold text-slate-200 uppercase">{p.id}</td>
                  <td className="px-4 py-3">
                    {p.installed ? (
                      <span className="inline-flex items-center text-emerald-400 font-semibold">
                        <CheckCircle2 className="w-4 h-4 mr-1" /> Yes
                      </span>
                    ) : (
                      <span className="inline-flex items-center text-slate-500">
                        <XCircle className="w-4 h-4 mr-1" /> No
                      </span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-slate-400">{p.version || '—'}</td>
                  <td className="px-4 py-3">
                    <span className="px-2 py-0.5 rounded bg-slate-800 text-sky-300 font-bold text-[11px]">
                      {p.control_level}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <span className="text-emerald-400 font-medium">
                      {p.capabilities?.terminal?.status || 'SUPPORTED'}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <span className="text-slate-400">
                      {p.capabilities?.structured_events?.status || 'UNSUPPORTED'}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <span className="text-emerald-400 font-medium">
                      {p.capabilities?.resume?.status || 'SUPPORTED'}
                    </span>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
};
