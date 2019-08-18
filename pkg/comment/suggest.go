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

func WithCaptionedSuggestion(caption, replacement string) *string {
	return WithSignature(botinfo.GetName(), strings.Join([]string{
		caption,
		"```suggestion",
		replacement,
		"```",
	}, "\n"))
}
