package host

import (
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/kivervinicius/ai-cli/internal/control/events"
	"github.com/kivervinicius/ai-cli/internal/control/registry"
)

var (
	ansiRegex     = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\([B0]|\x1b\][0-9;]*\x07|\x1b[=>]|\r`)
	oscTitleRegex = regexp.MustCompile(`\x1b\][02];([^\x07\x1b]+)(?:\x07|\x1b\\)`)

	// Question / Input Needed Patterns
	questionPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\?\s+for\s+sh`),
		regexp.MustCompile(`(?i)\[[yY]/[nN]\]|\([yY]/[nN]\)`),
		regexp.MustCompile(`(?i)do you want to proceed\?`),
		regexp.MustCompile(`(?i)por favor,?\s*revise o plano`),
		regexp.MustCompile(`(?i)artifact\s+to\s+review`),
		regexp.MustCompile(`(?i)select an option|escolha uma op[çc][ãa]o`),
		regexp.MustCompile(`(?i)waiting for (?:user )?input|aguardando (?:resposta|confirma[çc][ãa]o)`),
		regexp.MustCompile(`(?i)are you sure\?|tem certeza\?`),
		regexp.MustCompile(`(?i)approve\?|aprovar\?`),
		regexp.MustCompile(`(?i)\?\s+([^\n\r?]{4,80}\?)`),
		regexp.MustCompile(`(?i)choice\s*\[\d+-\d+\]:`),
	}

	// Task Completed Patterns
	taskCompletedPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)task\s+(?:completed|finished|conclu[ií]da)`),
		regexp.MustCompile(`(?i)successfully\s+(?:created|updated|built|installed|completed|generated)`),
		regexp.MustCompile(`(?i)tests?\s+pass(?:ing|ed):\s*100%|100%\s+tests?\s+pass`),
		regexp.MustCompile(`(?i)all\s+\d+\s+tests?\s+pass`),
		regexp.MustCompile(`(?i)tudo pronto!|sucesso!|finalizado com sucesso`),
		regexp.MustCompile(`(?i)the command exited with code 0`),
	}

	// Working State Patterns
	workingPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)thinking\.\.\.|pensando\.\.\.`),
		regexp.MustCompile(`(?i)running command|executando comando`),
		regexp.MustCompile(`(?i)analyzing|analisando`),
		regexp.MustCompile(`(?i)generating|gerando`),
		regexp.MustCompile(`(?i)searching|pesquisando`),
	}
)

// ExtractOSCTitle finds any OSC window title sequence in the raw chunk.
func ExtractOSCTitle(raw string) string {
	matches := oscTitleRegex.FindStringSubmatch(raw)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// AttentionDetector monitors stream chunks for a runtime session and emits dynamic updates.
type AttentionDetector struct {
	mu           sync.Mutex
	runtimeID    string
	providerID   string
	profileID    string
	workspace    string
	projectName  string
	buffer       strings.Builder
	lastState    registry.RuntimeState
	lastReason   string
	lastTitle    string
	lastUpdateAt time.Time
	onAttention  func(reason, context, dynamicTitle string, state registry.RuntimeState)
}

// AcknowledgeInput starts a fresh detection window after the user answers a
// prompt. Without this reset, the old prompt remains in the sliding buffer
// and can be reported again when the provider emits unrelated output.
func (d *AttentionDetector) AcknowledgeInput() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.buffer.Reset()
	d.lastState = registry.StateRunning
	d.lastReason = ""
	d.lastTitle = ""
	d.lastUpdateAt = time.Time{}
}

// NewAttentionDetector creates an attention detector for a session.
func NewAttentionDetector(runtimeID, providerID, profileID, workspace string, onAttention func(reason, context, dynamicTitle string, state registry.RuntimeState)) *AttentionDetector {
	projName := filepath.Base(workspace)
	if projName == "" || projName == "." || projName == "/" {
		projName = "Project"
	}

	return &AttentionDetector{
		runtimeID:   runtimeID,
		providerID:  providerID,
		profileID:   profileID,
		workspace:   workspace,
		projectName: projName,
		lastState:   registry.StateRunning,
		onAttention: onAttention,
	}
}

