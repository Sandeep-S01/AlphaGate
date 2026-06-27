package migrations

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFilesReturnsSortedUpMigrations(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "000002_second.up.sql"), "SELECT 2;")
	writeFile(t, filepath.Join(dir, "000001_first.up.sql"), "SELECT 1;")
	writeFile(t, filepath.Join(dir, "000001_first.down.sql"), "DROP 1;")

	files, err := LoadFiles(dir, DirectionUp)
	if err != nil {
		t.Fatalf("LoadFiles returned error: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 up migrations, got %d", len(files))
	}
	if files[0].Version != "000001" || files[1].Version != "000002" {
		t.Fatalf("expected sorted versions, got %+v", files)
	}
	if files[0].Name != "first" {
		t.Fatalf("expected name first, got %q", files[0].Name)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}
