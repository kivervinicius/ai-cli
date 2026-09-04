package store

import (
	"path/filepath"
	"testing"
)

func TestComposerSessionPersistsBriefTurnsSkillsAndPromptArtifacts(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "composer.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	project, err := st.CreateProject(Project{Name: "Composer", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	session, err := st.CreateComposerSession(ComposerSession{ProjectID: project.ID, State: ComposerExploring, Title: "Test strategy", BriefJSON: `{"goal":"design test strategy"}`})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.AppendComposerTurn(ComposerTurn{SessionID: session.ID, Role: ComposerUser, Content: "Help me define test coverage"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.UpsertComposerSkillProposal(ComposerSkillProposal{SessionID: session.ID, SkillID: "skill-webapp-testing", State: ComposerSkillSuggested, Reason: "Browser regression coverage"}); err != nil {
		t.Fatal(err)
	}
	artifact, err := st.CreatePromptArtifact(PromptArtifact{SessionID: session.ID, Content: "Implement the approved test strategy.", ContextJSON: `{}`, SkillIDsJSON: `["skill-webapp-testing"]`})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := st.GetComposerSession(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.BriefJSON != session.BriefJSON || loaded.ProjectID != project.ID {
		t.Fatalf("unexpected loaded session: %+v", loaded)
	}
	turns, err := st.ListComposerTurns(session.ID, 40)
	if err != nil || len(turns) != 1 || turns[0].Content != "Help me define test coverage" {
		t.Fatalf("unexpected turns: %+v, %v", turns, err)
	}
	skills, err := st.ListComposerSkillProposals(session.ID)
	if err != nil || len(skills) != 1 || skills[0].SkillID != "skill-webapp-testing" {
		t.Fatalf("unexpected skills: %+v, %v", skills, err)
	}
	if artifact.Version != 1 || artifact.Hash == "" {
		t.Fatalf("immutable artifact missing identity: %+v", artifact)
	}
}
