package migrations

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Direction string

const (
	DirectionUp   Direction = "up"
	DirectionDown Direction = "down"
)

type File struct {
	Version string
	Name    string
	Path    string
	SQL     string
}

func LoadFiles(dir string, direction Direction) ([]File, error) {
	pattern := filepath.Join(dir, fmt.Sprintf("*.%s.sql", direction))
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob migrations: %w", err)
	}

	sort.Strings(paths)
	if direction == DirectionDown {
		sort.Sort(sort.Reverse(sort.StringSlice(paths)))
	}

	files := make([]File, 0, len(paths))
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", path, err)
		}

		version, name, err := parseFilename(filepath.Base(path), direction)
		if err != nil {
			return nil, err
		}

		files = append(files, File{
			Version: version,
			Name:    name,
			Path:    path,
			SQL:     string(content),
		})
	}

	return files, nil
}

func parseFilename(filename string, direction Direction) (string, string, error) {
	suffix := fmt.Sprintf(".%s.sql", direction)
	if !strings.HasSuffix(filename, suffix) {
		return "", "", fmt.Errorf("invalid migration filename %q", filename)
	}

	base := strings.TrimSuffix(filename, suffix)
	parts := strings.SplitN(base, "_", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid migration filename %q", filename)
	}

	return parts[0], parts[1], nil
}
