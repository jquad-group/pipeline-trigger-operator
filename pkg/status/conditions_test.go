package apis

import (
	"testing"
)

func TestConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"ReconcileUnknown", ReconcileUnknown, "Unknown"},
		{"ReconcileError", ReconcileError, "Error"},
		{"ReconcileErrorReason", ReconcileErrorReason, "Failed"},
		{"ReconcileSuccess", ReconcileSuccess, "Success"},
		{"ReconcileSuccessReason", ReconcileSuccessReason, "Succeded"},
		{"ReconcileInProgress", ReconcileInProgress, "InProgress"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.constant != test.expected {
				t.Errorf("Expected %s to be %s, but got %s", test.name, test.expected, test.constant)
			}
		})
	}
}
