package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// --- LoadConfigFile ---.

func TestLoadConfigFile(t *testing.T) {
	testCases := []struct {
		assert      func(t *testing.T, cf *ConfigFile)
		assertError func(t *testing.T, err error)
		name        string
		content     string
	}{
		{
			name: "loads simple config into node tree",
			content: `default_branch: main
worktrees_dir: worktrees
`,
			assert: func(t *testing.T, cf *ConfigFile) {
				t.Helper()
				require.NotNil(t, cf.Root)
				assert.Equal(t, yaml.DocumentNode, cf.Root.Kind)
				assert.False(t, cf.ModTime.IsZero())
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name: "preserves comments in node tree",
			content: `# Top-level comment
default_branch: main  # inline comment
# Section comment
worktrees_dir: worktrees
`,
			assert: func(t *testing.T, cf *ConfigFile) {
				t.Helper()
				require.NotNil(t, cf.Root)
				mapping := cf.Root.Content[0]
				assert.Equal(t, yaml.MappingNode, mapping.Kind)
				// The first key node should have a head comment.
				assert.Contains(t, mapping.Content[0].HeadComment, "Top-level comment")
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name: "preserves unknown keys",
			content: `default_branch: main
custom_key: custom_value
worktrees_dir: worktrees
`,
			assert: func(t *testing.T, cf *ConfigFile) {
				t.Helper()
				mapping := cf.Root.Content[0]
				// Should have 3 key-value pairs = 6 content nodes.
				assert.Len(t, mapping.Content, 6)
				// Second key should be custom_key.
				assert.Equal(t, "custom_key", mapping.Content[2].Value)
				assert.Equal(t, "custom_value", mapping.Content[3].Value)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:    "returns error for invalid YAML",
			content: `{invalid: [yaml: broken`,
			assert:  nil,
			assertError: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Contains(t, err.Error(), "parse config YAML")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeTestFile(t, tc.content)
			cf, err := LoadConfigFile(path)
			tc.assertError(t, err)
			if tc.assert != nil {
				tc.assert(t, cf)
			}
		})
	}
}

func TestLoadConfigFile_nonexistent(t *testing.T) {
	_, err := LoadConfigFile("/nonexistent/path/config.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "open config file")
}

// --- UpdateNodeValue ---.

func TestUpdateNodeValue(t *testing.T) {
	testCases := []struct {
		value       any
		assert      func(t *testing.T, root *yaml.Node)
		assertError func(t *testing.T, err error)
		name        string
		input       string
		key         string
	}{
		{
			name:  "updates string value",
			input: "default_branch: main\n",
			key:   "default_branch",
			value: "develop",
			assert: func(t *testing.T, root *yaml.Node) {
				t.Helper()
				v, err := GetNodeValue(root, "default_branch")
				require.NoError(t, err)
				assert.Equal(t, "develop", v)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "updates bool value",
			input: "jira:\n  attachments:\n    enabled: false\n",
			key:   "jira.attachments.enabled",
			value: true,
			assert: func(t *testing.T, root *yaml.Node) {
				t.Helper()
				v, err := GetNodeValue(root, "jira.attachments.enabled")
				require.NoError(t, err)
				assert.Equal(t, "true", v)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "updates int value",
			input: "jira:\n  attachments:\n    max_size_mb: 50\n",
			key:   "jira.attachments.max_size_mb",
			value: 100,
			assert: func(t *testing.T, root *yaml.Node) {
				t.Helper()
				v, err := GetNodeValue(root, "jira.attachments.max_size_mb")
				require.NoError(t, err)
				assert.Equal(t, "100", v)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "updates int64 value",
			input: "jira:\n  attachments:\n    max_size_mb: 50\n",
			key:   "jira.attachments.max_size_mb",
			value: int64(200),
			assert: func(t *testing.T, root *yaml.Node) {
				t.Helper()
				v, err := GetNodeValue(root, "jira.attachments.max_size_mb")
				require.NoError(t, err)
				assert.Equal(t, "200", v)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "updates string list value",
			input: "jira:\n  filters:\n    status:\n      - Open\n      - Closed\n",
			key:   "jira.filters.status",
			value: []string{"In Progress", "Done", "Blocked"},
			assert: func(t *testing.T, root *yaml.Node) {
				t.Helper()
				v, err := GetNodeValue(root, "jira.filters.status")
				require.NoError(t, err)
				assert.Equal(t, []string{"In Progress", "Done", "Blocked"}, v)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "creates missing leaf key",
			input: "default_branch: main\n",
			key:   "worktrees_dir",
			value: "wt",
			assert: func(t *testing.T, root *yaml.Node) {
				t.Helper()
				v, err := GetNodeValue(root, "worktrees_dir")
				require.NoError(t, err)
				assert.Equal(t, "wt", v)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "creates missing intermediate and leaf keys",
			input: "default_branch: main\n",
			key:   "jira.host",
			value: "https://jira.example.com",
			assert: func(t *testing.T, root *yaml.Node) {
				t.Helper()
				v, err := GetNodeValue(root, "jira.host")
				require.NoError(t, err)
				assert.Equal(t, "https://jira.example.com", v)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "preserves comments when updating value",
			input: "default_branch: main  # primary branch\n",
			key:   "default_branch",
			value: "develop",
			assert: func(t *testing.T, root *yaml.Node) {
				t.Helper()
				mapping := root.Content[0]
				valNode := mapping.Content[1]
				assert.Equal(t, "develop", valNode.Value)
				assert.Contains(t, valNode.LineComment, "primary branch")
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "preserves other keys when updating",
			input: "default_branch: main\nunknown_key: keep_me\nworktrees_dir: wt\n",
			key:   "default_branch",
			value: "develop",
			assert: func(t *testing.T, root *yaml.Node) {
				t.Helper()
				v, err := GetNodeValue(root, "unknown_key")
				require.NoError(t, err)
				assert.Equal(t, "keep_me", v)
				v, err = GetNodeValue(root, "worktrees_dir")
				require.NoError(t, err)
				assert.Equal(t, "wt", v)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "returns error for nil root",
			input: "",
			key:   "anything",
			value: "val",
			assert: func(t *testing.T, root *yaml.Node) {
				t.Helper()
				// not called
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Contains(t, err.Error(), "nil root node")
			},
		},
		{
			name:  "returns error for unsupported value type",
			input: "default_branch: main\n",
			key:   "default_branch",
			value: 3.14,
			assert: func(t *testing.T, root *yaml.Node) {
				t.Helper()
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unsupported value type")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "returns error for nil root" {
				err := UpdateNodeValue(nil, tc.key, tc.value)
				tc.assertError(t, err)
				return
			}

			root := parseYAML(t, tc.input)
			err := UpdateNodeValue(root, tc.key, tc.value)
			tc.assertError(t, err)
			if err == nil {
				tc.assert(t, root)
			}
		})
	}
}

func TestUpdateNodeValue_empty_document(t *testing.T) {
	// An empty YAML document should get a mapping created.
	root := &yaml.Node{Kind: yaml.DocumentNode}
	err := UpdateNodeValue(root, "new_key", "new_value")
	require.NoError(t, err)

	v, err := GetNodeValue(root, "new_key")
	require.NoError(t, err)
	assert.Equal(t, "new_value", v)
}

// --- SaveConfigFile ---.

func TestSaveConfigFile(t *testing.T) {
	testCases := []struct {
		assert      func(t *testing.T, savedContent string)
		assertError func(t *testing.T, err error)
		name        string
		input       string
	}{
		{
			name:  "saves valid YAML",
			input: "default_branch: main\nworktrees_dir: worktrees\n",
			assert: func(t *testing.T, saved string) {
				t.Helper()
				assert.Contains(t, saved, "default_branch: main")
				assert.Contains(t, saved, "worktrees_dir: worktrees")
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name: "preserves comments through save",
			input: `# Top comment
default_branch: main  # inline comment
`,
			assert: func(t *testing.T, saved string) {
				t.Helper()
				assert.Contains(t, saved, "# Top comment")
				assert.Contains(t, saved, "# inline comment")
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name: "preserves key ordering through save",
			input: `worktrees_dir: worktrees
default_branch: main
custom_field: value
`,
			assert: func(t *testing.T, saved string) {
				t.Helper()
				// worktrees_dir should appear before default_branch.
				wtIdx := indexOf(saved, "worktrees_dir")
				dbIdx := indexOf(saved, "default_branch")
				cfIdx := indexOf(saved, "custom_field")
				assert.Less(t, wtIdx, dbIdx, "worktrees_dir should come before default_branch")
				assert.Less(t, dbIdx, cfIdx, "default_branch should come before custom_field")
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root := parseYAML(t, tc.input)
			dir := t.TempDir()
			outPath := filepath.Join(dir, "config.yaml")

			err := SaveConfigFile(outPath, root)
			tc.assertError(t, err)
			if err != nil {
				return
			}

			saved, err := os.ReadFile(outPath)
			require.NoError(t, err)
			tc.assert(t, string(saved))
		})
	}
}

func TestSaveConfigFile_nil_root(t *testing.T) {
	err := SaveConfigFile("/tmp/test.yaml", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil root node")
}

func TestSaveConfigFile_atomicity(t *testing.T) {
	// Verify no .tmp file is left behind after a successful save.
	root := parseYAML(t, "key: value\n")
	dir := t.TempDir()
	outPath := filepath.Join(dir, "config.yaml")

	err := SaveConfigFile(outPath, root)
	require.NoError(t, err)

	_, err = os.Stat(outPath + ".tmp")
	assert.True(t, os.IsNotExist(err), "temp file should be cleaned up")
}

// --- BackupConfigFile ---.

func TestBackupConfigFile(t *testing.T) {
	testCases := []struct {
		assert      func(t *testing.T, original, bakPath string)
		assertError func(t *testing.T, err error)
		name        string
		content     string
	}{
		{
			name:    "creates .bak copy",
			content: "default_branch: main\n",
			assert: func(t *testing.T, original, bakPath string) {
				t.Helper()
				origData, err := os.ReadFile(original)
				require.NoError(t, err)
				bakData, err := os.ReadFile(bakPath)
				require.NoError(t, err)
				assert.Equal(t, origData, bakData)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeTestFile(t, tc.content)
			err := BackupConfigFile(path)
			tc.assertError(t, err)
			if tc.assert != nil {
				tc.assert(t, path, path+".bak")
			}
		})
	}
}

func TestBackupConfigFile_nonexistent(t *testing.T) {
	err := BackupConfigFile("/nonexistent/path/config.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read config for backup")
}

// --- CheckExternalChange ---.

func TestCheckExternalChange(t *testing.T) {
	testCases := []struct {
		setup       func(t *testing.T) (path string, modTime time.Time)
		assert      func(t *testing.T, changed bool)
		assertError func(t *testing.T, err error)
		name        string
	}{
		{
			name: "no change when modtime matches",
			setup: func(t *testing.T) (string, time.Time) {
				t.Helper()
				path := writeTestFile(t, "key: value\n")
				info, err := os.Stat(path)
				require.NoError(t, err)
				return path, info.ModTime()
			},
			assert: func(t *testing.T, changed bool) {
				t.Helper()
				assert.False(t, changed)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name: "detects change when modtime differs",
			setup: func(t *testing.T) (string, time.Time) {
				t.Helper()
				path := writeTestFile(t, "key: value\n")
				oldTime := time.Now().Add(-1 * time.Hour)
				return path, oldTime
			},
			assert: func(t *testing.T, changed bool) {
				t.Helper()
				assert.True(t, changed)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path, modTime := tc.setup(t)
			changed, err := CheckExternalChange(path, modTime)
			tc.assertError(t, err)
			tc.assert(t, changed)
		})
	}
}

func TestCheckExternalChange_nonexistent(t *testing.T) {
	_, err := CheckExternalChange("/nonexistent/file.yaml", time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stat config file")
}

// --- Round-trip tests ---.

func TestRoundTrip_comments_preserved(t *testing.T) {
	input := `# Git Branch Manager Configuration
default_branch: main  # primary branch
# Worktree settings
worktrees_dir: worktrees
`
	path := writeTestFile(t, input)
	cf, err := LoadConfigFile(path)
	require.NoError(t, err)

	// Modify a value.
	err = UpdateNodeValue(cf.Root, "default_branch", "develop")
	require.NoError(t, err)

	// Save to a new file.
	outPath := filepath.Join(t.TempDir(), "out.yaml")
	err = SaveConfigFile(outPath, cf.Root)
	require.NoError(t, err)

	saved, err := os.ReadFile(outPath)
	require.NoError(t, err)
	content := string(saved)

	assert.Contains(t, content, "# Git Branch Manager Configuration")
	assert.Contains(t, content, "# primary branch")
	assert.Contains(t, content, "# Worktree settings")
	assert.Contains(t, content, "develop")
	assert.NotContains(t, content, "main")
}

func TestRoundTrip_unknown_keys_preserved(t *testing.T) {
	input := `default_branch: main
remotes:
  origin:
    url: git@github.com:user/repo.git
  upstream:
    url: git@github.com:org/repo.git
worktrees_dir: worktrees
future_feature:
  enabled: true
  setting: value
`
	path := writeTestFile(t, input)
	cf, err := LoadConfigFile(path)
	require.NoError(t, err)

	// Modify a known value.
	err = UpdateNodeValue(cf.Root, "default_branch", "develop")
	require.NoError(t, err)

	// Save to a new file.
	outPath := filepath.Join(t.TempDir(), "out.yaml")
	err = SaveConfigFile(outPath, cf.Root)
	require.NoError(t, err)

	saved, err := os.ReadFile(outPath)
	require.NoError(t, err)
	content := string(saved)

	// All unknown keys must survive.
	assert.Contains(t, content, "remotes:")
	assert.Contains(t, content, "origin:")
	assert.Contains(t, content, "git@github.com:user/repo.git")
	assert.Contains(t, content, "upstream:")
	assert.Contains(t, content, "git@github.com:org/repo.git")
	assert.Contains(t, content, "future_feature:")
	assert.Contains(t, content, "enabled: true")
	assert.Contains(t, content, "setting: value")

	// The modified value should be updated.
	assert.Contains(t, content, "default_branch: develop")
}

func TestRoundTrip_key_ordering_preserved(t *testing.T) {
	input := `worktrees_dir: worktrees
default_branch: main
jira:
  host: https://jira.example.com
`
	path := writeTestFile(t, input)
	cf, err := LoadConfigFile(path)
	require.NoError(t, err)

	err = UpdateNodeValue(cf.Root, "jira.host", "https://new.jira.com")
	require.NoError(t, err)

	outPath := filepath.Join(t.TempDir(), "out.yaml")
	err = SaveConfigFile(outPath, cf.Root)
	require.NoError(t, err)

	saved, err := os.ReadFile(outPath)
	require.NoError(t, err)
	content := string(saved)

	wtIdx := indexOf(content, "worktrees_dir")
	dbIdx := indexOf(content, "default_branch")
	jIdx := indexOf(content, "jira:")
	assert.Less(t, wtIdx, dbIdx)
	assert.Less(t, dbIdx, jIdx)
}

func TestRoundTrip_full_config_with_comments_and_unknown_keys(t *testing.T) {
	input := `# Git Branch Manager Configuration
# This file is managed by gbm

default_branch: main  # primary branch

# Remote configuration (not editable in TUI)
remotes:
  origin:
    url: git@github.com:user/repo.git
  upstream:
    url: git@github.com:org/repo.git

worktrees_dir: worktrees

# JIRA Integration
jira:
  host: https://jira.example.com
  me: user@example.com
  attachments:
    enabled: true
    max_size_mb: 50  # max file size
    directory: ".jira/attachments"
  filters:
    status:
      - Open
      - "In Progress"

# File copying settings
file_copy:
  auto:
    enabled: true
    source_worktree: "{default}"
    copy_ignored: true
    exclude:
      - "*.log"
      - node_modules/

# Unknown future setting
experimental:
  new_feature: on
`

	path := writeTestFile(t, input)
	cf, err := LoadConfigFile(path)
	require.NoError(t, err)

	// Make several modifications.
	require.NoError(t, UpdateNodeValue(cf.Root, "default_branch", "develop"))
	require.NoError(t, UpdateNodeValue(cf.Root, "jira.attachments.max_size_mb", 100))
	require.NoError(t, UpdateNodeValue(cf.Root, "jira.filters.status", []string{"Done", "Blocked"}))
	require.NoError(t, UpdateNodeValue(cf.Root, "file_copy.auto.copy_ignored", false))

	// Save.
	outPath := filepath.Join(t.TempDir(), "out.yaml")
	require.NoError(t, SaveConfigFile(outPath, cf.Root))

	saved, err := os.ReadFile(outPath)
	require.NoError(t, err)
	content := string(saved)

	// Comments preserved.
	assert.Contains(t, content, "# Git Branch Manager Configuration")
	assert.Contains(t, content, "# This file is managed by gbm")
	assert.Contains(t, content, "# primary branch")
	assert.Contains(t, content, "# Remote configuration (not editable in TUI)")
	assert.Contains(t, content, "# JIRA Integration")
	assert.Contains(t, content, "# max file size")
	assert.Contains(t, content, "# File copying settings")
	assert.Contains(t, content, "# Unknown future setting")

	// Unknown keys preserved.
	assert.Contains(t, content, "remotes:")
	assert.Contains(t, content, "origin:")
	assert.Contains(t, content, "upstream:")
	assert.Contains(t, content, "experimental:")
	assert.Contains(t, content, "new_feature: on")

	// Modifications applied.
	assert.Contains(t, content, "default_branch: develop")
	assert.Contains(t, content, "max_size_mb: 100")
	assert.Contains(t, content, "- Done")
	assert.Contains(t, content, "- Blocked")
	assert.NotContains(t, content, "- Open")
	assert.Contains(t, content, "copy_ignored: false")
}

// --- Missing sections ---.

func TestUpdateNodeValue_missing_sections(t *testing.T) {
	testCases := []struct {
		value       any
		assert      func(t *testing.T, root *yaml.Node)
		assertError func(t *testing.T, err error)
		name        string
		input       string
		key         string
	}{
		{
			name:  "creates deeply nested path",
			input: "default_branch: main\n",
			key:   "jira.attachments.enabled",
			value: true,
			assert: func(t *testing.T, root *yaml.Node) {
				t.Helper()
				v, err := GetNodeValue(root, "jira.attachments.enabled")
				require.NoError(t, err)
				assert.Equal(t, "true", v)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "creates missing intermediate section with existing parent",
			input: "jira:\n  host: https://jira.example.com\n",
			key:   "jira.attachments.enabled",
			value: true,
			assert: func(t *testing.T, root *yaml.Node) {
				t.Helper()
				// Original value should still be there.
				v, err := GetNodeValue(root, "jira.host")
				require.NoError(t, err)
				assert.Equal(t, "https://jira.example.com", v)
				// New value should exist.
				v, err = GetNodeValue(root, "jira.attachments.enabled")
				require.NoError(t, err)
				assert.Equal(t, "true", v)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root := parseYAML(t, tc.input)
			err := UpdateNodeValue(root, tc.key, tc.value)
			tc.assertError(t, err)
			if err == nil {
				tc.assert(t, root)
			}
		})
	}
}

// --- GetNodeValue ---.

func TestGetNodeValue(t *testing.T) {
	testCases := []struct {
		assert      func(t *testing.T, val any)
		assertError func(t *testing.T, err error)
		name        string
		input       string
		key         string
	}{
		{
			name:  "gets scalar string",
			input: "default_branch: main\n",
			key:   "default_branch",
			assert: func(t *testing.T, val any) {
				t.Helper()
				assert.Equal(t, "main", val)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "gets nested scalar",
			input: "jira:\n  host: https://jira.example.com\n",
			key:   "jira.host",
			assert: func(t *testing.T, val any) {
				t.Helper()
				assert.Equal(t, "https://jira.example.com", val)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "gets sequence as string slice",
			input: "jira:\n  filters:\n    status:\n      - Open\n      - Closed\n",
			key:   "jira.filters.status",
			assert: func(t *testing.T, val any) {
				t.Helper()
				assert.Equal(t, []string{"Open", "Closed"}, val)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "returns mapping node for complex values",
			input: "worktrees:\n  main:\n    branch: main\n",
			key:   "worktrees.main",
			assert: func(t *testing.T, val any) {
				t.Helper()
				node, ok := val.(*yaml.Node)
				require.True(t, ok)
				assert.Equal(t, yaml.MappingNode, node.Kind)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "returns error for missing key",
			input: "default_branch: main\n",
			key:   "nonexistent",
			assert: func(t *testing.T, val any) {
				t.Helper()
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Contains(t, err.Error(), "not found")
			},
		},
		{
			name:  "returns error for nil root",
			input: "",
			key:   "anything",
			assert: func(t *testing.T, val any) {
				t.Helper()
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Contains(t, err.Error(), "nil root node")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "returns error for nil root" {
				val, err := GetNodeValue(nil, tc.key)
				tc.assertError(t, err)
				if err == nil {
					tc.assert(t, val)
				}
				return
			}

			root := parseYAML(t, tc.input)
			val, err := GetNodeValue(root, tc.key)
			tc.assertError(t, err)
			if err == nil {
				tc.assert(t, val)
			}
		})
	}
}

// --- test helpers ---.

// writeTestFile writes content to a temp file and returns the path.
func writeTestFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)
	return path
}

// parseYAML parses a YAML string into a yaml.Node tree.
func parseYAML(t *testing.T, input string) *yaml.Node {
	t.Helper()
	var root yaml.Node
	err := yaml.Unmarshal([]byte(input), &root)
	require.NoError(t, err)
	return &root
}

// indexOf returns the byte index of the first occurrence of substr in s, or -1.
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
