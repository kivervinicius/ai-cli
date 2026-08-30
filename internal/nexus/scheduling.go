package nexus

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/nexus/runner"
	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

const (
	ScheduleAt            = "AT"
	ScheduleAfterRun      = "AFTER_RUN"
	ScheduleWhenResources = "WHEN_RESOURCES"
)

type missionSchedulePayload struct {
	AgentID      string                  `json:"agent_id,omitempty"`
	PlanRevision int                     `json:"plan_revision"`
	Contract     runner.AutonomyContract `json:"contract"`
}

func (n *Nexus) ScheduleMission(ctx context.Context, planID string, approvedRevision int, mode string, scheduledFor *time.Time, afterRunID, agentID string, contract runner.AutonomyContract) (*store.MissionSchedule, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	plan, err := st.GetWorkPlan(planID)
	if err != nil {
		return nil, err
	}
	if approvedRevision <= 0 {
		return nil, fmt.Errorf("approved_revision is required")
	}
	if plan.CurrentRevision != approvedRevision {
		return nil, fmt.Errorf("work plan changed after approval: approved revision %d, current revision %d; review and approve the latest plan", approvedRevision, plan.CurrentRevision)
	}
	mode = strings.ToUpper(strings.TrimSpace(mode))
	switch mode {
	case ScheduleAt:
		if scheduledFor == nil {
			return nil, fmt.Errorf("scheduled_for is required for AT schedule")
		}
		t := scheduledFor.UTC()
		scheduledFor = &t
	case ScheduleAfterRun:
		if strings.TrimSpace(afterRunID) == "" {
			return nil, fmt.Errorf("after_run_id is required for AFTER_RUN schedule")
		}
		if _, err := n.Runner().GetRun(ctx, afterRunID); err != nil {
			return nil, fmt.Errorf("dependency mission run not found: %w", err)
		}
	case ScheduleWhenResources:
	default:
		return nil, fmt.Errorf("unsupported schedule mode %q", mode)
	}
	payload, err := json.Marshal(missionSchedulePayload{AgentID: agentID, PlanRevision: approvedRevision, Contract: normalizeAutonomyContract(contract, "")})
	if err != nil {
		return nil, err
	}
	return st.CreateMissionSchedule(store.MissionSchedule{
		PlanID: plan.ID, ProjectID: plan.ProjectID, Mode: mode, ScheduledFor: scheduledFor,
		AfterRunID: afterRunID, ContractJSON: string(payload),
	})
}

func (n *Nexus) ListMissionSchedules(ctx context.Context, projectID string) ([]store.MissionSchedule, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	return st.ListMissionSchedules(projectID)
}

func (n *Nexus) CancelMissionSchedule(ctx context.Context, id string) error {
	st, err := n.OpenProject()
	if err != nil {
		return err
	}
	return st.UpdateMissionScheduleStatus(id, store.ScheduleCanceled)
}

func (n *Nexus) StartScheduleLoop() {
	n.schedulerOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(15 * time.Second)
			defer ticker.Stop()
			for {
				_ = n.processMissionSchedules(context.Background())
				<-ticker.C
			}
		}()
	})
}

func (n *Nexus) processMissionSchedules(ctx context.Context) error {
	st, err := n.OpenProject()
	if err != nil {
		return err
	}
	all, err := st.ListMissionSchedules("")
	if err != nil {
		return err
	}
	for _, schedule := range all {
		if schedule.Status == store.SchedulePending && schedule.Mode == ScheduleAfterRun && schedule.AfterRunID != "" {
			dependency, depErr := n.Runner().GetRun(ctx, schedule.AfterRunID)
			if depErr == nil && runner.IsTerminalState(dependency.State) && dependency.State != runner.StateCompletedVerified {
				_ = st.UpdateMissionScheduleStatus(schedule.ID, store.ScheduleFailed)
				continue
			}
		}
		if schedule.Status != store.ScheduleRunning || schedule.RunID == "" {
			continue
		}
		run, getErr := n.Runner().GetRun(ctx, schedule.RunID)
		if getErr != nil {
			continue
		}
		if run.State == runner.StateCompletedVerified {
			_ = st.UpdateMissionScheduleStatus(schedule.ID, store.ScheduleCompleted)
		} else if runner.IsTerminalState(run.State) {
			_ = st.UpdateMissionScheduleStatus(schedule.ID, store.ScheduleFailed)
		}
	}

	ready, err := st.ListReadyMissionSchedules(time.Now().UTC())
	if err != nil {
		return err
	}
	for _, schedule := range ready {
		if schedule.Mode == ScheduleWhenResources && !n.hasAutonomousResource() {
			continue
		}
		claimed, claimErr := st.ClaimMissionSchedule(schedule.ID)
		if claimErr != nil || !claimed {
			continue // another process/tick owns this schedule
		}
		var payload missionSchedulePayload
		if err := json.Unmarshal([]byte(schedule.ContractJSON), &payload); err != nil {
			_ = st.UpdateMissionScheduleStatus(schedule.ID, store.ScheduleFailed)
			continue
		}
		// Create durably first but do not start the worker until the schedule row
		// is bound. This prevents a losing duplicate claimant from doing work.
		currentPlan, planErr := st.GetWorkPlan(schedule.PlanID)
		if planErr != nil || payload.PlanRevision <= 0 {
			_ = st.UpdateMissionScheduleStatus(schedule.ID, store.ScheduleFailed)
			continue
		}
		run, startErr := n.startMissionRunAtRevision(ctx, currentPlan, payload.PlanRevision, payload.AgentID, payload.Contract, false)
		if startErr != nil {
			if schedule.Mode == ScheduleWhenResources {
				_ = st.UpdateMissionScheduleStatus(schedule.ID, store.SchedulePending)
			} else {
				_ = st.UpdateMissionScheduleStatus(schedule.ID, store.ScheduleFailed)
			}
			continue
		}
		if err := st.BindMissionScheduleRun(schedule.ID, run.ID); err != nil {
			_, _ = n.CancelMissionRun(ctx, run.ID, "schedule binding failed")
			_ = st.UpdateMissionScheduleStatus(schedule.ID, store.ScheduleFailed)
			continue
		}
		n.StartMissionWorker(run.ID)
	}
	return nil
}

func (n *Nexus) hasAutonomousResource() bool {
	accounts, err := n.ListResources()
	if err != nil || len(accounts) == 0 {
		return false
	}
	rec := RecommendResources(accounts, TaskRequirements{TaskKind: "coding", Role: "implementer", RequiredCapabilities: []string{"headless", "submit_prompt", "autonomous_coding"}}, PolicyBalanced)
	return rec.Recommended != nil
}
