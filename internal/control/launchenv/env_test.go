package launchenv

import (
	"os"
	"strings"
	"testing"
)

func TestMergeReplacesDuplicateOverride(t *testing.T) {
	got := Merge([]string{"A=old", "PATH=/driver/bin"}, map[string]string{"A": "new"}, nil)
	joined := strings.Join(got, "\n")
	if strings.Count(joined, "A=") != 1 || !strings.Contains(joined, "A=new") {
		t.Fatalf("override was not canonicalized: %v", got)
	}
}

func TestMergePrependsGuardPathAndPreservesDriverPATH(t *testing.T) {
	sep := string(os.PathListSeparator)
	got := Merge([]string{"PATH=/driver/bin"}, nil, []string{"/guard"})
	var path string
	for _, entry := range got {
		if strings.HasPrefix(entry, "PATH=") {
			path = strings.TrimPrefix(entry, "PATH=")
		}
	}
	if path != "/guard"+sep+"/driver/bin" {
		t.Fatalf("unexpected PATH %q", path)
	}
}
