package nexus

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/kivervinicius/ai-cli/internal/control/events"
	"github.com/kivervinicius/ai-cli/internal/control/notify"
)

const (
	DefaultLowQuotaThreshold = 0.30 // 30% remaining
	DefaultQuotaStepDelta    = 0.05 // 5% minimum drop to trigger consecutive notification
)

// QuotaNotificationAction represents an emitted alert action for an account.
type QuotaNotificationAction struct {
	Provider         string    `json:"provider"`
	Profile          string    `json:"profile"`
	DisplayName      string    `json:"display_name"`
	Kind             string    `json:"kind"` // "QUOTA_LOW" | "QUOTA_EXHAUSTED"
	RemainingPercent float64   `json:"remaining_percent"`
	ResetDesc        string    `json:"reset_desc,omitempty"`
	Title            string    `json:"title"`
	Body             string    `json:"body"`
	DeliveredAt      time.Time `json:"delivered_at"`
}

// AccountQuotaState tracks the notification lifecycle per provider account.
type AccountQuotaState struct {
	LastNotifiedRatio float64
	Exhausted         bool
	LastChecked       time.Time
}

// QuotaDropMonitor tracks quota decaimento and emits step-down alerts down to 0%,
// suppressing any further notifications once 0% is reached until quota is renewed.
type QuotaDropMonitor struct {
	mu           sync.Mutex
	states       map[string]*AccountQuotaState
	notifier     notify.Notifier
	eventBus     *events.Bus
	lowThreshold float64
	stepDelta    float64
}

var (
	defaultQuotaDropMonitor     *QuotaDropMonitor
	defaultQuotaDropMonitorOnce sync.Once
)

// DefaultQuotaDropMonitor returns the global singleton monitor.
func DefaultQuotaDropMonitor() *QuotaDropMonitor {
	defaultQuotaDropMonitorOnce.Do(func() {
		defaultQuotaDropMonitor = NewQuotaDropMonitor(notify.Default(), events.DefaultBus(), DefaultLowQuotaThreshold, DefaultQuotaStepDelta)
	})
	return defaultQuotaDropMonitor
}

// NewQuotaDropMonitor creates a new monitor instance.
func NewQuotaDropMonitor(notifier notify.Notifier, bus *events.Bus, lowThreshold, stepDelta float64) *QuotaDropMonitor {
	if lowThreshold <= 0 || lowThreshold > 1 {
		lowThreshold = DefaultLowQuotaThreshold
	}
	if stepDelta <= 0 || stepDelta > 1 {
		stepDelta = DefaultQuotaStepDelta
	}
	return &QuotaDropMonitor{
		states:       make(map[string]*AccountQuotaState),
		notifier:     notifier,
		eventBus:     bus,
		lowThreshold: lowThreshold,
		stepDelta:    stepDelta,
	}
}

// Reset clears state for testing or system reload.
func (m *QuotaDropMonitor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states = make(map[string]*AccountQuotaState)
}

