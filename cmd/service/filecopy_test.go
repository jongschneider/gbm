package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterFiles(t *testing.T) {
	tests := []struct {
		name    string
		files   []string
		exclude []string
		want    []string
	}{
		{
			name:    "no exclusions",
			files:   []string{"a.go", "b.txt", "c.md"},
			exclude: []string{},
			want:    []string{"a.go", "b.txt", "c.md"},
		},
		{
			name:    "exclude by extension",
			files:   []string{"a.log", "b.txt", "c.log"},
			exclude: []string{"*.log"},
			want:    []string{"b.txt"},
		},
		{
			name:    "exclude multiple patterns",
			files:   []string{".env", "app.log", "main.go", "node_modules.txt"},
			exclude: []string{".env", "*.log"},
			want:    []string{"main.go", "node_modules.txt"},
		},
		{
			name:    "exclude directory - note: simple glob doesn't match paths",
			files:   []string{"src/main.go", "node_modules/pkg.json", "dist/index.js"},
			exclude: []string{"node_modules/"},
			want:    []string{"src/main.go", "node_modules/pkg.json", "dist/index.js"},
		},
		{
			name:    "empty files list",
			files:   []string{},
			exclude: []string{"*.log"},
			want:    nil,
		},
		{
			name:    "all files excluded",
			files:   []string{"a.log", "b.log", "c.log"},
			exclude: []string{"*.log"},
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterFiles(tt.files, tt.exclude)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestMatchesAnyPattern(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		patterns []string
		want     bool
	}{
		{
			name:     "single pattern match",
			path:     "file.log",
			patterns: []string{"*.log"},
			want:     true,
		},
		{
			name:     "exact match",
			path:     ".env",
			patterns: []string{".env", ".gitignore"},
			want:     true,
		},
		{
			name:     "no match",
			path:     "main.go",
			patterns: []string{"*.log", ".env"},
			want:     false,
		},
		{
			name:     "empty patterns",
			path:     "file.txt",
			patterns: []string{},
			want:     false,
		},
		{
			name:     "directory pattern - simple glob only matches basename",
			path:     "node_modules/package.json",
			patterns: []string{"node_modules/"},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesAnyPattern(tt.path, tt.patterns)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		pattern string
		want    bool
	}{
		{
			name:    "wildcard extension",
			path:    "file.log",
			pattern: "*.log",
			want:    true,
		},
		{
			name:    "no wildcard exact match",
			path:    ".env",
			pattern: ".env",
			want:    true,
		},
		{
			name:    "bare wildcard",
			path:    "anything",
			pattern: "*",
			want:    true,
		},
		{
			name:    "no match",
			path:    "file.txt",
			pattern: "*.log",
			want:    false,
		},
		{
			name:    "prefix match",
			path:    "test_file.txt",
			pattern: "test_*",
			want:    true,
		},
		{
			name:    "directory pattern",
			path:    "node_modules",
			pattern: "node_modules/",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchGlob(tt.path, tt.pattern)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestGlobMatch(t *testing.T) {
	tests := []struct {
		name    string
		name_   string
		pattern string
		want    bool
	}{
		{
			name:    "exact match",
			name_:   "file.txt",
			pattern: "file.txt",
			want:    true,
		},
		{
			name:    "bare wildcard",
			name_:   "anything",
			pattern: "*",
			want:    true,
		},
		{
			name:    "suffix wildcard",
			name_:   "app.log",
			pattern: "*.log",
			want:    true,
		},
		{
			name:    "prefix wildcard",
			name_:   "test_file.go",
			pattern: "test_*",
			want:    true,
		},
		{
			name:    "no match suffix",
			name_:   "file.txt",
			pattern: "*.log",
			want:    false,
		},
		{
			name:    "directory with slash",
			name_:   "node_modules",
			pattern: "node_modules/",
			want:    true,
		},
		{
			name:    "no match directory",
			name_:   "src",
			pattern: "node_modules/",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := globMatch(tt.name_, tt.pattern)
			assert.Equal(t, tt.want, result)
		})
	}
}

// Note: Integration tests for CopyFilesToWorktree are covered by e2e_test.go
// This file focuses on unit tests for helper functions that are exported/testable.
