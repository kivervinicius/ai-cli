package nexus

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

func maestroStatus(version string, available bool) func() MaestroStatus {
	return func() MaestroStatus {
		status := MaestroStatus{Available: available, Mode: MaestroAssist}
		if version != "" {
			status.Capabilities = &MaestroCapability{Version: version}
		}
		return status
	}
}

func initContextGitRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	commands := [][]string{
		{"git", "init", "-b", "main", root},
		{"git", "-C", root, "config", "user.email", "nexus-test@example.invalid"},
		{"git", "-C", root, "config", "user.name", "Nexus Test"},
	}
	for _, args := range commands {
		if out, err := exec.Command(args[0], args[1:]...).CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %v: %s", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# Durable project context\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"git", "-C", root, "add", "AGENTS.md"}, {"git", "-C", root, "commit", "-m", "context baseline"}} {
		if out, err := exec.Command(args[0], args[1:]...).CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %v: %s", args, err, out)
		}
	}
	return root
}

func TestContextReadinessStartsMissing(t *testing.T) {
	n := openTestNexus(t)
	n.maestroStatus = maestroStatus("1.0.0", true)
	st, _ := n.OpenProject()
	project, err := st.CreateProject(store.Project{Name: "Context", CanonicalPath: initContextGitRepo(t), MaestroMode: store.MaestroAssist})
	if err != nil {
		t.Fatal(err)
	}
	readiness, err := n.ObserveContextReadiness(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if readiness.State != ContextMissing {
		t.Fatalf("expected MISSING, got %s", readiness.State)
	}
}

func TestPrepareContextRequiresDurableArtifacts(t *testing.T) {
	n := openTestNexus(t)
	n.maestroStatus = maestroStatus("1.0.0", true)
	st, _ := n.OpenProject()
	project, err := st.CreateProject(store.Project{Name: "Context", CanonicalPath: t.TempDir(), MaestroMode: store.MaestroAssist})
	if err != nil {
		t.Fatal(err)
	}
	readiness, err := n.PrepareContext(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if readiness.State != ContextFailed || readiness.Error == "" {
		t.Fatalf("expected FAILED without durable context, got %+v", readiness)
	}
}

func TestPrepareContextFailsWhenConfiguredMaestroUnavailable(t *testing.T) {
	n := openTestNexus(t)
	n.maestroStatus = maestroStatus("", false)
	st, _ := n.OpenProject()
	project, err := st.CreateProject(store.Project{Name: "Context", CanonicalPath: initContextGitRepo(t), MaestroMode: store.MaestroAssist})
	if err != nil {
		t.Fatal(err)
	}
	readiness, err := n.PrepareContext(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if readiness.State != ContextFailed || readiness.MaestroAvailable {
		t.Fatalf("configured Maestro must fail closed when unavailable: %+v", readiness)
	}
}

func TestPreparedContextBecomesStaleOnSourceAndMaestroDrift(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(t *testing.T, root string, n *Nexus)
	}{
		{name: "dirty", mutate: func(t *testing.T, root string, _ *Nexus) {
			if err := os.WriteFile(filepath.Join(root, "dirty.txt"), []byte("dirty\n"), 0600); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "head", mutate: func(t *testing.T, root string, _ *Nexus) {
			if err := os.WriteFile(filepath.Join(root, "next.txt"), []byte("next\n"), 0600); err != nil {
				t.Fatal(err)
			}
			for _, args := range [][]string{{"git", "-C", root, "add", "next.txt"}, {"git", "-C", root, "commit", "-m", "next"}} {
				if out, err := exec.Command(args[0], args[1:]...).CombinedOutput(); err != nil {
					t.Fatalf("%v failed: %v: %s", args, err, out)
				}
			}
		}},
		{name: "branch", mutate: func(t *testing.T, root string, _ *Nexus) {
			if out, err := exec.Command("git", "-C", root, "checkout", "-b", "feature").CombinedOutput(); err != nil {
				t.Fatalf("branch failed: %v: %s", err, out)
			}
		}},
		{name: "maestro-version", mutate: func(_ *testing.T, _ string, n *Nexus) {
			n.maestroStatus = maestroStatus("2.0.0", true)
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			n := openTestNexus(t)
			n.maestroStatus = maestroStatus("1.0.0", true)
			st, _ := n.OpenProject()
			root := initContextGitRepo(t)
			project, err := st.CreateProject(store.Project{Name: "Context", CanonicalPath: root, MaestroMode: store.MaestroAssist})
			if err != nil {
				t.Fatal(err)
			}
			ready, err := n.PrepareContext(project.ID)
			if err != nil || ready.State != ContextReady {
				t.Fatalf("prepare context: %+v %v", ready, err)
			}
			tc.mutate(t, root, n)
			observed, err := n.ObserveContextReadiness(project.ID)
			if err != nil {
				t.Fatal(err)
			}
			if observed.State != ContextStale {
				t.Fatalf("expected STALE after %s drift, got %+v", tc.name, observed)
			}
		})
	}
}
