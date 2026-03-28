package wt

import "testing"

type testMsg struct {
	Name  string
	Value int
}

func (m testMsg) Validate() error {
	return RequiredFields(m, "Name", "Value")
}

func TestValidateMessageValid(t *testing.T) {
	msg := testMsg{Name: "test", Value: 42}
	if err := ValidateMessage(msg); err != nil {
		t.Errorf("expected no error: %v", err)
	}
}

func TestValidateMessageInvalid(t *testing.T) {
	msg := testMsg{Name: "", Value: 0}
	err := ValidateMessage(msg)
	if err == nil {
		t.Error("expected error for empty fields")
	}
}

func TestValidateMessageNoValidator(t *testing.T) {
	// Type without Validate() method
	type plainMsg struct{ X int }
	err := ValidateMessage(plainMsg{})
	if err != nil {
		t.Errorf("non-validator should return nil: %v", err)
	}
}

func TestRequiredFieldsMissing(t *testing.T) {
	msg := testMsg{Name: "", Value: 0}
	err := RequiredFields(msg, "Name", "Value")
	if err == nil {
		t.Error("expected error")
	}
}

func TestRequiredFieldsPresent(t *testing.T) {
	msg := testMsg{Name: "ok", Value: 1}
	err := RequiredFields(msg, "Name", "Value")
	if err != nil {
		t.Errorf("expected no error: %v", err)
	}
}

func TestRequiredFieldsPartial(t *testing.T) {
	msg := testMsg{Name: "ok", Value: 0}
	err := RequiredFields(msg, "Name", "Value")
	if err == nil {
		t.Error("expected error for zero Value")
	}
}

func TestRequiredFieldsNonStruct(t *testing.T) {
	err := RequiredFields("not a struct", "Field")
	if err == nil {
		t.Error("expected error for non-struct")
	}
}
