package hwinfo

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

// Scanner buffer constants
const (
	scannerInitialBufferSize = 64 * 1024   // 64 KB initial buffer
	scannerMaxTokenSize      = 1024 * 1024 // 1 MB max line size
)

// TimestampFormat is the HWInfo CSV timestamp format (DD.M.YYYY,HH:MM:SS.mmm).
// Example: 27.3.2026,22:59:55.972
const TimestampFormat = "02.1.2006,15:04:05.999"

// FilterCSV reads a HWInfo CSV file and returns rows from the last N minutes.
//
// It uses the last timestamp found in the file as the reference point (not system time).
// This makes it work correctly with old files and avoids timezone issues.
// Rows older than (lastTimestamp - window) are filtered out.
// Reading stops when the footer is encountered (a line identical to the header).
//
// Parameters:
//   - path: Path to the HWInfo CSV file
//   - window: Time window - only the last 'window' duration of data is returned
//
// Returns:
//   - string: The complete filtered CSV content (header + filtered rows, footer excluded)
//   - error: Explicit error if parsing fails (file I/O, timestamp format, malformed lines)
//
// Timestamp format: DD.M.YYYY,HH:MM:SS.mmm (e.g., 27.3.2026,22:59:55.972)
func FilterCSV(path string, window time.Duration) (string, error) {
	// Open file
	file, err := os.Open(path)
	if err != nil {
		return "", err // os.PathError already descriptive
	}
	defer file.Close()

	// Create scanner
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, scannerInitialBufferSize), scannerMaxTokenSize)

	// Read header (line 0)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", err
		}
		return "", FilterCSVError{Line: 0, Message: "empty file"}
	}
	header := strings.TrimPrefix(scanner.Text(), "\ufeff") // Strip BOM for footer comparison

	// PASS 1: Find the last timestamp in the file (before footer)
	var lastTimestamp time.Time
	lineNum := 1

	for scanner.Scan() {
		line := scanner.Text()

		// Check for footer (any line starting with "Date,Time" after the header)
		if strings.HasPrefix(line, "Date,Time,") {
			break
		}

		// Skip empty lines
		if line == "" {
			lineNum++
			continue
		}

		// Extract first two fields (date and time)
		fields := strings.SplitN(line, ",", 3)
		if len(fields) < 2 {
			return "", FilterCSVError{
				Line:    lineNum,
				Message: "malformed line: less than 2 fields",
			}
		}

		// Parse timestamp (normalize first to handle missing leading zeros)
		timestampStr := normalizeTimestamp(fields[0] + "," + fields[1])
		timestamp, err := time.Parse(TimestampFormat, timestampStr)
		if err != nil {
			return "", FilterCSVError{
				Line:    lineNum,
				Message: fmt.Sprintf("invalid timestamp format %q", timestampStr),
				Err:     err,
			}
		}

		// Keep track of the last (most recent) timestamp
		if timestamp.After(lastTimestamp) {
			lastTimestamp = timestamp
		}

		lineNum++
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		return "", err
	}

	// Calculate cutoff: lastTimestamp - window
	cutoffTime := lastTimestamp.Add(-window)

	// Reset scanner for PASS 2
	file.Seek(0, 0)
	scanner = bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, scannerInitialBufferSize), scannerMaxTokenSize)

	// Read header again
	if !scanner.Scan() {
		return "", FilterCSVError{Line: 0, Message: "empty file"}
	}

	// Build result: start with header + newline
	var result strings.Builder
	result.WriteString(header)
	result.WriteRune('\n')

	// PASS 2: Filter lines based on cutoff time
	inWindow := false
	lineNum = 1

	for scanner.Scan() {
		line := scanner.Text()

		// Check for footer (any line starting with "Date,Time" after the header)
		if strings.HasPrefix(line, "Date,Time,") {
			break
		}

		// Skip empty lines
		if line == "" {
			lineNum++
			continue
		}

		// Optimization: once in window, all subsequent lines are valid
		if !inWindow {
			fields := strings.SplitN(line, ",", 3)
			if len(fields) < 2 {
				return "", FilterCSVError{
					Line:    lineNum,
					Message: "malformed line: less than 2 fields",
				}
			}

			timestampStr := normalizeTimestamp(fields[0] + "," + fields[1])
			timestamp, err := time.Parse(TimestampFormat, timestampStr)
			if err != nil {
				return "", FilterCSVError{
					Line:    lineNum,
					Message: fmt.Sprintf("invalid timestamp format %q", timestampStr),
					Err:     err,
				}
			}

			// Check if within time window
			if timestamp.Before(cutoffTime) {
				lineNum++
				continue // Too old, skip
			}

			inWindow = true // All subsequent lines will be valid
		}

		// Add line to result
		result.WriteString(line)
		result.WriteRune('\n')
		lineNum++
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		return "", err
	}

	return result.String(), nil
}

// FilterCSVError wraps errors that occur during CSV filtering with context.
type FilterCSVError struct {
	Line    int    // Line number where the error occurred (0-indexed, 0=header)
	Message string // Error description
	Err     error  // Underlying error (if any)
}

func (e FilterCSVError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("line %d: %s: %v", e.Line, e.Message, e.Err)
	}
	return fmt.Sprintf("line %d: %s", e.Line, e.Message)
}

func (e FilterCSVError) Unwrap() error {
	return e.Err
}

// normalizeTimestamp adds missing leading zeros to HWInfo timestamps.
// HWInfo produces inconsistent formats: "23:0:1.978" instead of "23:00:01.978"
// Input format: "DD.M.YYYY,HH:MM:SS.mmm" (possibly missing leading zeros)
// Output format: "DD.M.YYYY,HH:MM:SS.mmm" (always with leading zeros)
func normalizeTimestamp(ts string) string {
	parts := strings.SplitN(ts, ",", 2)
	if len(parts) != 2 {
		return ts
	}
	datePart := parts[0]
	timePart := parts[1]

	// Split time into H:M:S.mmm
	timeParts := strings.SplitN(timePart, ":", 3)
	if len(timeParts) != 3 {
		return ts
	}

	hours := timeParts[0]
	minutes := timeParts[1]
	secParts := strings.SplitN(timeParts[2], ".", 2)
	seconds := secParts[0]
	millis := ""
	if len(secParts) == 2 {
		millis = "." + secParts[1]
	}

	// Add leading zeros if needed
	if len(hours) == 1 {
		hours = "0" + hours
	}
	if len(minutes) == 1 {
		minutes = "0" + minutes
	}
	if len(seconds) == 1 {
		seconds = "0" + seconds
	}

	return datePart + "," + hours + ":" + minutes + ":" + seconds + millis
}
