package plangraph

import (
	"fmt"
	"strings"
)

// Node is the dependency-relevant projection of a WorkPackage.
type Node struct {
	ID           string
	Title        string
	Dependencies []string
}

// NormalizeAndValidate resolves dependency references to package IDs and
// rejects malformed DAGs before they can be persisted or executed.
func NormalizeAndValidate(nodes []Node) ([]Node, error) {
	out := make([]Node, len(nodes))
	copy(out, nodes)

	byID := make(map[string]int, len(out))
	titleIDs := make(map[string][]string, len(out))
	for i := range out {
		out[i].ID = strings.TrimSpace(out[i].ID)
		out[i].Title = strings.TrimSpace(out[i].Title)
		if out[i].ID == "" {
			return nil, fmt.Errorf("package id is required")
		}
		if _, exists := byID[out[i].ID]; exists {
			return nil, fmt.Errorf("duplicate package id %s", out[i].ID)
		}
		byID[out[i].ID] = i
		if out[i].Title != "" {
			titleIDs[out[i].Title] = append(titleIDs[out[i].Title], out[i].ID)
		}
	}

	for i := range out {
		seen := map[string]struct{}{}
		normalized := make([]string, 0, len(out[i].Dependencies))
		for _, raw := range out[i].Dependencies {
			dep := strings.TrimSpace(raw)
			if dep == "" {
				continue
			}
			resolved := dep
			if _, ok := byID[dep]; !ok {
				matches := titleIDs[dep]
				switch len(matches) {
				case 0:
					return nil, fmt.Errorf("package %s has unknown dependency %s", out[i].ID, dep)
				case 1:
					resolved = matches[0]
				default:
					return nil, fmt.Errorf("package %s has ambiguous dependency title %q", out[i].ID, dep)
				}
			}
			if resolved == out[i].ID {
				return nil, fmt.Errorf("package %s depends on itself", out[i].ID)
			}
			if _, duplicate := seen[resolved]; duplicate {
				continue
			}
			seen[resolved] = struct{}{}
			normalized = append(normalized, resolved)
		}
		out[i].Dependencies = normalized
	}

	deps := make(map[string][]string, len(out))
	for _, node := range out {
		deps[node.ID] = node.Dependencies
	}
	visiting := make(map[string]bool, len(out))
	visited := make(map[string]bool, len(out))
	var visit func(string) error
	visit = func(id string) error {
		if visiting[id] {
			return fmt.Errorf("dependency cycle includes %s", id)
		}
		if visited[id] {
			return nil
		}
		visiting[id] = true
		for _, dep := range deps[id] {
			if err := visit(dep); err != nil {
				return err
			}
		}
		visiting[id] = false
		visited[id] = true
		return nil
	}
	for id := range deps {
		if err := visit(id); err != nil {
			return nil, err
		}
	}
	return out, nil
}
