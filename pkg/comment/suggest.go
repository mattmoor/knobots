package comment

import (
	"strings"

	"github.com/mattmoor/knobots/pkg/botinfo"
)

func WithSuggestion(replacement string) *string {
	return WithSignature(botinfo.GetName(), strings.Join([]string{
		"```suggestion",
		replacement,
		"```",
	}, "\n"))
}
