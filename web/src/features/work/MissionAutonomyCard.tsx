import React from 'react';
import { ShieldCheck } from 'lucide-react';
import { Badge, Card } from '../../design-system';
import type { AutonomyContract } from '../../types';
import { linesToList, listToLines } from './missionAutonomyModel';

export const MissionAutonomyCard: React.FC<{
  value: AutonomyContract;
  onChange: (value: AutonomyContract) => void;
}> = ({ value, onChange }) => {
  const patch = <K extends keyof AutonomyContract,>(key: K, next: AutonomyContract[K]) => {
    onChange({ ...value, [key]: next });
  };

  const toggle = (key: keyof Pick<AutonomyContract,
    | 'auto_remediate'
    | 'require_verification'
    | 'disallow_destructive_git'
    | 'escalate_on_failure'
    | 'allow_tool_auto_approval'
    | 'allow_git_push'
    | 'allow_deploy'
    | 'allow_external_network'
    | 'allow_secret_access'
    | 'allow_paid_services'>, label: string, dangerous = false) => (
    <label style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 12, cursor: 'pointer' }}>
      <input
        type="checkbox"
        checked={value[key]}
        onChange={(event) => patch(key, event.target.checked)}
      />
      <span>{label}</span>
      {dangerous && value[key] ? <Badge tone="warning">explicitly allowed</Badge> : null}
    </label>
  );

  const sensitiveEnabled = [
    value.allow_git_push,
    value.allow_deploy,
    value.allow_external_network,
    value.allow_secret_access,
    value.allow_paid_services,
  ].filter(Boolean).length;

  const numberField = (
    key: 'max_retries' | 'max_total_iterations' | 'max_no_progress' | 'package_timeout_seconds',
    label: string,
    min: number,
  ) => (
    <label style={{ display: 'grid', gap: 4, fontSize: 11, color: 'var(--color-text-muted)' }}>
      <span>{label}</span>
      <input
        type="number"
        min={min}
        value={value[key]}
        onChange={(event) => patch(key, Math.max(min, Number(event.target.value) || min))}
        style={{ padding: '7px 8px', borderRadius: 6, border: '1px solid var(--color-border)', background: 'var(--color-surface)', color: 'inherit' }}
      />
    </label>
  );

  return (
    <Card style={{ padding: 16 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', gap: 10, alignItems: 'center', marginBottom: 10 }}>
        <div style={{ fontWeight: 600, display: 'flex', alignItems: 'center', gap: 6 }}>
          <ShieldCheck size={15} /> Autonomy Contract
        </div>
        <Badge tone={sensitiveEnabled > 0 ? 'warning' : 'success'}>
          {sensitiveEnabled > 0 ? `${sensitiveEnabled} sensitive grants` : 'local-first'}
        </Badge>
      </div>
      <p style={{ margin: '0 0 12px', fontSize: 11, color: 'var(--color-text-muted)' }}>
        This exact contract is snapshotted with the Mission. Push, deploy, network, secrets and paid services remain denied unless you explicitly enable them.
      </p>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8, marginBottom: 12 }}>
        {numberField('max_retries', 'Retries / package', 1)}
        {numberField('max_total_iterations', 'Mission iteration budget', 1)}
        {numberField('max_no_progress', 'No-progress limit', 1)}
        {numberField('package_timeout_seconds', 'Package timeout (seconds)', 30)}
      </div>

      <div style={{ display: 'grid', gap: 7, marginBottom: 12 }}>
        {toggle('require_verification', 'Require reproducible verification')}
        {toggle('auto_remediate', 'Automatically diagnose and remediate failures')}
        {toggle('escalate_on_failure', 'Escalate to user when bounded remediation is exhausted')}
        {toggle('disallow_destructive_git', 'Block destructive Git operations')}
        {toggle('allow_tool_auto_approval', 'Allow bounded tool auto-approval')}
      </div>

      <details style={{ marginBottom: 12 }}>
        <summary style={{ cursor: 'pointer', fontSize: 12, fontWeight: 600 }}>Sensitive permissions</summary>
        <div style={{ display: 'grid', gap: 7, paddingTop: 8 }}>
          {toggle('allow_git_push', 'Allow remote Git push', true)}
          {toggle('allow_deploy', 'Allow deployment actions', true)}
          {toggle('allow_external_network', 'Allow external-network subprocess tools', true)}
          {toggle('allow_secret_access', 'Allow secret-manager / credential tools', true)}
          {toggle('allow_paid_services', 'Allow paid/cloud service mutation tools', true)}
        </div>
      </details>

      <label style={{ display: 'grid', gap: 4, fontSize: 11, color: 'var(--color-text-muted)', marginBottom: 10 }}>
        <span>Allowed file patterns — one glob per line; empty means Mission worktree scope</span>
        <textarea
          rows={3}
          value={listToLines(value.allowed_file_patterns)}
          onChange={(event) => patch('allowed_file_patterns', linesToList(event.target.value))}
          placeholder={'src/**\ntests/**'}
          style={{ padding: 8, borderRadius: 6, border: '1px solid var(--color-border)', background: 'var(--color-surface)', color: 'inherit', resize: 'vertical' }}
        />
      </label>

      <label style={{ display: 'grid', gap: 4, fontSize: 11, color: 'var(--color-text-muted)' }}>
        <span>Verification commands — one command per line; empty enables project auto-detection</span>
        <textarea
          rows={3}
          value={listToLines(value.verification_commands)}
          onChange={(event) => patch('verification_commands', linesToList(event.target.value))}
          placeholder={'go test ./...\nnpm test'}
          disabled={!value.require_verification}
          style={{ padding: 8, borderRadius: 6, border: '1px solid var(--color-border)', background: 'var(--color-surface)', color: 'inherit', resize: 'vertical' }}
        />
      </label>
    </Card>
  );
};
