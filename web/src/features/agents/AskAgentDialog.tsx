import React, { useMemo, useState } from 'react';
import { MessageSquareText, Play } from 'lucide-react';
import { Button, Dialog, Textarea } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { Agent } from '../../types';
import { askActionForStatus } from './askAgentModel';

export const AskAgentDialog: React.FC<{
  agent: Agent | null;
  onClose: () => void;
  onSent?: (agent: Agent) => void | Promise<void>;
}> = ({ agent, onClose, onSent }) => {
  const [prompt, setPrompt] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const action = useMemo(() => askActionForStatus(agent?.status || 'STOPPED'), [agent?.status]);

  const submit = async () => {
    if (!agent || !prompt.trim() || busy) return;
    setBusy(true); setError('');
    try {
      await nexus.askAgent(agent.id, prompt.trim(), action.startIfNeeded);
      setPrompt('');
      await onSent?.(agent);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally { setBusy(false); }
  };

  return <Dialog open={!!agent} onClose={onClose} title={agent ? `${action.label} · ${agent.name}` : action.label} wide>
    <div className="nx-ask-agent-dialog">
      <p className="nx-muted-copy">Sends work to this persistent Agent identity. No new Agent is created.</p>
      <Textarea rows={7} value={prompt} onChange={setPrompt} placeholder="Describe what this Agent should do next…" autoFocus />
      {error && <div className="nx-inline-error">{error}</div>}
      <div className="nx-dialog-actions">
        <Button onClick={onClose}>Cancel</Button>
        <Button tone="brand" disabled={!prompt.trim() || busy} onClick={() => void submit()}>
          {action.startIfNeeded ? <Play size={14}/> : <MessageSquareText size={14}/>} {busy ? 'Sending…' : action.label}
        </Button>
      </div>
    </div>
  </Dialog>;
};
