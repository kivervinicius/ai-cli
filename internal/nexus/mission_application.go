package nexus

import (
	"context"

	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

// MissionApplicationService owns durable Mission CRUD and its planning-side
// task/assignment views. MissionRun execution remains on the runtime service.
type MissionApplicationService struct {
	nexus *Nexus
}

type MissionPatch struct {
	Name        *string
	Description *string
	Status      *string
	Goal        *string
	Scope       *string
	RiskLevel   *string
}

type MissionStats struct {
	Total     int `json:"total"`
	Pending   int `json:"pending"`
	Active    int `json:"active"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
}

type MissionDetail struct {
	Mission     *store.Mission
	Tasks       []store.MissionTask
	Assignments []store.MissionAssignment
	Stats       MissionStats
}

func NewMissionApplicationService(n *Nexus) *MissionApplicationService {
	return &MissionApplicationService{nexus: n}
}

func (s *MissionApplicationService) store(ctx context.Context) (*store.Store, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.nexus.OpenProject()
}

func (s *MissionApplicationService) List(ctx context.Context, projectID string) ([]store.Mission, error) {
	st, err := s.store(ctx)
	if err != nil {
		return nil, err
	}
	return st.ListMissions(projectID)
}

func (s *MissionApplicationService) Create(ctx context.Context, mission *store.Mission) (*store.Mission, error) {
	st, err := s.store(ctx)
	if err != nil {
		return nil, err
	}
	if err := st.CreateMission(mission); err != nil {
		return nil, err
	}
	return mission, nil
}

func (s *MissionApplicationService) Detail(ctx context.Context, missionID string) (MissionDetail, error) {
	st, err := s.store(ctx)
	if err != nil {
		return MissionDetail{}, err
	}
	mission, err := st.GetMission(missionID)
	if err != nil {
		return MissionDetail{}, err
	}
	tasks, err := st.ListTasks(missionID)
	if err != nil {
		return MissionDetail{}, err
	}
	assignments, err := st.ListAssignments(missionID)
	if err != nil {
		return MissionDetail{}, err
	}
	total, pending, active, completed, failed, err := st.MissionStats(missionID)
	if err != nil {
		return MissionDetail{}, err
	}
	return MissionDetail{
		Mission:     mission,
		Tasks:       tasks,
		Assignments: assignments,
		Stats:       MissionStats{Total: total, Pending: pending, Active: active, Completed: completed, Failed: failed},
	}, nil
}

func (s *MissionApplicationService) Update(ctx context.Context, missionID string, patch MissionPatch) (*store.Mission, error) {
	st, err := s.store(ctx)
	if err != nil {
		return nil, err
	}
	mission, err := st.GetMission(missionID)
	if err != nil {
		return nil, err
	}
	if patch.Name != nil {
		mission.Name = *patch.Name
	}
	if patch.Description != nil {
		mission.Description = *patch.Description
	}
	if patch.Status != nil {
		mission.Status = *patch.Status
	}
	if patch.Goal != nil {
		mission.Goal = *patch.Goal
	}
	if patch.Scope != nil {
		mission.Scope = *patch.Scope
	}
	if patch.RiskLevel != nil {
		mission.RiskLevel = *patch.RiskLevel
	}
	if err := st.UpdateMission(mission); err != nil {
		return nil, err
	}
	return mission, nil
}

func (s *MissionApplicationService) Delete(ctx context.Context, missionID string) error {
	st, err := s.store(ctx)
	if err != nil {
		return err
	}
	return st.DeleteMission(missionID)
}

func (s *MissionApplicationService) CreateTask(ctx context.Context, task *store.MissionTask) error {
	st, err := s.store(ctx)
	if err != nil {
		return err
	}
	return st.CreateTask(task)
}

func (s *MissionApplicationService) Assign(ctx context.Context, assignment *store.MissionAssignment) error {
	st, err := s.store(ctx)
	if err != nil {
		return err
	}
	return st.CreateAssignment(assignment)
}
