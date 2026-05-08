package runfiles

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStorePrepareRunAndChecksum(t *testing.T) {
	store := New(t.TempDir())
	if err := store.PrepareRun("run-1"); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		store.RunDir("run-1"),
		store.CarouselOutputDir("run-1"),
		store.VideoOutputDir("run-1"),
	} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if !info.IsDir() {
			t.Fatalf("expected %s to be a directory", path)
		}
	}

	briefPath := store.BriefPath("run-1", 1)
	if err := WriteJSON(briefPath, map[string]string{"topic": "AI agents"}); err != nil {
		t.Fatal(err)
	}
	sum, err := Checksum(briefPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(sum, "sha256:") {
		t.Fatalf("unexpected checksum %q", sum)
	}
	if filepath.Base(briefPath) != "brief.v1.json" {
		t.Fatalf("unexpected brief path %q", briefPath)
	}
}
