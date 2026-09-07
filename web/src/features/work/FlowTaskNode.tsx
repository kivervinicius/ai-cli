import { Handle, Position, type Node, type NodeProps } from '@xyflow/react';
import { Bot, CircleCheck, CircleDot } from 'lucide-react';
import { Badge } from '../../design-system';
import type { FlowStepModel } from './flowModel';
import styles from './FlowCanvas.module.scss';

export type FlowTaskNodeData = { step: FlowStepModel; selected: boolean };

export const FlowTaskNode = ({ data }: NodeProps<Node<FlowTaskNodeData, 'task'>>) => {
  const { step, selected } = data;
  return (
    <article className={styles.node} data-selected={selected ? 'true' : 'false'}>
      <Handle type="target" position={Position.Left} aria-label={`Connect to ${step.title}`} />
      <div className={styles.nodeTitle}>
        {step.status === 'VERIFIED' ? <CircleCheck size={13} /> : <CircleDot size={13} />}
        <strong>{step.title || step.id}</strong>
      </div>
      <div className={styles.nodeMeta}>
        <Badge tone="default">{step.assignmentStrategy}</Badge>
        {step.parallelGroup && <Badge tone="brand">{step.parallelGroup}</Badge>}
      </div>
      <small>
        {step.dependencies.length ? `after ${step.dependencies.join(', ')}` : 'entry node'}
      </small>
      <span className={styles.nodeAgent}>
        <Bot size={11} />
        {step.agentId || step.role || 'Auto resource'}
      </span>
      <Handle type="source" position={Position.Right} aria-label={`Connect from ${step.title}`} />
    </article>
  );
};