// ProcessChunk ingests terminal output chunk, analyzes state, and triggers updates when state changes.
func (d *AttentionDetector) ProcessChunk(chunk []byte) {
	d.mu.Lock()
	defer d.mu.Unlock()

	raw := string(chunk)
	d.buffer.WriteString(raw)

	// Keep sliding buffer of last 8 KB
	if d.buffer.Len() > 8192 {
		s := d.buffer.String()
		d.buffer.Reset()
		d.buffer.WriteString(s[len(s)-4096:])
	}

	cleanText := StripANSI(d.buffer.String())
	lines := strings.Split(cleanText, "\n")
	var recentLines []string
	for i := len(lines) - 1; i >= 0 && len(recentLines) < 6; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed != "" {
			recentLines = append([]string{trimmed}, recentLines...)
		}
	}
	recentText := strings.Join(recentLines, " ")

	// 1. Check OSC title if present
	oscTitle := ExtractOSCTitle(raw)

	// 2. Classify state and extract context
	var state registry.RuntimeState = registry.StateRunning
	reason := ""
	attentionCtx := ""
	dynamicTitle := ""

	// Check Questions / Prompts
	for _, p := range questionPatterns {
		if p.MatchString(recentText) {
			state = registry.StateWaiting
			reason = "QUESTION"
			// Extract context line
			for i := len(recentLines) - 1; i >= 0; i-- {
				if p.MatchString(recentLines[i]) {
					attentionCtx = recentLines[i]
					break
				}
			}
			if attentionCtx == "" && len(recentLines) > 0 {
				attentionCtx = recentLines[len(recentLines)-1]
			}
			if len(attentionCtx) > 80 {
				attentionCtx = attentionCtx[:77] + "..."
			}
			dynamicTitle = "❓ [" + d.projectName + "] Pergunta: " + attentionCtx
			break
		}
	}

	// Check Task Completed if not in question
	if reason == "" {
		for _, p := range taskCompletedPatterns {
			if p.MatchString(recentText) {
				state = registry.StateRunning
				reason = "TASK_COMPLETED"
				for i := len(recentLines) - 1; i >= 0; i-- {
					if p.MatchString(recentLines[i]) {
						attentionCtx = recentLines[i]
						break
					}
				}
				if attentionCtx == "" && len(recentLines) > 0 {
					attentionCtx = recentLines[len(recentLines)-1]
				}
				if len(attentionCtx) > 80 {
					attentionCtx = attentionCtx[:77] + "..."
				}
				dynamicTitle = "✅ [" + d.projectName + "] Concluído: " + attentionCtx
				break
			}
		}
	}

	// Check Working if not in question or completed
	if reason == "" {
		for _, p := range workingPatterns {
			if p.MatchString(recentText) {
				state = registry.StateRunning
				reason = "WORKING"
				for i := len(recentLines) - 1; i >= 0; i-- {
					if p.MatchString(recentLines[i]) {
						attentionCtx = recentLines[i]
						break
					}
				}
				if len(attentionCtx) > 60 {
					attentionCtx = attentionCtx[:57] + "..."
				}
				dynamicTitle = "⏳ [" + d.projectName + "] " + attentionCtx
				break
			}
		}
	}

	// Default to OSC Title or Active Session title
	if dynamicTitle == "" {
		if oscTitle != "" {
			dynamicTitle = "⚡ [" + d.projectName + "] " + oscTitle
			attentionCtx = oscTitle
		} else {
			dynamicTitle = "⚡ [" + d.projectName + "] " + strings.ToUpper(d.providerID) + " (" + d.profileID + ")"
		}
	}

	// Rate-limit state updates (only on change or after debounce)
	if state == d.lastState && reason == d.lastReason && dynamicTitle == d.lastTitle && time.Since(d.lastUpdateAt) < 3*time.Second {
		return
	}

	d.lastState = state
	d.lastReason = reason
	d.lastTitle = dynamicTitle
	d.lastUpdateAt = time.Now()

	// Update Registry
	_ = registry.DefaultRegistry().UpdateAttention(d.runtimeID, state, reason, attentionCtx, d.projectName, attentionCtx, dynamicTitle)

	// Publish Event if Attention is Required or Task Completed
	if reason == "QUESTION" {
		events.DefaultBus().Publish(events.NewEvent(
			d.runtimeID,
			d.providerID,
			d.profileID,
			events.EventApprovalRequired,
			"["+d.projectName+"] Pergunta requer atenção: "+attentionCtx,
			map[string]any{
				"project_name":     d.projectName,
				"attention_reason": reason,
				"context":          attentionCtx,
				"dynamic_title":    dynamicTitle,
			},
		))
	} else if reason == "TASK_COMPLETED" {
		events.DefaultBus().Publish(events.NewEvent(
			d.runtimeID,
			d.providerID,
			d.profileID,
			events.EventToolFinished,
			"["+d.projectName+"] Tarefa completada: "+attentionCtx,
			map[string]any{
				"project_name":     d.projectName,
				"attention_reason": reason,
				"context":          attentionCtx,
				"dynamic_title":    dynamicTitle,
			},
		))
	}

	// Trigger callback
	if d.onAttention != nil {
		d.onAttention(reason, attentionCtx, dynamicTitle, state)
	}
}
