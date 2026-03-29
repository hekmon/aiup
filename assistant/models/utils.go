package models

import "strings"

const (
	listDottedPrefix = "• "
)

func formatListDotted(items []string, indent int) string {
	return formatList(items, listDottedPrefix, indent)
}

func formatList(items []string, itemPrefix string, indent int) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return strings.Repeat(" ", indent) + items[0]
	default:
		// Prepare for multi line string building
		var (
			size    int
			builder strings.Builder
		)
		for _, item := range items {
			size += indent + len(itemPrefix) + len(item) + 1
		}
		builder.Grow(size)
		// Build output
		for index, item := range items {
			builder.WriteString(strings.Repeat(" ", indent))
			builder.WriteString(itemPrefix)
			builder.WriteString(item)
			if index != len(items)-1 {
				builder.WriteRune('\n')
			}
		}
		// Return result
		return builder.String()
	}
}
