package conversation

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Conversation struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Provider     string    `json:"provider"`
	Workspace    string    `json:"workspace"`
	LastModified time.Time `json:"last_modified"`
}

// ListRecent returns unique recent conversations from AGY and Codex.
func ListRecent(limit int, workspaceFilter string) []Conversation {
	convs := make(map[string]*Conversation)

	// 1. Discover AGY conversations
	loadAgyConversations(convs)

	// 2. Discover Codex conversations
	loadCodexConversations(convs)

	var list []Conversation
	for _, c := range convs {
		if c.ID == "" {
			continue
		}
		list = append(list, *c)
	}

	// Prioritize current workspace, then sort by LastModified descending
	cwd, _ := os.Getwd()
	sort.Slice(list, func(i, j int) bool {
		iCwd := cwd != "" && (list[i].Workspace == cwd || strings.HasPrefix(cwd, list[i].Workspace))
		jCwd := cwd != "" && (list[j].Workspace == cwd || strings.HasPrefix(cwd, list[j].Workspace))
		if iCwd != jCwd {
			return iCwd
		}
		return list[i].LastModified.After(list[j].LastModified)
	})

	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}
	return list
}

func loadAgyConversations(convs map[string]*Conversation) {
	homeCandidates := []string{
		os.Getenv("AI_REAL_HOME"),
		"/home/desenvolvedor",
	}
	if h, err := os.UserHomeDir(); err == nil {
		homeCandidates = append(homeCandidates, h)
	}

	for _, h := range homeCandidates {
		if h == "" {
			continue
		}
		histFile := filepath.Join(h, ".gemini", "antigravity-cli", "history.jsonl")
		f, err := os.Open(histFile)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var entry struct {
				Display        string `json:"display"`
				Timestamp      int64  `json:"timestamp"`
				Workspace      string `json:"workspace"`
				ConversationID string `json:"conversationId"`
				Type           string `json:"type"`
			}
			if err := json.Unmarshal([]byte(line), &entry); err == nil && entry.ConversationID != "" {
				t := time.UnixMilli(entry.Timestamp)
				c, exists := convs[entry.ConversationID]
				if !exists {
					title := entry.Display
					if strings.HasPrefix(title, "/") {
						title = "Conversation " + entry.ConversationID[:8]
					}
					if len(title) > 60 {
						title = title[:57] + "..."
					}
					convs[entry.ConversationID] = &Conversation{
						ID:           entry.ConversationID,
						Title:        title,
						Provider:     "agy",
						Workspace:    entry.Workspace,
						LastModified: t,
					}
				} else {
					if t.After(c.LastModified) {
						c.LastModified = t
					}
					if c.Title == "" || strings.HasPrefix(c.Title, "Conversation ") {
						if !strings.HasPrefix(entry.Display, "/") && entry.Display != "" {
							title := entry.Display
							if len(title) > 60 {
								title = title[:57] + "..."
							}
							c.Title = title
						}
					}
				}
			}
		}
		f.Close()
		break // Found valid history
	}
}

func loadCodexConversations(convs map[string]*Conversation) {
	homeCandidates := []string{
		os.Getenv("AI_REAL_HOME"),
		"/home/desenvolvedor",
	}
	if h, err := os.UserHomeDir(); err == nil {
		homeCandidates = append(homeCandidates, h)
	}

	for _, h := range homeCandidates {
		if h == "" {
			continue
		}
		indexFile := filepath.Join(h, ".codex", "session_index.jsonl")
		f, err := os.Open(indexFile)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var entry struct {
				ID         string `json:"id"`
				ThreadName string `json:"thread_name"`
				UpdatedAt  string `json:"updated_at"`
			}
			if err := json.Unmarshal([]byte(line), &entry); err == nil && entry.ID != "" {
				t, _ := time.Parse(time.RFC3339, entry.UpdatedAt)
				title := entry.ThreadName
				if title == "" {
					title = "Codex Thread " + entry.ID[:8]
				}
				if len(title) > 60 {
					title = title[:57] + "..."
				}
				convs[entry.ID] = &Conversation{
					ID:           entry.ID,
					Title:        title,
					Provider:     "codex",
					LastModified: t,
				}
			}
		}
		f.Close()
		break
	}
}
