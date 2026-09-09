package content

import (
	"strings"
	"unicode"
)

// NormalizeSlug returns a stable Unicode-aware lower-case slug.
func NormalizeSlug(value string) string {
	var result strings.Builder
	pendingSeparator := false
	for _, character := range strings.ToLower(strings.TrimSpace(value)) {
		switch {
		case unicode.IsLetter(character) || unicode.IsNumber(character):
			if pendingSeparator && result.Len() > 0 {
				result.WriteByte('-')
			}
			result.WriteRune(character)
			pendingSeparator = false
		default:
			pendingSeparator = result.Len() > 0
		}
	}
	return result.String()
}
