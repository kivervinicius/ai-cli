import { useEffect, useMemo, useState } from 'react';
import {
  Background,
  Controls,
  MiniMap,
  ReactFlow,
  applyNodeChanges,
  type Connection,
  type Edge,
  type Node,
  type OnNodesChange,
  type XYPosition,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { useTranslation } from 'react-i18next';
import { Network } from 'lucide-react';
import { executionWaves, type FlowDraftModel } from './flowModel';
import { FlowTaskNode, type FlowTaskNodeData } from './FlowTaskNode';
import styles from './FlowCanvas.module.scss';
import { asArray } from '../../lib/safeArray';

const NODE_WIDTH = 190;
const NODE_HEIGHT = 122;
const COLUMN_GAP = 90;
const ROW_GAP = 24;
type FlowNode = Node<FlowTaskNodeData, 'task'>;
const nodeTypes = { task: FlowTaskNode };

function initialPositions(flow: FlowDraftModel): Record<string, XYPosition> {
  const positions: Record<string, XYPosition> = {};
  executionWaves(flow).forEach((wave, column) =>
    wave.forEach((id, row) => {
      positions[id] = { x: column * (NODE_WIDTH + COLUMN_GAP), y: row * (NODE_HEIGHT + ROW_GAP) };
    }),
  );
  return positions;
}

export const FlowCanvas = ({
  flow,
  selectedId,
  onSelect,
  onConnect,
}: {
  flow: FlowDraftModel;
  selectedId?: string;
  onSelect: (stepId: string) => void;
  onConnect?: (from: string, to: string) => void;
}) => {
  const { t } = useTranslation();
  const [nodes, setNodes] = useState<FlowNode[]>([]);
  const [edges, setEdges] = useState<Edge[]>([]);
  const [graphError, setGraphError] = useState<string | null>(null);
  const stepById = useMemo(
    () => new Map((flow.steps || []).map((step) => [step.id, step])),
    [flow.steps],
  );

  useEffect(() => {
    try {
      const positions = initialPositions(flow);
      setNodes(
        (flow.steps || []).map((step) => ({
          id: step.id,
          type: 'task',
          position: positions[step.id] || { x: 0, y: 0 },
          data: { step, selected: false },
        })),
      );
      setEdges(
        (flow.steps || []).flatMap((step) =>
          (step.dependencies || []).map((from) => ({
            id: `${from}-${step.id}`,
            source: from,
            target: step.id,
            type: 'smoothstep',
            animated: step.status === 'EXECUTING',
          })),
        ),
      );
      setGraphError(null);
    } catch (error) {
      setGraphError(error instanceof Error ? error.message : String(error));
      setNodes([]);
      setEdges([]);
    }
  }, [flow]);

  useEffect(() => {
    setNodes((current) =>
      current.map((node) => ({
        ...node,
        data: { ...node.data, selected: node.id === selectedId },
      })),
    );
  }, [selectedId]);

  const onNodesChange: OnNodesChange<FlowNode> = (changes) => {
    setNodes((current) => applyNodeChanges(changes, current) as FlowNode[]);
  };
  const handleConnect = (connection: Connection) => {
    if (!connection.source || !connection.target || connection.source === connection.target) return;
    onConnect?.(connection.source, connection.target);
  };

  if (graphError)
    return (
      <div className={styles.invalid} role="alert">
        <Network size={18} />
        <strong>{t('flow.invalid.title', 'Invalid flow')}</strong>
        <span>{graphError}</span>
      </div>
    );

  return (
    <section className={styles.canvas} aria-label={t('flow.graph.label', 'Flow dependency graph')}>
      <div className={styles.summary} role="status">
        <span>
          {t('flow.graph.summary', '{{nodes}} nodes · {{edges}} edges', {
            nodes: nodes.length,
            edges: edges.length,
          })}
        </span>
        <span>
          {t('flow.graph.hint', 'Drag nodes to arrange. Connect handles to edit dependencies.')}
        </span>
      </div>
      <div className={styles.viewport}>
        <ReactFlow
          nodes={nodes}
          edges={edges}
          nodeTypes={nodeTypes}
          onNodesChange={onNodesChange}
          onConnect={handleConnect}
          onNodeClick={(_, node) => onSelect(node.id)}
          fitView
          fitViewOptions={{ padding: 0.2 }}
          nodesDraggable
          nodesConnectable={Boolean(onConnect)}
          deleteKeyCode={null}
          aria-label={t('flow.graph.label', 'Flow dependency graph')}
        >
          <Background gap={20} size={1} />
          <Controls showInteractive={false} />
          <MiniMap
            nodeColor={(node) =>
              node.id === selectedId ? 'var(--nx-accent)' : 'var(--nx-border-strong)'
            }
          />
        </ReactFlow>
      </div>
      <span className={styles.visuallyHidden}>
        {Array.from(stepById.values())
          .map((step) => `${step.title}: ${asArray<string>(step.dependencies).join(', ')}`)
          .join('. ')}
      </span>
    </section>
  );
};
