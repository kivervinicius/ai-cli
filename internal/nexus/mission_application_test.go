package nexus

import (
	"context"
	"errors"
	"testing"

	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

func TestMissionApplicationServiceCRUDAndDetail(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Missions", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	service := NewMissionApplicationService(n)

	mission, err := service.Create(context.Background(), &store.Mission{ProjectID: project.ID, Name: "Ship", Goal: "Ship safely"})
	if err != nil {
		t.Fatal(err)
	}
	if mission.ID == "" || mission.Status == "" {
		t.Fatalf("mission was not persisted: %#v", mission)
	}
	if err := service.CreateTask(context.Background(), &store.MissionTask{MissionID: mission.ID, Name: "Build"}); err != nil {
		t.Fatal(err)
	}
	agent, err := st.CreateAgent(store.Agent{ProjectID: project.ID, Name: "Builder"})
	if err != nil {
		t.Fatal(err)
	}
	tasks, err := st.ListTasks(mission.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Assign(context.Background(), &store.MissionAssignment{MissionID: mission.ID, TaskID: tasks[0].ID, AgentID: agent.ID}); err != nil {
		t.Fatal(err)
	}
	detail, err := service.Detail(context.Background(), mission.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Tasks) != 1 || len(detail.Assignments) != 1 || detail.Stats.Total != 1 {
		t.Fatalf("unexpected mission detail: %#v", detail)
	}

	listed, err := service.List(context.Background(), project.ID)
	if err != nil || len(listed) != 1 {
		t.Fatalf("missions = %#v, err = %v", listed, err)
	}

	if err := service.Delete(context.Background(), mission.ID); err != nil {
		t.Fatal(err)
	}
}

func TestMissionApplicationServiceRejectsCanceledContext(t *testing.T) {
	n := openTestNexus(t)
	service := NewMissionApplicationService(n)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := service.List(ctx, "project"); !errors.Is(err, context.Canceled) {
		t.Fatalf("list error = %v, want context canceled", err)
	}
}
