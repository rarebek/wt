package middleware

import "testing"

func TestFeatureFlags(t *testing.T) {
	ff := NewFeatureFlags()

	if ff.Enabled("dark_mode") {
		t.Error("unset flag should be false")
	}

	ff.Set("dark_mode", true)
	if !ff.Enabled("dark_mode") {
		t.Error("set flag should be true")
	}

	ff.Set("dark_mode", false)
	if ff.Enabled("dark_mode") {
		t.Error("disabled flag should be false")
	}
}
