package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hekmon/aiup/overclocking/hwinfo"
)

func main() {
	// Parse flags
	csvPath := flag.String("csv", "", "Path to HWInfo CSV file (required)")
	windowStr := flag.String("window", "3m", "Time window (e.g., 3m, 15m, 1h)")
	verbose := flag.Bool("verbose", false, "Show filtered lines")
	flag.Parse()

	if *csvPath == "" {
		fmt.Fprintln(os.Stderr, "Error: -csv flag is required")
		flag.Usage()
		os.Exit(1)
	}

	// Parse window duration
	window, err := time.ParseDuration(*windowStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing window duration: %v\n", err)
		os.Exit(1)
	}

	// Get original file size
	fileInfo, err := os.Stat(*csvPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting file info: %v\n", err)
		os.Exit(1)
	}
	originalSize := fileInfo.Size()

	// Count original lines
	originalLines, err := countLines(*csvPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error counting lines: %v\n", err)
		os.Exit(1)
	}

	// Filter CSV
	filteredContent, err := hwinfo.FilterCSV(*csvPath, window)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error filtering CSV: %v\n", err)
		os.Exit(1)
	}

	// Calculate filtered stats
	filteredSize := int64(len(filteredContent))
	filteredLines := strings.Count(filteredContent, "\n")

	// Extract last timestamp from filtered content for display
	lastTimestamp := extractLastTimestamp(filteredContent)

	// Display results
	fmt.Println("=== HWInfo CSV Filter Results ===")
	fmt.Printf("File:          %s\n", *csvPath)
	fmt.Printf("Window:        %s\n", window)
	if lastTimestamp != "" {
		fmt.Printf("Reference:     %s (last timestamp in file)\n", lastTimestamp)
	}
	fmt.Println()
	fmt.Println("--- Size ---")
	fmt.Printf("Original:      %d bytes\n", originalSize)
	fmt.Printf("Filtered:      %d bytes\n", filteredSize)
	fmt.Printf("Reduction:     %.1f%%\n", float64(originalSize-filteredSize)/float64(originalSize)*100)
	fmt.Println()
	fmt.Println("--- Lines ---")
	fmt.Printf("Original:      %d lines\n", originalLines)
	fmt.Printf("Filtered:      %d lines\n", filteredLines)
	fmt.Printf("Reduction:     %.1f%%\n", float64(originalLines-filteredLines)/float64(originalLines)*100)

	// Show filtered content if verbose
	if *verbose {
		fmt.Println()
		fmt.Println("--- Filtered Content ---")
		fmt.Println(filteredContent)
	}
}

// extractLastTimestamp extracts the last timestamp line from filtered content
func extractLastTimestamp(content string) string {
	lines := strings.Split(content, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if len(lines[i]) > 10 && strings.Contains(lines[i], ",") {
			parts := strings.SplitN(lines[i], ",", 3)
			if len(parts) >= 2 {
				return parts[0] + "," + parts[1]
			}
		}
	}
	return ""
}

// countLines counts the number of lines in a file
func countLines(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	count := 0
	buf := make([]byte, 32*1024)
	for {
		n, err := file.Read(buf)
		if n > 0 {
			for _, b := range buf[:n] {
				if b == '\n' {
					count++
				}
			}
		}
		if err != nil {
			break
		}
	}
	return count, nil
}
