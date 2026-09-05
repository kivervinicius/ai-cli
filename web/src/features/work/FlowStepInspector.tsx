import React from 'react';
import { Diff, Rows3, Scissors, Trash2, WandSparkles } from 'lucide-react';
import { Badge, Button, Card, Input, Select, Textarea } from '../../design-system';
import type { Agent } from '../../types';
import type { FlowDraftModel, FlowStepModel } from './flowModel';

const lines = (value: string) =>
  value
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean);
const joined = (value: string[] | undefined) => (value || []).join('\n');

export const FlowStepInspector: React.FC<{
  flow: FlowDraftModel;
  step: FlowStepModel | null;
  agents: Agent[];
  onChange: (patch: Partial<FlowStepModel>) => void;
  onAction: (action: 'REFINE' | 'EXPAND' | 'COMPARE' | 'SPLIT' | 'REMOVE') => void;
}> = ({ flow, step, agents, onChange, onAction }) => {
  if (!step)
    return (
      <Card className="nx-flow-inspector nx-flow-inspector--empty">
        <strong>Select a Flow Step</strong>
        <p>Assignment, dependencies, context and verification are edited here.</p>
      </Card>
    );
  const candidates = flow.steps.filter((candidate) => candidate.id !== step.id);
  const deps = step.dependencies || [];
  return (
    <Card className="nx-flow-inspector">
      <div className="nx-flow-inspector__header">
        <div>
          <span className="nx-eyebrow">STEP INSPECTOR</span>
          <h3>{step.title}</h3>
        </div>
        <Badge tone="brand">{step.assignmentStrategy}</Badge>
      </div>
      <label>
        <span>Title</span>
        <Input value={step.title} onChange={(title) => onChange({ title })} />
      </label>
      <label>
        <span>Goal</span>
        <Textarea rows={4} value={step.goal} onChange={(goal) => onChange({ goal })} />
      </label>
      <div className="nx-flow-inspector__grid">
        <Select
          label="Assignment"
          value={step.assignmentStrategy}
          onChange={(value) =>
            onChange({
              assignmentStrategy: value as FlowStepModel['assignmentStrategy'],
              agentId: value === 'EXISTING' ? step.agentId : undefined,
            })
          }
          options={[
            { value: 'EXISTING', label: 'Existing Agent' },
            { value: 'CREATE', label: 'Create specialist' },
            { value: 'AUTO', label: 'Auto / Scheduler' },
          ]}
        />
        {step.assignmentStrategy === 'EXISTING' ? (
          <Select
            label="Agent"
            value={step.agentId || ''}
            onChange={(agentId) => onChange({ agentId })}
            placeholder="Select Agent"
            options={agents.map((agent) => ({
              value: agent.id,
              label: `${agent.name} · ${agent.status}`,
            }))}
          />
        ) : (
          <label>
            <span>Role</span>
            <Input value={step.role} onChange={(role) => onChange({ role })} />
          </label>
        )}
        <Select
          label="Resource policy"
          value={step.resourcePolicy || 'BALANCED'}
          onChange={(resourcePolicy) => onChange({ resourcePolicy })}
          options={[
            { value: 'BALANCED', label: 'Balanced' },
            { value: 'PRESERVE_QUOTA', label: 'Preserve quota' },
            { value: 'PREFER_PROVIDER', label: 'Prefer provider' },
            { value: 'MANUAL', label: 'Manual restriction' },
          ]}
        />
        <label>
          <span>Parallel group</span>
          <Input
            value={step.parallelGroup || ''}
            onChange={(parallelGroup) => onChange({ parallelGroup: parallelGroup || undefined })}
          />
        </label>
        <label>
          <span>Provider restriction</span>
          <Input
            value={step.provider || ''}
            onChange={(provider) => onChange({ provider: provider || undefined })}
            placeholder="optional"
          />
        </label>
        <label>
          <span>Profile restriction</span>
          <Input
            value={step.profile || ''}
            onChange={(profile) => onChange({ profile: profile || undefined })}
            placeholder="optional"
          />
        </label>
      </div>
      <div className="nx-flow-inspector__section">
        <strong>Dependencies</strong>
        <div className="nx-flow-dependencies">
          {candidates.map((candidate) => {
            const checked = deps.includes(candidate.id);
            return (
              <label key={candidate.id}>
                <input
                  type="checkbox"
                  checked={checked}
                  onChange={() =>
                    onChange({
                      dependencies: checked
                        ? deps.filter((id) => id !== candidate.id)
                        : [...deps, candidate.id],
                    })
                  }
                />
                <span>{candidate.title}</span>
              </label>
            );
          })}
        </div>
      </div>
      <label>
        <span>Acceptance criteria · one per line</span>
        <Textarea
          rows={4}
          value={joined(step.acceptanceCriteria)}
          onChange={(value) => onChange({ acceptanceCriteria: lines(value) })}
        />
      </label>
      <label>
        <span>Relevant paths · one per line</span>
        <Textarea
          rows={3}
          value={joined(step.relevantPaths)}
          onChange={(value) => onChange({ relevantPaths: lines(value) })}
        />
      </label>
      <label>
        <span>Maestro skills · real catalog IDs only</span>
        <Textarea
          rows={3}
          value={joined(step.maestroSkills)}
          onChange={(value) => onChange({ maestroSkills: lines(value) })}
        />
      </label>
      <label>
        <span>Verification requirements · one per line</span>
        <Textarea
          rows={3}
          value={joined(step.verificationRequirements)}
          onChange={(value) => onChange({ verificationRequirements: lines(value) })}
        />
      </label>
      <div className="nx-flow-inspector__actions">
        <Button size="sm" onClick={() => onAction('REFINE')}>
          <WandSparkles size={12} />
          Refine locally
        </Button>
        <Button size="sm" onClick={() => onAction('EXPAND')}>
          <Rows3 size={12} />
          Expand
        </Button>
        <Button size="sm" onClick={() => onAction('COMPARE')}>
          <Diff size={12} />
          Compare
        </Button>
        <Button size="sm" onClick={() => onAction('SPLIT')}>
          <Scissors size={12} />
          Split
        </Button>
        <Button size="sm" tone="danger" onClick={() => onAction('REMOVE')}>
          <Trash2 size={12} />
          Remove
        </Button>
      </div>
    </Card>
  );
};
