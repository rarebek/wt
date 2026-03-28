package wt

import (
	"fmt"
	"reflect"
	"strings"
)

// Validator can validate itself. Implement on message types for automatic validation.
type Validator interface {
	Validate() error
}

// ValidateMessage checks if a decoded message implements Validator and validates it.
// Returns nil if the message doesn't implement Validator.
func ValidateMessage(msg any) error {
	if v, ok := msg.(Validator); ok {
		return v.Validate()
	}
	return nil
}

// RequiredFields checks that the given struct fields are non-zero.
// Useful for simple message validation without a full validation library.
//
// Usage:
//
//	type ChatMsg struct {
//	    User string
//	    Text string
//	}
//	func (m ChatMsg) Validate() error {
//	    return wt.RequiredFields(m, "User", "Text")
//	}
func RequiredFields(v any, fields ...string) error {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("wt: RequiredFields expects a struct, got %T", v)
	}

	var missing []string
	for _, name := range fields {
		f := val.FieldByName(name)
		if !f.IsValid() {
			missing = append(missing, name)
			continue
		}
		if f.IsZero() {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("wt: required fields missing: %s", strings.Join(missing, ", "))
	}
	return nil
}
