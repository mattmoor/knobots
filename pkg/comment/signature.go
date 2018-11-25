package comment

import (
	"fmt"
	"strings"

	"github.com/mattmoor/knobots/pkg/botinfo"
)

// This should be unique per bot.
var signature = fmt.Sprintf("<!--%s-->", botinfo.GetName())

func HasSignature(comment string) bool {
	return strings.Contains(comment, signature)
}

func WithSignature(comment string) *string {
	both := signature + "\n" + comment
	return &both
}
