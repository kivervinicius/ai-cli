import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { nexus } from '../../nexus/api';
import { asArray } from '../../lib/safeArray';
import type { Agent, AttentionItem, ProjectAgentSummary } from '../../types';

const EXPANDED_KEY = 'nx_rail_expanded_projects';

export type ProjectOutlineAttention = {
  needsYou: AttentionItem[];
  failed: AttentionItem[];
};

function loadExpandedIds(): Set<string> {
  try {
    const raw = localStorage.getItem(EXPANDED_KEY);
    if (!raw) return new Set();
    const parsed = JSON.parse(raw) as unknown;
    return new Set(asArray<string>(parsed).filter((id) => typeof id === 'string'));
  } catch {
    return new Set();
  }
}

function persistExpandedIds(ids: Set<string>) {
  try {
    localStorage.setItem(EXPANDED_KEY, JSON.stringify([...ids]));
  } catch {
    /* ignore quota / private mode */
  }
}

export function groupAttentionByProject(attention: {
  needs_you?: AttentionItem[] | null;
  failed?: AttentionItem[] | null;
}): Map<string, ProjectOutlineAttention> {
  const map = new Map<string, ProjectOutlineAttention>();
  const ensure = (projectId: string): ProjectOutlineAttention => {
    let entry = map.get(projectId);
    if (!entry) {
      entry = { needsYou: [], failed: [] };
      map.set(projectId, entry);
    }
    return entry;
  };
  for (const item of asArray<AttentionItem>(attention.needs_you)) {
    if (!item?.project_id) continue;
    ensure(item.project_id).needsYou.push(item);
  }
  for (const item of asArray<AttentionItem>(attention.failed)) {
    if (!item?.project_id) continue;
    ensure(item.project_id).failed.push(item);
  }
  return map;
}

export function summariesByProjectId(
  summaries: ProjectAgentSummary[] | null | undefined,
): Map<string, ProjectAgentSummary> {
  const map = new Map<string, ProjectAgentSummary>();
  for (const summary of asArray<ProjectAgentSummary>(summaries)) {
    if (!summary?.project_id) continue;
    map.set(summary.project_id, summary);
  }
  return map;
}

export function useProjectRailOutline() {
  const [summaries, setSummaries] = useState<ProjectAgentSummary[]>([]);
  const [attentionByProject, setAttentionByProject] = useState<
    Map<string, ProjectOutlineAttention>
  >(() => new Map());
  const [expandedIds, setExpandedIds] = useState<Set<string>>(loadExpandedIds);
  const [agentsByProject, setAgentsByProject] = useState<Map<string, Agent[]>>(() => new Map());
  const [loadingAgents, setLoadingAgents] = useState<Set<string>>(() => new Set());
  const agentsCacheRef = useRef(agentsByProject);
  agentsCacheRef.current = agentsByProject;

  const refreshOutline = useCallback(async () => {
    const [summaryList, attention] = await Promise.all([
      nexus.listProjectSummaries().catch(() => [] as ProjectAgentSummary[]),
      nexus.getAttention().catch(() => ({
        needs_you: [],
        completed: [],
        failed: [],
        all_items: [],
        total_needs: 0,
      })),
    ]);
    setSummaries(asArray<ProjectAgentSummary>(summaryList));
    setAttentionByProject(groupAttentionByProject(attention));
  }, []);

  useEffect(() => {
    void refreshOutline();
  }, [refreshOutline]);

  const summaryMap = useMemo(() => summariesByProjectId(summaries), [summaries]);

  const setExpanded = useCallback((projectId: string, expanded: boolean) => {
    setExpandedIds((prev) => {
      const next = new Set(prev);
      if (expanded) next.add(projectId);
      else next.delete(projectId);
      persistExpandedIds(next);
      return next;
    });
  }, []);

  const toggleExpanded = useCallback(
    (projectId: string) => {
      const willExpand = !expandedIds.has(projectId);
      setExpanded(projectId, willExpand);
      return willExpand;
    },
    [expandedIds, setExpanded],
  );

  const loadAgents = useCallback(async (projectId: string) => {
    if (agentsCacheRef.current.has(projectId)) return;
    setLoadingAgents((prev) => new Set(prev).add(projectId));
    try {
      const agents = asArray<Agent>(await nexus.listAgents(projectId));
      setAgentsByProject((prev) => {
        const next = new Map(prev);
        next.set(projectId, agents);
        return next;
      });
    } catch {
      setAgentsByProject((prev) => {
        const next = new Map(prev);
        next.set(projectId, []);
        return next;
      });
    } finally {
      setLoadingAgents((prev) => {
        const next = new Set(prev);
        next.delete(projectId);
        return next;
      });
    }
  }, []);

  useEffect(() => {
    for (const projectId of expandedIds) {
      void loadAgents(projectId);
    }
  }, [expandedIds, loadAgents]);

  const invalidateAgents = useCallback((projectId: string) => {
    setAgentsByProject((prev) => {
      if (!prev.has(projectId)) return prev;
      const next = new Map(prev);
      next.delete(projectId);
      return next;
    });
  }, []);

  return {
    summaryMap,
    attentionByProject,
    expandedIds,
    agentsByProject,
    loadingAgents,
    toggleExpanded,
    setExpanded,
    loadAgents,
    invalidateAgents,
    refreshOutline,
  };
}
