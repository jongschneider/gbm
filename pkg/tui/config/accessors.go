package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrUnknownKey is returned when a dot-path key is not recognized.
var ErrUnknownKey = errors.New("unknown config key")

// ConfigAccessor provides get/set access to config values by dot-path key.
// The TUI uses this interface to read and write config fields without
// depending on the concrete Config struct (which lives in cmd/service/).
type ConfigAccessor interface {
	GetValue(key string) any
	SetValue(key string, value any) error

	// ReloadFromFile re-reads the config file at the given path and
	// re-unmarshals its contents into the underlying config struct.
	// This is used after the external editor saves a fixed config so that
	// the accessor reflects the new file contents.
	ReloadFromFile(path string) error
}

// CoerceValue converts a raw value (typically a string from user input) to
// the Go type expected by the target FieldType. This bridges the gap between
// text input and typed config fields.
//
// Conversion rules:
//   - String/SensitiveString: value passed through as string (or fmt.Sprint)
//   - Int: string parsed via strconv.Atoi; int/int64 passed through
//   - Bool: string parsed via strconv.ParseBool; bool passed through
//   - StringList: []string passed through; string split on comma
//   - ObjectList: passed through (no coercion)
func CoerceValue(ft FieldType, value any) (any, error) {
	switch ft {
	case String, SensitiveString:
		return coerceString(value)
	case Int:
		return coerceInt(value)
	case Bool:
		return coerceBool(value)
	case StringList:
		return coerceStringList(value)
	case ObjectList:
		return value, nil
	default:
		return nil, fmt.Errorf("unsupported field type: %s", ft)
	}
}

func coerceString(value any) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	default:
		return fmt.Sprint(value), nil
	}
}

func coerceInt(value any) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, fmt.Errorf("cannot convert %q to int: %w", v, err)
		}
		return n, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to int", value)
	}
}

func coerceBool(value any) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		b, err := strconv.ParseBool(strings.TrimSpace(v))
		if err != nil {
			return false, fmt.Errorf("cannot convert %q to bool: %w", v, err)
		}
		return b, nil
	default:
		return false, fmt.Errorf("cannot convert %T to bool", value)
	}
}

func coerceStringList(value any) ([]string, error) {
	switch v := value.(type) {
	case []string:
		return v, nil
	case string:
		if v == "" {
			return []string{}, nil
		}
		parts := strings.Split(v, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		return parts, nil
	default:
		return nil, fmt.Errorf("cannot convert %T to []string", value)
	}
}
