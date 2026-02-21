package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ConfigFile holds the results of loading a config file: the parsed struct,
// the raw YAML node tree (for comment-preserving round-trips), and the
// file's modification time (for external-change detection).
type ConfigFile struct {
	Root    *yaml.Node
	ModTime time.Time
}

// LoadConfigFile reads a YAML config file and returns both the raw node tree
// and the file's modification time. The node tree preserves comments, key
// ordering, and unknown keys so that SaveConfigFile can write them back
// unchanged. Callers that need a typed Config struct should yaml.Unmarshal
// the same file bytes into their struct separately.
func LoadConfigFile(path string) (*ConfigFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config file: %w", err)
	}
	defer f.Close() //nolint:errcheck // read-only file; Close error is non-actionable

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat config file: %w", err)
	}

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse config YAML: %w", err)
	}

	return &ConfigFile{
		Root:    &root,
		ModTime: info.ModTime(),
	}, nil
}

// UpdateNodeValue sets the value at the given dot-path key in the YAML node
// tree. It navigates through mapping nodes to find the target key and updates
// its value node in place, preserving comments and surrounding structure.
//
// Supported value types: string, bool, int, int64, []string.
//
// If intermediate mapping sections are missing, they are created. If the final
// key is missing within its parent mapping, it is appended.
func UpdateNodeValue(root *yaml.Node, key string, value any) error {
	if root == nil {
		return errors.New("nil root node")
	}

	// Unwrap document node to get the top-level mapping.
	mapping := root
	if mapping.Kind == yaml.DocumentNode {
		if len(mapping.Content) == 0 {
			// Empty document -- create a mapping.
			mapping.Content = []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}
		}
		mapping = mapping.Content[0]
	}

	if mapping.Kind != yaml.MappingNode {
		return fmt.Errorf("expected mapping node at root, got kind %d", mapping.Kind)
	}

	parts := strings.Split(key, ".")

	return setNestedValue(mapping, parts, value)
}

// SaveConfigFile writes the YAML node tree back to the given path, preserving
// comments, key ordering, and unknown keys. The file is written atomically via
// a temporary file + rename to avoid partial writes on crash.
func SaveConfigFile(path string, root *yaml.Node) error {
	if root == nil {
		return errors.New("nil root node")
	}

	data, err := marshalNode(root)
	if err != nil {
		return fmt.Errorf("marshal config YAML: %w", err)
	}

	// Write to a temp file first, then rename for atomicity.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write temp config file: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		// Clean up the temp file on rename failure.
		os.Remove(tmp) //nolint:errcheck // best-effort cleanup on rename failure
		return fmt.Errorf("rename temp config file: %w", err)
	}

	return nil
}

// BackupConfigFile creates a backup copy of the config file at path + ".bak".
func BackupConfigFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config for backup: %w", err)
	}

	bakPath := path + ".bak"
	if err := os.WriteFile(bakPath, data, 0o644); err != nil {
		return fmt.Errorf("write backup file: %w", err)
	}

	return nil
}

// CheckExternalChange reports whether the file at path has been modified since
// the given modTime. Returns true if the file's current modification time
// differs from modTime (i.e., another process has changed it).
func CheckExternalChange(path string, modTime time.Time) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, fmt.Errorf("stat config file: %w", err)
	}

	return !info.ModTime().Equal(modTime), nil
}

// --- internal helpers ---.

// setNestedValue recursively navigates or creates mapping sections along
// the key path, then sets the leaf value.
func setNestedValue(mapping *yaml.Node, parts []string, value any) error {
	if len(parts) == 0 {
		return errors.New("empty key path")
	}

	if mapping.Kind != yaml.MappingNode {
		return fmt.Errorf("expected mapping node, got kind %d", mapping.Kind)
	}

	target := parts[0]
	remaining := parts[1:]

	// Search for the key in the mapping's Content (key/value pairs).
	for i := 0; i < len(mapping.Content)-1; i += 2 {
		keyNode := mapping.Content[i]
		valNode := mapping.Content[i+1]

		if keyNode.Value != target {
			continue
		}

		// Found the key.
		if len(remaining) == 0 {
			// Leaf: update the value node in place.
			return updateValueNode(valNode, value)
		}

		// Intermediate: recurse into the child mapping.
		if valNode.Kind != yaml.MappingNode {
			return fmt.Errorf("expected mapping at %q, got kind %d", target, valNode.Kind)
		}
		return setNestedValue(valNode, remaining, value)
	}

	// Key not found -- append it.
	if len(remaining) == 0 {
		// Append a leaf key/value pair.
		newKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: target}
		newVal, err := newValueNode(value)
		if err != nil {
			return fmt.Errorf("create value node for %q: %w", target, err)
		}
		mapping.Content = append(mapping.Content, newKey, newVal)
		return nil
	}

	// Append an intermediate mapping and recurse.
	newKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: target}
	newMapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	mapping.Content = append(mapping.Content, newKey, newMapping)
	return setNestedValue(newMapping, remaining, value)
}

