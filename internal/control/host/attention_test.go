package host

import (
	"sync"
	"testing"

	"github.com/kivervinicius/ai-cli/internal/control/registry"
)

func TestAttentionDetectorQuestions(t *testing.T) {
	var mu sync.Mutex
	var gotReason, gotContext, gotTitle string
	var gotState registry.RuntimeState

	detector := NewAttentionDetector("rt-123", "agy", "default", "/workspace/my-project", func(reason, context, dynamicTitle string, state registry.RuntimeState) {
		mu.Lock()
		defer mu.Unlock()
		gotReason = reason
		gotContext = context
		gotTitle = dynamicTitle
		gotState = state
	})

	// Test question detection
	promptChunk := []byte("\r\n\x1b[33mPor favor, revise o plano. Se estiver de acordo, aprove para iniciarmos a implementação imediata.\x1b[0m\r\n? for shortcuts\r\n")
	detector.ProcessChunk(promptChunk)

	mu.Lock()
	defer mu.Unlock()

	if gotReason != "QUESTION" {
		t.Fatalf("expected reason QUESTION, got %q", gotReason)
	}
	if gotContext == "" {
		t.Fatal("expected non-empty context")
	}
	if gotState != registry.StateWaiting {
		t.Fatalf("expected state WAITING, got %v", gotState)
	}
	if gotTitle == "" || !containsAny(gotTitle, "my-project", "Pergunta") {
		t.Fatalf("expected dynamic title with project and Pergunta, got %q", gotTitle)
	}
}

func TestAttentionDetectorIgnoresShortcutHint(t *testing.T) {
	var gotReason string
	detector := NewAttentionDetector("rt-shortcuts", "codex", "default", "/workspace/project", func(reason, _, _ string, _ registry.RuntimeState) {
		gotReason = reason
	})

	detector.ProcessChunk([]byte("\r\n? for shortcuts\r\n"))
	if gotReason == "QUESTION" {
		t.Fatal("terminal shortcut hint must not be treated as an agent question")
	}
}

func TestAttentionDetectorTaskCompleted(t *testing.T) {
	var mu sync.Mutex
	var gotReason, gotContext, gotTitle string

	detector := NewAttentionDetector("rt-456", "codex", "default", "/workspace/backend-api", func(reason, context, dynamicTitle string, state registry.RuntimeState) {
		mu.Lock()
		defer mu.Unlock()
		gotReason = reason
		gotContext = context
		gotTitle = dynamicTitle
	})

	// Test task completion
	chunk := []byte("\r\n\x1b[32mTask completed: Authentication endpoints created and all 15 tests pass.\x1b[0m\r\n")
	detector.ProcessChunk(chunk)

	mu.Lock()
	defer mu.Unlock()

	if gotReason != "TASK_COMPLETED" {
		t.Fatalf("expected reason TASK_COMPLETED, got %q", gotReason)
	}
	if gotContext == "" {
		t.Fatal("expected non-empty context")
	}
	if gotTitle == "" || !containsAny(gotTitle, "backend-api", "Concluído") {
		t.Fatalf("expected dynamic title with project and Concluído, got %q", gotTitle)
	}
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if !contains(s, sub) {
			return false
		}
	}
	return true
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
