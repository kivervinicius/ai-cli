package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestContextReadinessRoundTrip(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "nexus.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	project, err := st.CreateProject(Project{Name: "P", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Nanosecond)
	_, err = st.PutContextReadiness(ContextReadinessRecord{ProjectID: project.ID, State: "READY", FingerprintHash: "abc", FingerprintJSON: `{"branch":"main"}`, MaestroVersion: "1.0.0", HydratedAt: &now})
	if err != nil {
		t.Fatal(err)
	}
	got, err := st.GetContextReadiness(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != "READY" || got.FingerprintHash != "abc" || got.MaestroVersion != "1.0.0" || got.HydratedAt == nil {
		t.Fatalf("unexpected round-trip: %+v", got)
	}
}
