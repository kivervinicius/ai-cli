package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "nexus.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestStoreOpenMigratesIdempotently(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nexus.db")
	s1, err := Open(path)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	v1, _ := s1.SchemaVersion()
	if v1 < 1 {
		t.Fatalf("expected schema version >= 1, got %d", v1)
	}
	_ = s1.Close()

	// Re-open must not re-apply migrations or fail.
	s2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer s2.Close()
	v2, _ := s2.SchemaVersion()
	if v2 != v1 {
		t.Fatalf("schema version changed on reopen: %d -> %d", v1, v2)
	}
}

func TestProjectCRUD(t *testing.T) {
	s := openTestStore(t)
	dir := t.TempDir()

	p, err := s.CreateProject(Project{Name: "Omega API", CanonicalPath: dir})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if !strings.HasPrefix(p.ID, "prj_") {
		t.Errorf("expected prj_ id prefix, got %q", p.ID)
	}
	if p.Slug != "omega-api" {
		t.Errorf("expected slug omega-api, got %q", p.Slug)
	}
	if p.MaestroMode != MaestroAssist {
		t.Errorf("default maestro mode should be ASSIST, got %q", p.MaestroMode)
	}

	got, err := s.GetProject(p.ID)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if got.CanonicalPath != dir {
		t.Errorf("canonical path mismatch: %q != %q", got.CanonicalPath, dir)
	}

	// MRU ordering
	p2dir := t.TempDir()
	p2, err := s.CreateProject(Project{Name: "Omnia", CanonicalPath: p2dir})
	if err != nil {
		t.Fatalf("create second project: %v", err)
	}
	_ = s.TouchProject(p.ID) // p becomes most recent
	list, err := s.ListProjects()
	if err != nil {
		t.Fatalf("list projects: %v", err)
	}
	if len(list) < 2 || list[0].ID != p.ID {
		t.Errorf("expected project %s first (MRU), got %+v", p.ID, list)
	}

	// Delete
	if err := s.DeleteProject(p2.ID); err != nil {
		t.Fatalf("delete project: %v", err)
	}
	if _, err := s.GetProject(p2.ID); err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestCanonicalPathValidation(t *testing.T) {
	if _, err := CanonicalPath(""); err == nil {
		t.Error("empty path must be rejected")
	}
	if _, err := CanonicalPath("/definitely/not/exists-xyz"); err == nil {
		t.Error("non-existent path must be rejected")
	}
	f := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(f, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := CanonicalPath(f); err == nil {
		t.Error("file path must be rejected (not a directory)")
	}
}

func TestAgentLifecycle(t *testing.T) {
	s := openTestStore(t)
	projDir := t.TempDir()
	proj, err := s.CreateProject(Project{Name: "Alpha", CanonicalPath: projDir})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	a, err := s.CreateAgent(Agent{ProjectID: proj.ID, Name: "Backend Developer"})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if !strings.HasPrefix(a.ID, "agt_") {
		t.Errorf("expected agt_ prefix, got %q", a.ID)
	}

	// Config revisions
	rev1, err := s.AddRevision(a.ID, `{"provider":"auto","model":"default"}`)
	if err != nil {
		t.Fatalf("add revision 1: %v", err)
	}
	rev2, err := s.AddRevision(a.ID, `{"provider":"auto","model":"deep"}`)
	if err != nil {
		t.Fatalf("add revision 2: %v", err)
	}
	if rev1.Revision != 1 || rev2.Revision != 2 {
		t.Errorf("expected revisions 1,2 got %d,%d", rev1.Revision, rev2.Revision)
	}
	revs, err := s.ListRevisions(a.ID)
	if err != nil || len(revs) != 2 {
		t.Fatalf("expected 2 revisions, got %d (%v)", len(revs), err)
	}
	if revs[0].Revision != 2 {
		t.Errorf("newest revision first, got %d", revs[0].Revision)
	}

	// Runtime generations
	g1, err := s.AddGeneration(RuntimeGeneration{
		AgentID: a.ID, RevisionID: rev1.ID, RuntimeID: "rt_abc", Provider: "codex",
		Profile: "personal", ProviderSession: "sess-1", StartedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("add generation: %v", err)
	}
	gens, err := s.ListGenerations(a.ID)
	if err != nil || len(gens) != 1 {
		t.Fatalf("expected 1 generation, got %d (%v)", len(gens), err)
	}
	cur, err := s.CurrentGeneration(a.ID)
	if err != nil || cur.ID != g1.ID {
		t.Fatalf("current generation mismatch: %v %+v", err, cur)
	}

	// Lineage
	if err := s.AddLineage(LineageEntry{
		AgentID: a.ID, Relation: "CONTEXT_HANDOFF", SourceRuntime: "rt_abc",
		SourceSession: "sess-1", TargetRuntime: "rt_def", TargetSession: "sess-9",
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("add lineage: %v", err)
	}
	lin, err := s.ListLineage(a.ID)
	if err != nil || len(lin) != 1 {
		t.Fatalf("expected 1 lineage entry, got %d (%v)", len(lin), err)
	}

	// IDOR guard: reading agent under a foreign project must fail.
	otherProj, _ := s.CreateProject(Project{Name: "Other", CanonicalPath: t.TempDir()})
	if _, err := s.GetAgent(a.ID, otherProj.ID); err != ErrNotFound {
		t.Errorf("expected ErrNotFound for cross-project agent access, got %v", err)
	}

	// Update + delete
	a.Status = "WORKING"
	if err := s.UpdateAgent(a); err != nil {
		t.Fatalf("update agent: %v", err)
	}
	if err := s.DeleteAgent(a.ID, proj.ID); err != nil {
		t.Fatalf("delete agent: %v", err)
	}
	if _, err := s.GetAgent(a.ID, proj.ID); err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestProjectLayoutPersistence(t *testing.T) {
	s := openTestStore(t)
	proj, _ := s.CreateProject(Project{Name: "L", CanonicalPath: t.TempDir()})
	if err := s.SaveLayout(proj.ID, `{"openAgents":["agt_a"],"splits":[]}`); err != nil {
		t.Fatalf("save layout: %v", err)
	}
	layout, err := s.GetLayout(proj.ID)
	if err != nil || !strings.Contains(layout, "agt_a") {
		t.Fatalf("layout roundtrip failed: %q (%v)", layout, err)
	}
}
