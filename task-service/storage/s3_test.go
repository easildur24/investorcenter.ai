package storage

import (
	"testing"
)

func TestValidateS3Key_Valid(t *testing.T) {
	tests := []struct {
		name   string
		s3Key  string
		taskID string
	}{
		{"simple file", "worker-results/abc-123/report.txt", "abc-123"},
		{"nested path", "worker-results/abc-123/subdir/file.pdf", "abc-123"},
		{"uuid task id", "worker-results/550e8400-e29b-41d4-a716-446655440000/analysis.txt", "550e8400-e29b-41d4-a716-446655440000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateS3Key(tt.s3Key, tt.taskID)
			if err != nil {
				t.Errorf("ValidateS3Key(%q, %q) returned error: %v", tt.s3Key, tt.taskID, err)
			}
		})
	}
}

func TestValidateS3Key_Invalid(t *testing.T) {
	tests := []struct {
		name   string
		s3Key  string
		taskID string
	}{
		{"wrong prefix", "other-prefix/abc-123/report.txt", "abc-123"},
		{"wrong task id", "worker-results/wrong-id/report.txt", "abc-123"},
		{"no prefix", "report.txt", "abc-123"},
		{"traversal attempt", "worker-results/../secret/key", "abc-123"},
		{"empty key", "", "abc-123"},
		{"missing trailing slash", "worker-results/abc-123", "abc-123"},
		{"different task in path", "worker-results/other-task/report.txt", "abc-123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateS3Key(tt.s3Key, tt.taskID)
			if err == nil {
				t.Errorf("ValidateS3Key(%q, %q) expected error but got nil", tt.s3Key, tt.taskID)
			}
		})
	}
}
