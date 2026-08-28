export interface Workspace {
  name: string;
  path: string;
  provider?: string;
  profile?: string;
  is_active: boolean;
}

export interface RuntimeSession {
  runtime_id: string;
  provider: string;
  profile: string;
  provider_session_id?: string;
  workspace: string;
  pid: number;
  host_pid: number;
  state: 'STARTING' | 'RUNNING' | 'HANDOFF' | 'STOPPED' | 'FAILED' | 'STALE';
  control_level: string;
  control_endpoint: string;
  parent_runtime_id?: string;
  handoff_type?: string;
  lineage_id?: string;
  started_at: string;
}

export interface CapabilityEvidence {
  status: 'SUPPORTED' | 'PARTIAL' | 'UNSUPPORTED' | 'UNKNOWN';
  provider_version?: string;
  mechanism?: string;
  reason?: string;
  tested: boolean;
}

export interface EffectiveCapabilities {
  process: CapabilityEvidence;
  terminal: CapabilityEvidence;
  attach: CapabilityEvidence;
  structured_events: CapabilityEvidence;
  sessions: CapabilityEvidence;
  resume: CapabilityEvidence;
  fork: CapabilityEvidence;
  submit_prompt: CapabilityEvidence;
  cancel_turn: CapabilityEvidence;
  approvals: CapabilityEvidence;
  native_ui_attach: CapabilityEvidence;
  headless: CapabilityEvidence;
  slash_control: CapabilityEvidence;
  control_level: string;
}

export interface ProviderInfo {
  id: string;
  installed: boolean;
  version: string;
  control_level: string;
  capabilities: EffectiveCapabilities;
}

export interface ProfileInfo {
  name: string;
  provider: string;
  account_email?: string;
  plan?: string;
  authenticated: boolean;
  is_default: boolean;
}

export interface EventRecord {
  id: string;
  runtime_id: string;
  provider_id: string;
  profile_id: string;
  type: string;
  summary: string;
  data: Record<string, any>;
  timestamp: string;
}
