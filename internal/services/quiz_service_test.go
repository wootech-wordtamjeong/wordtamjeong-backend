package services

import "testing"

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		defaultValue bool
		want         bool
	}{
		{name: "unset uses true default", defaultValue: true, want: true},
		{name: "unset uses false default", defaultValue: false, want: false},
		{name: "true", value: "true", want: true},
		{name: "enabled with whitespace", value: " YES ", want: true},
		{name: "false", value: "false", defaultValue: true, want: false},
		{name: "disabled", value: "off", defaultValue: true, want: false},
		{name: "invalid uses default", value: "invalid", defaultValue: true, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const key = "TEST_ENABLE_INTERNAL_SCHEDULER"
			t.Setenv(key, tt.value)

			if got := getEnvBool(key, tt.defaultValue); got != tt.want {
				t.Fatalf("getEnvBool() = %t, want %t", got, tt.want)
			}
		})
	}
}
