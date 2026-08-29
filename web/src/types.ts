export interface Workspace {
  id?: string;
  name: string;
  path: string;
  provider?: string;
  profile?: string;
  is_active?: boolean;
  created_at?: string;
  last_used_at?: string;
}

export interface RuntimeSession {
  runtime_id: string;
  title?: string;
  provider_id?: string;
  profile_id?: string;
  provider?: string;
  profile?: string;
  provider_session_id?: string;
  workspace: string;
  pid: number;
  host_pid: number;
  state: 'STARTING' | 'RUNNING' | 'HANDOFF' | 'WAITING' | 'APPROVAL' | 'DETACHED' | 'STOPPING' | 'STOPPED' | 'FAILED' | 'STALE';
  control_level: string;
  control_endpoint: string;
  parent_runtime_id?: string;
  handoff_type?: string;
  lineage_id?: string;
  started_at: string;
  attention_reason?: 'QUESTION' | 'APPROVAL' | 'TASK_COMPLETED' | 'WORKING' | 'IDLE' | 'ERROR';
  attention_context?: string;
  project_name?: string;
  last_task_summary?: string;
  dynamic_title?: string;
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

export interface ProviderAccount {
  id: string;
  provider: string;
  profile: string;
  display_name: string;
  authenticated: boolean;
  is_default: boolean;
  available: boolean;
  quota_remaining: number;
  quota_total: number;
  rate_limited: boolean;
  health: string;
  last_checked: string;
}

export interface EventRecord {
  id: string;
  runtime_id: string;
  provider_id?: string;
  profile_id?: string;
  provider?: string;
  profile?: string;
  type: string;
  summary: string;
  data: Record<string, any>;
  timestamp: string;
}

/* ---- IAPro Nexus domain types ---- */

export interface Project {
  id: string;
  name: string;
  slug: string;
  canonical_path: string;
  repo_remote: string;
  repo_url: string;
  default_branch: string;
  maestro_mode: 'OFF' | 'ASSIST' | 'ORCHESTRATE';
  resource_policy: string;
  default_isolation: string;
  settings: string;
  created_at: string;
  updated_at: string;
  last_opened_at?: string;
}

export interface Agent {
  id: string;
  project_id: string;
  name: string;
  role: string;
  status: string;
  current_revision_id?: string;
  continuity_status: string;
  created_at: string;
  updated_at: string;
  last_started_at?: string;
}

export interface RuntimeGeneration {
  id: string;
  agent_id: string;
  revision_id: string;
  runtime_id: string;
  provider: string;
  profile: string;
  provider_session: string;
  continuity: string;
  started_at: string;
  stopped_at?: string;
  state: string;
}

export interface LineageEntry {
  id: string;
  agent_id: string;
  relation: string;
  source_runtime: string;
  source_session: string;
  target_runtime: string;
  target_session: string;
  checkpoint_id: string;
  created_at: string;
}

export interface AgentRevision {
  id: string;
  agent_id: string;
  revision: number;
  config: string;
  created_at: string;
}

export interface AgentDetail {
  agent: Agent;
  generations: RuntimeGeneration[];
  lineage: LineageEntry[];
  revisions: AgentRevision[];
}

/* ---- Gate 3: Agent Configuration ---- */

export interface AgentConfig {
  provider: string;
  profile: string;
  model?: string;
  options?: Record<string, any>;
  workspace?: string;
  isolation?: string;
  maestro_mode?: string;
  continuity_policy?: string;
  environment?: Record<string, string>;
  allocation?: {
    prefer_provider?: string;
    max_concurrent?: number;
    quota_preserve?: boolean;
    cooldown_seconds?: number;
  };
}

export interface ConfigImpact {
  mode: 'LIVE_SAME_RUNTIME' | 'RESTART_RUNTIME' | 'NEW_SESSION';
  changed_fields: string[];
  requires_restart: boolean;
  requires_new_session: boolean;
  current_config?: AgentConfig;
  proposed_config?: AgentConfig;
  warnings?: string[];
}

/* ---- OS Filesystem & Deep Integration ---- */

export interface FSEntry {
  name: string;
  path: string;
  is_dir: boolean;
  is_git: boolean;
  tech: string[];
  mod_time: string;
  size_bytes: number;
  child_count?: number;
  permissions: string;
}

export interface FSBookmark {
  label: string;
  path: string;
  icon: string;
}

export interface FSBrowseResult {
  current_path: string;
  parent_path: string;
  breadcrumbs: string[];
  entries: FSEntry[];
  bookmarks: FSBookmark[];
  is_git: boolean;
  git_branch?: string;
  tech?: string[];
}

export interface FSScanResult {
  name: string;
  path: string;
  branch: string;
  tech: string[];
  mod_time: string;
  is_imported: boolean;
}

export interface FSInspectResult {
  path: string;
  exists: boolean;
  is_dir: boolean;
  is_git: boolean;
  git_branch?: string;
  git_remote?: string;
  suggested_name: string;
  tech: string[];
}

export interface GitBranchesResult {
  project_id: string;
  canonical_path: string;
  current_branch: string;
  default_branch: string;
  branches: string[];
  remote_branches: string[];
  is_clean: boolean;
  modified_count: number;
}

export interface GitCheckoutResult {
  success: boolean;
  current_branch: string;
  output?: string;
  error?: string;
}

export interface WorkPackage {
  id: string;
  title: string;
  goal: string;
  priority: 'CRITICAL' | 'HIGH' | 'NORMAL' | 'LOW';
  status: 'PENDING' | 'READY' | 'ALLOCATING' | 'COMPILING' | 'EXECUTING' | 'TESTING' | 'REVIEWING' | 'VERIFIED' | 'FAILED' | 'BLOCKED';
  dependencies: string[];
  parallel_group?: string;
  role: string;
  task_requirements?: string;
  agent_allocation?: string;
  maestro_gates?: string[];
  acceptance_criteria: string[];
  shared_artifacts?: string[];
  compiled_prompt?: string;
}

export interface PlanPhase {
  id: string;
  title: string;
  description?: string;
  order: number;
  packages: WorkPackage[];
}

export interface WorkPlan {
  id: string;
  project_id: string;
  mission_id?: string;
  title: string;
  description: string;
  status: 'DRAFT' | 'READY' | 'EXECUTING' | 'COMPLETED' | 'BLOCKED';
  current_revision: number;
  phases: PlanPhase[];
  structured_facts?: Record<string, string>;
  created_at: string;
  updated_at: string;
}

export interface PlanRevision {
  id: string;
  plan_id: string;
  revision: number;
  snapshot_json: string;
  change_summary: string;
  created_at: string;
}

export interface AutonomyContract {
  max_retries: number;
  max_total_iterations: number;
  auto_remediate: boolean;
  require_verification: boolean;
  disallow_destructive_git: boolean;
  verification_commands?: string[];
  escalate_on_failure: boolean;
}

export interface PackageRun {
  id: string;
  package_id: string;
  title: string;
  state: string;
  attempt: number;
  assigned_agent: string;
  error_message?: string;
  started_at: string;
  finished_at?: string;
}

export interface MissionRun {
  id: string;
  plan_id: string;
  project_id: string;
  state: string;
  contract: AutonomyContract;
  current_pkg_index: number;
  package_runs: PackageRun[];
  started_at: string;
  updated_at: string;
  completed_at?: string;
}

export interface ResourceCandidate {
  account: ProviderAccount;
  rank: number;
  total_score: number;
  confidence: 'LIVE' | 'CACHED' | 'ESTIMATED' | 'UNKNOWN';
  score_breakdown: Record<string, number>;
  pros: string[];
  cons: string[];
  eligible: boolean;
  rejection_reason?: string;
}

export interface RecommendationResult {
  requirements: {
    task_kind: string;
    role: string;
    current_provider?: string;
    prefer_provider?: string;
  };
  policy: string;
  recommended?: ResourceCandidate;
  candidates: ResourceCandidate[];
  explanation: string;
}
