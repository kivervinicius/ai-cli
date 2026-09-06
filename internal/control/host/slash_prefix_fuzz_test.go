package host

import "testing"

func FuzzSlashPrefixRouter(f *testing.F) {
	seeds := []string{
		"/ai status",
		"/ai status\n",
		"/ai usage extra\n",
		"/help\n",
		"//ai literal\n",
		"//ai\n",
		"/a",
		"/ai\n",
		"normal text with /ai inside\n",
		"/ai\x03",
		"/ai stop\n",
		"/ai handoff codex:work\n",
		"/ai continue claude\n",
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		r := NewSlashPrefixRouter()
		for _, b := range data {
			_ = r.ProcessByte(b)
		}
		// Reset must never panic.
		r.Reset()
		_ = r.ProcessByte('/')
		_ = r.ProcessByte('a')
		_ = r.ProcessByte('i')
		_ = r.ProcessByte(' ')
		_ = r.ProcessByte('\n')
	})
}
