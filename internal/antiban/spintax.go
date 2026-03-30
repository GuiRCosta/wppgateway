package antiban

import (
	"math/rand/v2"
	"strings"
)

// ProcessSpintax processes spintax syntax: {option1|option2|option3}
// Each group is replaced with a randomly selected option.
// Supports nested spintax.
func ProcessSpintax(text string) string {
	result := strings.Builder{}
	i := 0

	for i < len(text) {
		// Skip template variables {{var}} — they are NOT spintax
		if i+1 < len(text) && text[i] == '{' && text[i+1] == '{' {
			closeIdx := strings.Index(text[i+2:], "}}")
			if closeIdx != -1 {
				end := i + 2 + closeIdx + 2
				result.WriteString(text[i:end])
				i = end
				continue
			}
		}

		if text[i] == '{' {
			// Find matching closing brace (handle nesting)
			end := findMatchingBrace(text, i)
			if end == -1 {
				result.WriteByte(text[i])
				i++
				continue
			}

			inner := text[i+1 : end]
			options := splitOptions(inner)

			if len(options) > 0 {
				chosen := options[rand.IntN(len(options))]
				// Recursively process nested spintax
				result.WriteString(ProcessSpintax(chosen))
			}

			i = end + 1
		} else {
			result.WriteByte(text[i])
			i++
		}
	}

	return result.String()
}

// findMatchingBrace finds the index of the closing brace matching the opening at pos.
func findMatchingBrace(text string, pos int) int {
	depth := 0
	for i := pos; i < len(text); i++ {
		switch text[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// splitOptions splits the inner content by | but respects nested braces.
func splitOptions(inner string) []string {
	var options []string
	depth := 0
	start := 0

	for i := 0; i < len(inner); i++ {
		switch inner[i] {
		case '{':
			depth++
		case '}':
			depth--
		case '|':
			if depth == 0 {
				options = append(options, inner[start:i])
				start = i + 1
			}
		}
	}

	options = append(options, inner[start:])
	return options
}
