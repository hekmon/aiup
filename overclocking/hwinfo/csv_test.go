package hwinfo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestNormalizeTimestamp tests the normalizeTimestamp function with various input formats
func TestNormalizeTimestamp(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Standard format with leading zeros",
			input:    "27.3.2026,23:05:09.972",
			expected: "27.3.2026,23:05:09.972",
		},
		{
			name:     "Missing leading zero in minutes",
			input:    "27.3.2026,23:5:09.972",
			expected: "27.3.2026,23:05:09.972",
		},
		{
			name:     "Missing leading zeros in minutes and seconds",
			input:    "27.3.2026,23:5:9.972",
			expected: "27.3.2026,23:05:09.972",
		},
		{
			name:     "Missing leading zero in hour",
			input:    "27.3.2026,5:05:09.972",
			expected: "27.3.2026,05:05:09.972",
		},
		{
			name:     "All missing leading zeros",
			input:    "7.3.2026,5:5:9.972",
			expected: "7.3.2026,05:05:09.972", // Only time components are normalized, not date
		},
		{
			name:     "Single digit hour minute second",
			input:    "27.3.2026,0:0:1.978",
			expected: "27.3.2026,00:00:01.978",
		},
		{
			name:     "No milliseconds",
			input:    "27.3.2026,23:5:9",
			expected: "27.3.2026,23:05:09",
		},
		{
			name:     "Already normalized",
			input:    "27.3.2026,22:59:55.972",
			expected: "27.3.2026,22:59:55.972",
		},
		{
			name:     "Invalid format - no comma",
			input:    "27.3.202623:05:09.972",
			expected: "27.3.202623:05:09.972", // Should return unchanged
		},
		{
			name:     "Invalid format - empty",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeTimestamp(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeTimestamp(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestFilterCSV tests the FilterCSV function with various scenarios
func TestFilterCSV(t *testing.T) {
	t.Run("Basic filtering - keep last N minutes", func(t *testing.T) {
		// Create test file with timestamps spanning 10 minutes
		content := `Date,Time,Value
27.3.2026,23:0:0.000,old1
27.3.2026,23:2:0.000,old2
27.3.2026,23:5:0.000,old3
27.3.2026,23:8:0.000,new1
27.3.2026,23:9:0.000,new2
27.3.2026,23:10:0.000,new3
Date,Time,Value` // Footer

		tmpFile := createTempFile(t, content)
		defer os.Remove(tmpFile)

		// Use 3 minute window - should keep only last 3 entries (23:8, 23:9, 23:10)
		result, err := FilterCSV(tmpFile, 3*time.Minute)
		if err != nil {
			t.Fatalf("FilterCSV failed: %v", err)
		}

		lines := strings.Split(result, "\n")
		// Should have: header + 3 data lines + empty line from trailing newline = 4-5 lines
		// Count non-empty lines
		nonEmptyLines := 0
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				nonEmptyLines++
			}
		}

		if nonEmptyLines < 4 {
			t.Errorf("Expected at least 4 non-empty lines (header + 3 data), got %d", nonEmptyLines)
		}

		// Verify old entries are filtered out
		if strings.Contains(result, "old1") || strings.Contains(result, "old2") || strings.Contains(result, "old3") {
			t.Error("Old entries should be filtered out")
		}

		// Verify new entries are kept
		if !strings.Contains(result, "new1") || !strings.Contains(result, "new2") || !strings.Contains(result, "new3") {
			t.Error("New entries should be kept")
		}
	})

	t.Run("Empty file", func(t *testing.T) {
		tmpFile := createTempFile(t, "")
		defer os.Remove(tmpFile)

		_, err := FilterCSV(tmpFile, 1*time.Minute)
		if err == nil {
			t.Error("Expected error for empty file")
		}
		if !strings.Contains(err.Error(), "empty file") {
			t.Errorf("Expected 'empty file' error, got: %v", err)
		}
	})

	t.Run("Malformed line - less than 2 fields", func(t *testing.T) {
		content := `Date,Time,Value
27.3.2026,23:10:0.000,ok
singlefield
Date,Time,Value`
		tmpFile := createTempFile(t, content)
		defer os.Remove(tmpFile)

		_, err := FilterCSV(tmpFile, 1*time.Minute)
		if err == nil {
			t.Error("Expected error for malformed line")
		}
		if !strings.Contains(err.Error(), "less than 2 fields") {
			t.Errorf("Expected 'less than 2 fields' error, got: %v", err)
		}
	})

	t.Run("Invalid timestamp format", func(t *testing.T) {
		content := `Date,Time,Value
invalid,date,ok
Date,Time,Value`
		tmpFile := createTempFile(t, content)
		defer os.Remove(tmpFile)

		_, err := FilterCSV(tmpFile, 1*time.Minute)
		if err == nil {
			t.Error("Expected error for invalid timestamp")
		}
		if !strings.Contains(err.Error(), "invalid timestamp format") {
			t.Errorf("Expected 'invalid timestamp format' error, got: %v", err)
		}
	})

	t.Run("Footer detection by prefix", func(t *testing.T) {
		// Footer with different length than header (real-world case)
		content := `Date,Time,Value
27.3.2026,23:10:0.000,data
27.3.2026,23:10:30.000,data2
Date,Time,Value,"Extra Column"` // Different footer

		tmpFile := createTempFile(t, content)
		defer os.Remove(tmpFile)

		result, err := FilterCSV(tmpFile, 10*time.Minute)
		if err != nil {
			t.Fatalf("FilterCSV failed: %v", err)
		}

		// Should not contain footer
		if strings.Contains(result, "Extra Column") {
			t.Error("Footer should not be included in result")
		}
	})

	t.Run("BOM handling", func(t *testing.T) {
		// File with BOM at start
		content := "\ufeffDate,Time,Value\n27.3.2026,23:10:0.000,data\nDate,Time,Value"
		tmpFile := createTempFile(t, content)
		defer os.Remove(tmpFile)

		result, err := FilterCSV(tmpFile, 10*time.Minute)
		if err != nil {
			t.Fatalf("FilterCSV failed: %v", err)
		}

		// Result should not contain BOM
		if strings.HasPrefix(result, "\ufeff") {
			t.Error("Result should not contain BOM")
		}

		// Should contain the header without BOM
		if !strings.HasPrefix(result, "Date,Time,Value") {
			t.Error("Result should start with header")
		}
	})

	t.Run("All lines within window", func(t *testing.T) {
		content := `Date,Time,Value
27.3.2026,23:10:0.000,data1
27.3.2026,23:10:30.000,data2
27.3.2026,23:10:59.000,data3
Date,Time,Value`
		tmpFile := createTempFile(t, content)
		defer os.Remove(tmpFile)

		result, err := FilterCSV(tmpFile, 10*time.Minute)
		if err != nil {
			t.Fatalf("FilterCSV failed: %v", err)
		}

		// All data lines should be kept
		if !strings.Contains(result, "data1") || !strings.Contains(result, "data2") || !strings.Contains(result, "data3") {
			t.Error("All data lines should be kept when within window")
		}
	})

	t.Run("Empty lines are skipped", func(t *testing.T) {
		content := `Date,Time,Value
27.3.2026,23:10:0.000,data1

27.3.2026,23:10:30.000,data2
Date,Time,Value`
		tmpFile := createTempFile(t, content)
		defer os.Remove(tmpFile)

		result, err := FilterCSV(tmpFile, 10*time.Minute)
		if err != nil {
			t.Fatalf("FilterCSV failed: %v", err)
		}

		// Should not error on empty lines
		if !strings.Contains(result, "data1") || !strings.Contains(result, "data2") {
			t.Error("Data lines should be kept")
		}
	})
}

// createTempFile creates a temporary file with the given content and returns its path
func createTempFile(t *testing.T, content string) string {
	t.Helper()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.csv")

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	return tmpFile
}

// TestFilterCSVFileNotFound tests that FilterCSV returns an error for non-existent files
func TestFilterCSVFileNotFound(t *testing.T) {
	_, err := FilterCSV("/nonexistent/path/file.csv", 1*time.Minute)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}
