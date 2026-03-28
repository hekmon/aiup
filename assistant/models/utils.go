package models

import "strings"

const (
	listDottedPrefix = "• "
)

func formatListDotted(items []string, indent int) string {
	// Prepare
	var (
		size    int
		builder strings.Builder
	)
	for _, item := range items {
		size += indent + len(listDottedPrefix) + len(item) + 1
	}
	builder.Grow(size)
	// Build output
	for index, item := range items {
		builder.WriteString(strings.Repeat(" ", indent))
		builder.WriteString(listDottedPrefix)
		builder.WriteString(item)
		if index != len(items)-1 {
			builder.WriteRune('\n')
		}
	}
	// Return result
	return builder.String()
}
