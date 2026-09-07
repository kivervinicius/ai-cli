package nexus

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kivervinicius/ai-cli/internal/control/events"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/quota"
	"github.com/kivervinicius/ai-cli/internal/profile"
)

const quotaMonitorInterval = time.Minute

// QuotaMonitorService keeps quota alerts independent from any UI surface.
// A single service belongs to one Nexus process and is stopped with its context.
type QuotaMonitorService struct {
	monitor  *QuotaDropMonitor
	bus      *events.Bus
	mu       sync.Mutex
	ctx      context.Context
	cancel   context.CancelFunc
	started  bool
	wg       sync.WaitGroup
	degraded map[string]bool
}

func NewQuotaMonitorService(monitor *QuotaDropMonitor, bus *events.Bus) *QuotaMonitorService {
	if monitor == nil {
		monitor = DefaultQuotaDropMonitor()
	}
	if bus == nil {
		bus = events.DefaultBus()
	}
	return &QuotaMonitorService{monitor: monitor, bus: bus, degraded: make(map[string]bool)}
}

// Start is idempotent. The first pass runs immediately, then the ticker wakes
// once per minute; it never polls in a tight loop.
func (s *QuotaMonitorService) Start(ctx context.Context) {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return
	}
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.started = true
	s.wg.Add(1)
	s.mu.Unlock()
	go s.loop()
}

func (s *QuotaMonitorService) loop() {
	defer s.wg.Done()
	s.check()
	ticker := time.NewTicker(quotaMonitorInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.check()
		}
	}
}

func (s *QuotaMonitorService) Stop() {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return
	}
	s.cancel()
	s.started = false
	s.mu.Unlock()
	s.wg.Wait()
}

func (s *QuotaMonitorService) check() {
	profiles, err := profile.List()
	if err != nil {
		s.emitDegraded("profiles", "unable to enumerate profiles: "+err.Error())
		return
	}
	for _, p := range profiles {
		if p.Disabled {
			continue
		}
		account := profile.GetAccountInfo(p.Provider, p.Name)
		if !account.Authenticated {
			continue
		}
		snapshot := profile.GetUsageSnapshot(p.Provider, p.Name)
		if snapshot.Status != model.UsageLive && snapshot.Status != model.UsageCached {
			s.emitDegraded(p.Provider+":"+p.Name, fmt.Sprintf("quota read is %s", snapshot.Status))
			continue
		}
		s.clearDegraded(p.Provider + ":" + p.Name)
		view := quota.BuildQuotaView(snapshot, account.Email, account.Plan)
		acc := ProviderAccount{Provider: p.Provider, Profile: p.Name, DisplayName: account.Email, Authenticated: true, QuotaView: &view, QuotaRemaining: 0, QuotaTotal: 1, LastChecked: time.Now()}
		if remaining, ok := view.BestGroupRemaining(); ok {
			acc.QuotaRemaining = remaining / 100
		}
		s.monitor.CheckAccount(acc)
	}
}

func (s *QuotaMonitorService) emitDegraded(key, reason string) {
	s.mu.Lock()
	if s.degraded[key] {
		s.mu.Unlock()
		return
	}
	s.degraded[key] = true
	s.mu.Unlock()
	s.bus.Publish(events.NewEvent("", "", "", events.EventQuotaMonitorDegraded, reason, map[string]any{"reason": reason, "profile_key": key}))
}

func (s *QuotaMonitorService) clearDegraded(key string) {
	s.mu.Lock()
	was := s.degraded[key]
	if was {
		delete(s.degraded, key)
	}
	s.mu.Unlock()
	if was {
		s.bus.Publish(events.NewEvent("", "", "", events.EventQuotaMonitorRecovered, "quota monitoring recovered", map[string]any{"profile_key": key}))
	}
}
