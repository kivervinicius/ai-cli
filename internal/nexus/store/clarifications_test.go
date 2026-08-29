package store

import (
	"path/filepath"
	"testing"
)

func TestClarificationRoundTripAndResolution(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "nexus.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	projectDir := t.TempDir()
	project, err := s.CreateProject(Project{Name: "P", CanonicalPath: projectDir})
	if err != nil {
		t.Fatal(err)
	}

	created, err := s.CreateClarification(Clarification{
		ProjectID:    project.ID,
		Goal:         "ship product",
		Status:       ClarificationPending,
		IntentJSON:   `{"intent":"ship product"}`,
		UnknownsJSON: `[{"key":"db","level":"BLOCKING"}]`,
		FactsJSON:    `{}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" {
		t.Fatal("expected id")
	}

	created.Status = ClarificationResolved
	created.FactsJSON = `{"db":"postgres"}`
	created.UnknownsJSON = `[{"key":"db","level":"BLOCKING","answer":"postgres","is_resolved":true}]`
	if err := s.UpdateClarification(*created); err != nil {
		t.Fatal(err)
	}

	got, err := s.GetClarification(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != ClarificationResolved || got.FactsJSON != `{"db":"postgres"}` {
		t.Fatalf("unexpected clarification: %+v", got)
	}
}
