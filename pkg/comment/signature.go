package comment

import (
	"fmt"
	"strings"
)

func sig(name string) string {
	return fmt.Sprintf("<!--%s-->", name)
}

func HasSignature(name, comment string) bool {
	return strings.Contains(comment, sig(name))
}

func WithSignature(name, comment string) *string {
	both := sig(name) + "\n" + comment
	return &both
}
