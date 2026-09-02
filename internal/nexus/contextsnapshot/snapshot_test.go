package contextsnapshot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildIsAllowlistedBoundedAndRedacted(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "DEV"), 0700); err != nil {
		t.Fatal(err)
	}
	secret := "OPENAI_API_KEY=sk-proj-1234567890abcdef1234567890abcdef"
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("architecture\n"+secret+"\n"+strings.Repeat("x", MaxExcerptBytes*2)), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "DEV", "CONTEXT.md"), []byte("durable context"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "private.env"), []byte("PASSWORD=do-not-read"), 0600); err != nil {
		t.Fatal(err)
	}

	envelope, err := Build(root, Metadata{ProjectID: "p1", Branch: "main", Head: "abc", MaestroVersion: "1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(envelope.Excerpts) != 2 {
		t.Fatalf("expected only allowlisted existing docs, got %#v", envelope.Excerpts)
	}
	if envelope.Bytes > MaxTotalBytes {
		t.Fatalf("context exceeded total bound: %d", envelope.Bytes)
	}
	joined := ""
	for _, excerpt := range envelope.Excerpts {
		joined += excerpt.Path + "\n" + excerpt.Content + "\n"
	}
	if strings.Contains(joined, "sk-proj-") {
		t.Fatalf("secret leaked in context: %s", joined)
	}
	if strings.Contains(joined, "private.env") || strings.Contains(joined, "do-not-read") {
		t.Fatalf("arbitrary file leaked in context: %s", joined)
	}
	if !envelope.Truncated {
		t.Fatal("expected oversized AGENTS.md to mark envelope truncated")
	}
}

func TestBuildDoesNotRecursivelyWalkSourceTree(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src", "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("read me"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "nested", "secret.ts"), []byte("const token = 'should-not-be-read'"), 0600); err != nil {
		t.Fatal(err)
	}
	envelope, err := Build(root, Metadata{})
	if err != nil {
		t.Fatal(err)
	}
	if len(envelope.Excerpts) != 1 || envelope.Excerpts[0].Path != "README.md" {
		t.Fatalf("unexpected recursive context: %#v", envelope.Excerpts)
	}
}
