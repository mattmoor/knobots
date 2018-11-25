package comment

import (
	"strings"
)

func WithSuggestion(replacement string) *string {
	return WithSignature(strings.Join([]string{
		"```suggestion",
		replacement,
		"```",
	}, "\n"))
}