// CheckAccount inspects an account and emits a notification if quota dropped below thresholds.
func (m *QuotaDropMonitor) CheckAccount(acc ProviderAccount) *QuotaNotificationAction {
	// 1. Skip accounts with unknown quota or non-positive total
	if acc.QuotaView == nil || acc.QuotaView.Status == "UNKNOWN" || acc.QuotaTotal <= 0 {
		return nil
	}

	ratio := acc.QuotaRemaining / acc.QuotaTotal
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}

	key := fmt.Sprintf("%s:%s", acc.Provider, acc.Profile)
	displayName := acc.DisplayName
	if displayName == "" {
		displayName = fmt.Sprintf("%s (%s)", acc.Provider, acc.Profile)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	st, exists := m.states[key]
	if !exists {
		st = &AccountQuotaState{
			LastNotifiedRatio: 1.0,
			Exhausted:         false,
			LastChecked:       time.Now(),
		}
		m.states[key] = st
	}
	st.LastChecked = time.Now()

	// 2. Recovery / Auto-Rearm: if quota recovered above the low threshold or gained significant capacity (>15%)
	if ratio > m.lowThreshold || ratio >= st.LastNotifiedRatio+0.15 {
		st.Exhausted = false
		st.LastNotifiedRatio = ratio
		return nil
	}

	// Extract reset description if available
	resetDesc := ""
	if acc.QuotaView != nil {
		for _, g := range acc.QuotaView.ModelGroups {
			for _, w := range g.Windows {
				if w.ResetDesc != "" {
					resetDesc = w.ResetDesc
					break
				}
			}
			if resetDesc != "" {
				break
			}
		}
	}

	pct := math.Round(ratio * 100)

	// 3. Exhausted state (0% remaining or RATE_LIMITED)
	if ratio <= 0.001 || acc.RateLimited {
		// If already exhausted and notified, suppress further notifications! ("e não avisar mais")
		if st.Exhausted {
			return nil
		}

		st.Exhausted = true
		st.LastNotifiedRatio = 0.0

		action := &QuotaNotificationAction{
			Provider:         acc.Provider,
			Profile:          acc.Profile,
			DisplayName:      displayName,
			Kind:             string(events.EventQuotaExhausted),
			RemainingPercent: 0,
			ResetDesc:        resetDesc,
			Title:            fmt.Sprintf("Nexus · Quota Esgotada: %s", displayName),
			DeliveredAt:      time.Now(),
		}

		if resetDesc != "" {
			action.Body = fmt.Sprintf("A quota de %s atingiu 0%%. Reset previsto em: %s. Alterne o perfil ou aguarde a renovação.", displayName, resetDesc)
		} else {
			action.Body = fmt.Sprintf("A quota de %s atingiu 0%%. Alterne o perfil para continuar executando agentes.", displayName)
		}

		m.dispatch(action)
		return action
	}

	// 4. Low quota step-down check (below lowThreshold, e.g. <= 30%)
	if ratio <= m.lowThreshold {
		isFirstDropBelowThreshold := st.LastNotifiedRatio > m.lowThreshold
		droppedSignificant := ratio <= (st.LastNotifiedRatio - m.stepDelta)

		// Also trigger on crossing distinct milestones: 20%, 10%, 5%
		milestones := []float64{0.20, 0.10, 0.05}
		crossedMilestone := false
		for _, ms := range milestones {
			if st.LastNotifiedRatio > ms && ratio <= ms {
				crossedMilestone = true
				break
			}
		}

		if isFirstDropBelowThreshold || droppedSignificant || crossedMilestone {
			st.LastNotifiedRatio = ratio
			st.Exhausted = false

			action := &QuotaNotificationAction{
				Provider:         acc.Provider,
				Profile:          acc.Profile,
				DisplayName:      displayName,
				Kind:             string(events.EventQuotaLow),
				RemainingPercent: pct,
				ResetDesc:        resetDesc,
				Title:            fmt.Sprintf("Nexus · Quota Baixa: %s", displayName),
				DeliveredAt:      time.Now(),
			}

			if resetDesc != "" {
				action.Body = fmt.Sprintf("A quota de %s caiu para %.0f%%. Reset em: %s.", displayName, pct, resetDesc)
			} else {
				action.Body = fmt.Sprintf("A quota de %s caiu para %.0f%% restante.", displayName, pct)
			}

			m.dispatch(action)
			return action
		}
	}

	return nil
}

// CheckAccounts processes a slice of accounts and returns all triggered actions.
func (m *QuotaDropMonitor) CheckAccounts(accounts []ProviderAccount) []QuotaNotificationAction {
	var actions []QuotaNotificationAction
	for _, acc := range accounts {
		if act := m.CheckAccount(acc); act != nil {
			actions = append(actions, *act)
		}
	}
	return actions
}

// dispatch delivers the notification through the OS notifier and the internal event bus.
func (m *QuotaDropMonitor) dispatch(action *QuotaNotificationAction) {
	if m.notifier != nil {
		tag := fmt.Sprintf("quota:%s:%s:%s", action.Provider, action.Profile, action.Kind)
		_ = m.notifier.Notify(notify.Payload{
			Title: action.Title,
			Body:  action.Body,
			Tag:   tag,
		})
	}

	if m.eventBus != nil {
		eventType := events.EventQuotaLow
		if action.Kind == string(events.EventQuotaExhausted) {
			eventType = events.EventQuotaExhausted
		}
		m.eventBus.Publish(events.NewEvent(
			"", // runtimeID empty for account/system-level events
			action.Provider,
			action.Profile,
			eventType,
			action.Body,
			map[string]any{
				"provider":          action.Provider,
				"profile":           action.Profile,
				"display_name":      action.DisplayName,
				"remaining_percent": action.RemainingPercent,
				"reset_desc":        action.ResetDesc,
				"title":             action.Title,
			},
		))
	}
}
