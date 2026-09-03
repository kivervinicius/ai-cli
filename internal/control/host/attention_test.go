package host

import (
	"sync"
	"testing"

	"github.com/kivervinicius/ai-cli/internal/control/notify"
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

	promptChunk := []byte("\r\n\x1b[33mPor favor, revise o plano. Se estiver de acordo, aprove para iniciarmos a implementação imediata.\x1b[0m\r\n? for shortcuts\r\n")
	detector.ProcessChunk(promptChunk)

	mu.Lock()
	defer mu.Unlock()

	if gotReason != "QUESTION" {
		t.Fatalf("expected reason QUESTION, got %q", gotReason)
	}
	if gotContext == "" || !contains(gotContext, "revise o plano") {
		t.Fatalf("expected context with question phrase, got %q", gotContext)
	}
	if detector.lastPrompt != PromptKindFreeText {
		t.Fatalf("expected prompt_kind free_text, got %q", detector.lastPrompt)
	}
	if gotState != registry.StateWaiting {
		t.Fatalf("expected state WAITING, got %v", gotState)
	}
	if gotTitle == "" || !contains(gotTitle, "my-project") {
		t.Fatalf("expected dynamic title with project, got %q", gotTitle)
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

func TestAttentionDetectorIgnoresIsolatedYN(t *testing.T) {
	var gotReason string
	detector := NewAttentionDetector("rt-yn-alone", "codex", "default", "/workspace/project", func(reason, _, _ string, _ registry.RuntimeState) {
		gotReason = reason
	})
	detector.ProcessChunk([]byte("\r\n[y/n]\r\n"))
	if gotReason == "QUESTION" {
		t.Fatal("isolated [y/n] must not be treated as a question")
	}
}

func TestAttentionDetectorYNWithQuestion(t *testing.T) {
	var gotReason, gotContext string
	detector := NewAttentionDetector("rt-yn-q", "codex", "default", "/workspace/project", func(reason, context, _ string, _ registry.RuntimeState) {
		gotReason = reason
		gotContext = context
	})
	detector.ProcessChunk([]byte("\r\nDeseja executar a migração? [y/N]\r\n"))
	if gotReason != "QUESTION" {
		t.Fatalf("expected QUESTION, got %q", gotReason)
	}
	if detector.lastPrompt != PromptKindYN {
		t.Fatalf("expected yn, got %q", detector.lastPrompt)
	}
	if !contains(gotContext, "Deseja executar") {
		t.Fatalf("context must include the question, got %q", gotContext)
	}
}

func TestAttentionDetectorClearsOnWorkingOutput(t *testing.T) {
	detector := NewAttentionDetector("rt-clear", "codex", "default", "/workspace/project", nil)
	detector.ProcessChunk([]byte("\r\nDeseja executar a migração? [y/N]\r\n"))
	if detector.lastReason != "QUESTION" {
		t.Fatalf("setup failed: %q", detector.lastReason)
	}
	detector.ProcessChunk([]byte("\r\nthinking...\r\n"))
	if detector.lastReason == "QUESTION" {
		t.Fatal("working output must clear QUESTION state")
	}
	if detector.lastKind != AttentionWorking {
		t.Fatalf("expected working kind, got %q", detector.lastKind)
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
	if gotTitle == "" || !contains(gotTitle, "backend-api") {
		t.Fatalf("expected dynamic title with project, got %q", gotTitle)
	}
}

func TestAttentionDetectorOSNotifyOnlyOnEvidence(t *testing.T) {
	rec := &notify.Recorder{}
	notify.SetDefault(rec)
	t.Cleanup(func() { notify.SetDefault(nil) })

	detector := NewAttentionDetector("rt-notify", "codex", "default", "/workspace/demo", nil)
	detector.ProcessChunk([]byte("\r\n? for shortcuts\r\n"))
	if len(rec.Payloads) != 0 {
		t.Fatalf("shortcut chrome must not OS-notify, got %#v", rec.Payloads)
	}

	detector.ProcessChunk([]byte("\r\nDeseja executar a migração? [y/N]\r\n"))
	if len(rec.Payloads) != 1 {
		t.Fatalf("expected one OS notify for real yn question, got %d", len(rec.Payloads))
	}

	detector.ProcessChunk([]byte("\r\nDeseja executar a migração? [y/N]\r\n"))
	if len(rec.Payloads) != 1 {
		t.Fatalf("same fingerprint must not re-notify, got %d", len(rec.Payloads))
	}
}

func TestAttentionDetectorIgnoresNumberedListAlone(t *testing.T) {
	var gotReason string
	detector := NewAttentionDetector("rt-list", "codex", "default", "/workspace/project", func(reason, _, _ string, _ registry.RuntimeState) {
		gotReason = reason
	})
	detector.ProcessChunk([]byte("\r\n1. Implement auth middleware\r\n2. Add tests\r\n"))
	if gotReason == "QUESTION" {
		t.Fatal("numbered narration must not be treated as a choice prompt")
	}
}

func TestAttentionDetectorIgnoresReplacementGarbage(t *testing.T) {
	var gotReason string
	detector := NewAttentionDetector("rt-garbage", "codex", "default", "/workspace/project", func(reason, _, _ string, _ registry.RuntimeState) {
		gotReason = reason
	})
	detector.ProcessChunk([]byte("\r\n\uFFFD\uFFFD\uFFFD\uFFFD what next?\r\n"))
	// After sanitization-like filtering, diamond-heavy lines are garbage.
	// Even with a trailing question, majority garbage must not notify.
	detector.ProcessChunk([]byte("\r\n\uFFFD\uFFFD\uFFFD\uFFFD\uFFFD\uFFFD\uFFFD\uFFFD\r\n"))
	if gotReason == "QUESTION" {
		t.Fatal("UTF-8 replacement garbage must not be treated as needs_user")
	}
}

func TestAttentionDetectorNumberedListWithAskIsChoice(t *testing.T) {
	detector := NewAttentionDetector("rt-choice-ask", "codex", "default", "/workspace/project", nil)
	detector.ProcessChunk([]byte("\r\nSelect an option:\r\n1. Continue\r\n"))
	if detector.lastReason != "QUESTION" {
		t.Fatalf("expected QUESTION for numbered choice under an ask, got %q", detector.lastReason)
	}
	if detector.lastPrompt != PromptKindChoice {
		t.Fatalf("expected choice, got %q", detector.lastPrompt)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
