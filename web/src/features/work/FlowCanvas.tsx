import React, { useMemo } from 'react';
import { ArrowRight, Bot, CircleCheck, CircleDot, Network } from 'lucide-react';
import { Badge } from '../../design-system';
import { executionWaves, type FlowDraftModel } from './flowModel';

export const FlowCanvas: React.FC<{ flow: FlowDraftModel; selectedId?: string; onSelect: (stepId: string) => void }> = ({ flow, selectedId, onSelect }) => {
  const graph = useMemo(() => {
    try { return { waves: executionWaves(flow), error: '' }; }
    catch (error) { return { waves: [] as string[][], error: error instanceof Error ? error.message : String(error) }; }
  }, [flow]);
  const byId = useMemo(() => new Map(flow.steps.map((step) => [step.id, step])), [flow.steps]);
  if (graph.error) return <div className="nx-flow-canvas nx-flow-canvas--invalid"><Network size={18}/><strong>Invalid Flow</strong><span>{graph.error}</span></div>;
  return <div className="nx-flow-canvas" aria-label="Flow dependency graph">
    {graph.waves.map((wave, waveIndex) => <React.Fragment key={`wave-${waveIndex}`}>
      <section className="nx-flow-wave">
        <div className="nx-flow-wave__label">Wave {waveIndex + 1}</div>
        <div className="nx-flow-wave__nodes">
          {wave.map((id) => { const step=byId.get(id)!; return <button type="button" key={id} className="nx-flow-node" data-selected={selectedId===id?'true':'false'} onClick={()=>onSelect(id)}>
            <div className="nx-flow-node__title">{step.status==='VERIFIED'?<CircleCheck size={13}/>:<CircleDot size={13}/>}<strong>{step.title || id}</strong></div>
            <div className="nx-flow-node__meta"><Badge tone="default">{step.assignmentStrategy}</Badge>{step.parallelGroup&&<Badge tone="brand">{step.parallelGroup}</Badge>}</div>
            <small>{step.dependencies.length ? `after ${step.dependencies.join(', ')}` : 'entry step'}</small>
            <span className="nx-flow-node__agent"><Bot size={11}/>{step.agentId || step.role || 'Auto resource'}</span>
          </button>; })}
        </div>
      </section>
      {waveIndex<graph.waves.length-1&&<div className="nx-flow-wave-arrow"><ArrowRight size={18}/></div>}
    </React.Fragment>)}
  </div>;
};