// updateValueNode sets the value of an existing yaml.Node in place, preserving
// its comments and style. For scalar values it updates Value/Tag; for sequences
// it replaces Content.
func updateValueNode(node *yaml.Node, value any) error {
	switch v := value.(type) {
	case string:
		node.Kind = yaml.ScalarNode
		node.Tag = "!!str"
		node.Value = v
		node.Content = nil
		return nil

	case bool:
		node.Kind = yaml.ScalarNode
		node.Tag = "!!bool"
		node.Value = strconv.FormatBool(v)
		node.Content = nil
		return nil

	case int:
		node.Kind = yaml.ScalarNode
		node.Tag = "!!int"
		node.Value = strconv.Itoa(v)
		node.Content = nil
		return nil

	case int64:
		node.Kind = yaml.ScalarNode
		node.Tag = "!!int"
		node.Value = strconv.FormatInt(v, 10)
		node.Content = nil
		return nil

	case []string:
		node.Kind = yaml.SequenceNode
		node.Tag = "!!seq"
		node.Value = ""
		node.Style = 0
		items := make([]*yaml.Node, 0, len(v))
		for _, s := range v {
			items = append(items, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: s,
			})
		}
		node.Content = items
		return nil

	default:
		// Complex types (maps, struct slices) are serialised via
		// yaml.Node.Encode, which honours struct yaml tags. This loses
		// per-field comments but correctly round-trips structured data.
		rv := reflect.ValueOf(value)
		if rv.Kind() != reflect.Map && rv.Kind() != reflect.Slice {
			return fmt.Errorf("unsupported value type %T", value)
		}
		var doc yaml.Node
		if err := doc.Encode(value); err != nil {
			return fmt.Errorf("encode complex value for %T: %w", value, err)
		}
		encoded := &doc
		if encoded.Kind == yaml.DocumentNode && len(encoded.Content) > 0 {
			encoded = encoded.Content[0]
		}
		node.Kind = encoded.Kind
		node.Tag = encoded.Tag
		node.Value = encoded.Value
		node.Style = encoded.Style
		node.Content = encoded.Content
		return nil
	}
}

// newValueNode creates a fresh yaml.Node for the given Go value.
func newValueNode(value any) (*yaml.Node, error) {
	node := &yaml.Node{}
	if err := updateValueNode(node, value); err != nil {
		return nil, err
	}
	return node, nil
}

// marshalNode encodes a yaml.Node tree to bytes. The encoder is configured
// with 2-space indentation to match typical YAML style.
func marshalNode(root *yaml.Node) ([]byte, error) {
	var buf strings.Builder
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)

	if err := enc.Encode(root); err != nil {
		return nil, err
	}

	if err := enc.Close(); err != nil {
		return nil, err
	}

	return []byte(buf.String()), nil
}

// GetNodeValue retrieves the value at the given dot-path key from the YAML
// node tree. It returns the raw string value and the node kind, or an error
// if the path does not exist. For sequence nodes it returns a []string.
func GetNodeValue(root *yaml.Node, key string) (any, error) {
	if root == nil {
		return nil, errors.New("nil root node")
	}

	mapping := root
	if mapping.Kind == yaml.DocumentNode {
		if len(mapping.Content) == 0 {
			return nil, errors.New("empty document")
		}
		mapping = mapping.Content[0]
	}

	if mapping.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected mapping node at root, got kind %d", mapping.Kind)
	}

	parts := strings.Split(key, ".")
	return getNestedValue(mapping, parts)
}

// getNestedValue recursively navigates mapping nodes to find the leaf value.
func getNestedValue(mapping *yaml.Node, parts []string) (any, error) {
	if len(parts) == 0 {
		return nil, errors.New("empty key path")
	}

	target := parts[0]
	remaining := parts[1:]

	for i := 0; i < len(mapping.Content)-1; i += 2 {
		keyNode := mapping.Content[i]
		valNode := mapping.Content[i+1]

		if keyNode.Value != target {
			continue
		}

		if len(remaining) == 0 {
			return nodeToValue(valNode)
		}

		if valNode.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("expected mapping at %q, got kind %d", target, valNode.Kind)
		}
		return getNestedValue(valNode, remaining)
	}

	return nil, fmt.Errorf("key %q not found", target)
}

// nodeToValue converts a yaml.Node to a Go value.
func nodeToValue(node *yaml.Node) (any, error) {
	switch node.Kind {
	case yaml.ScalarNode:
		return node.Value, nil
	case yaml.SequenceNode:
		items := make([]string, 0, len(node.Content))
		for _, child := range node.Content {
			if child.Kind == yaml.ScalarNode {
				items = append(items, child.Value)
			}
		}
		return items, nil
	case yaml.MappingNode:
		// Return the node itself for complex types (worktrees map, etc.)
		return node, nil
	default:
		return nil, fmt.Errorf("unsupported node kind %d", node.Kind)
	}
}
