// Package config provides field metadata types and section declarations
// that drive the Config TUI. Every editable config field is described by a
// FieldMeta value; section slices group those fields by tab.
package config

import (
	"errors"
	"fmt"
)

// FieldType enumerates the kinds of values a config field can hold.
// The TUI renderer and editor use this to choose the appropriate widget.
type FieldType int

const (
	// String is a plain text value.
	String FieldType = iota
	// SensitiveString displays as masked (********) when unfocused.
	SensitiveString
	// Int is an integer value.
	Int
	// Bool is a boolean toggle.
	Bool
	// StringList is an ordered list of strings (e.g. status filters).
	StringList
	// ObjectList is a list of structured objects (e.g. file copy rules).
	ObjectList
)

// String returns the human-readable name of a FieldType.
func (ft FieldType) String() string {
	switch ft {
	case String:
		return "String"
	case SensitiveString:
		return "SensitiveString"
	case Int:
		return "Int"
	case Bool:
		return "Bool"
	case StringList:
		return "StringList"
	case ObjectList:
		return "ObjectList"
	default:
		return fmt.Sprintf("FieldType(%d)", int(ft))
	}
}

// FieldMeta describes a single config field for the TUI.
// The Key uses dot-path notation matching the YAML structure
// (e.g. "jira.attachments.max_size_mb"). Overlay fields such as
// file-copy rules and worktree entries use short keys relative to
// their parent object (e.g. "source_worktree", "branch").
type FieldMeta struct {
	Validate    func(any) error
	Key         string
	Label       string
	Group       string
	Description string
	Type        FieldType
}

// --- Reusable validation functions ---.

// ValidateRequired returns an error if the value is an empty string.
func ValidateRequired(v any) error {
	s, ok := v.(string)
	if !ok {
		return errors.New("expected a string value")
	}
	if s == "" {
		return errors.New("this field is required")
	}
	return nil
}

// ValidatePositiveInt returns an error if the value is not a positive integer.
func ValidatePositiveInt(v any) error {
	switch n := v.(type) {
	case int:
		if n <= 0 {
			return errors.New("must be a positive integer")
		}
		return nil
	case int64:
		if n <= 0 {
			return errors.New("must be a positive integer")
		}
		return nil
	default:
		return fmt.Errorf("expected an integer, got %T", v)
	}
}

// ValidateNonNegativeInt returns an error if the value is a negative integer.
func ValidateNonNegativeInt(v any) error {
	switch n := v.(type) {
	case int:
		if n < 0 {
			return errors.New("must be zero or a positive integer")
		}
		return nil
	case int64:
		if n < 0 {
			return errors.New("must be zero or a positive integer")
		}
		return nil
	default:
		return fmt.Errorf("expected an integer, got %T", v)
	}
}
