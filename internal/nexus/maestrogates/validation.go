package maestrogates

import (
	"fmt"
	"strings"
)

// ValidateStrict validates gates already approved into a WorkPlan against the
// live Maestro capability catalog. Requested gates are safety/process
// requirements: they must never be silently dropped when Maestro degrades.
func ValidateStrict(requested []string, available bool, catalog []string, cause error) ([]string, error) {
	if len(requested) == 0 {
		return nil, nil
	}
	if !available {
		if cause != nil {
			return nil, fmt.Errorf("MAESTRO_DEGRADED: %w", cause)
		}
		return nil, fmt.Errorf("MAESTRO_DEGRADED: maestro is unavailable")
	}
	allowed := make(map[string]struct{}, len(catalog))
	for _, item := range catalog {
		item = strings.TrimSpace(item)
		if item != "" {
			allowed[item] = struct{}{}
		}
	}
	out := make([]string, 0, len(requested))
	for _, gate := range requested {
		gate = strings.TrimSpace(gate)
		if gate == "" {
			continue
		}
		if _, ok := allowed[gate]; !ok {
			return nil, fmt.Errorf("maestro gate/skill %q is unavailable", gate)
		}
		out = append(out, gate)
	}
	return out, nil
}
