package whitespace

import (
	"context"
	"strings"
	"unicode"

	"github.com/google/go-github/github"
	"sourcegraph.com/sourcegraph/go-diff/diff"

	"github.com/mattmoor/knobots/pkg/botinfo"
	"github.com/mattmoor/knobots/pkg/comment"
	"github.com/mattmoor/knobots/pkg/handler"
	"github.com/mattmoor/knobots/pkg/reviewrequest"
	"github.com/mattmoor/knobots/pkg/reviewresult"
	"github.com/mattmoor/knobots/pkg/visitor"
)

type whitespace struct{}

var _ handler.Interface = (*whitespace)(nil)

func New() handler.Interface {
	return &whitespace{}
}

func (*whitespace) GetType() interface{} {
	return &reviewrequest.Response{}
}

func (*whitespace) Handle(ctx context.Context, x interface{}) (handler.Response, error) {
	rrr := x.(*reviewrequest.Response)

	var comments []*github.DraftReviewComment
	err := visitor.Hunks(ctx, rrr.Owner, rrr.Repository, rrr.PullRequest,
		func(path string, hs []*diff.Hunk) (visitor.VisitControl, error) {
			// TODO(mattmoor): Base this on .gitattributes (we should build a library).
			if strings.HasPrefix(path, "vendor/") {
				return visitor.Continue, nil
			}
			// Each hunk header @@ takes a line.
			// For subsequent hunks, this is covered by the trailing `\n`
			// in each hunk, but the first needs to start at offset 1.
			offset := 1
			lastSeen := ""
			for _, hunk := range hs {
				lines := strings.Split(string(hunk.Body), "\n")
				for _, line := range lines {
					lastSeen = line
					// Increase our offset for each line we see.
					if strings.HasPrefix(line, "+") {
						orig := line[1:]
						updated := strings.TrimRightFunc(orig, unicode.IsSpace)
						if updated != orig {
							position := offset // Copy it because of &.
							comments = append(comments, &github.DraftReviewComment{
								Path:     &path,
								Position: &position,
								Body: comment.WithCaptionedSuggestion(
									"Remove trailing whitespace:",
									updated,
								),
							})
						}
					}
					// Increase our offset for each line we see.
					offset++
				}
			}
			offset--

			// If the last offset is the first line in the file, and it looks
			// like a path, then don't complain as this is very likely a symlink.
			if offset == 1 {
				if strings.HasPrefix(lastSeen, "+../") {
					return visitor.Continue, nil
				}
			}

			// Check if the last line was added, but wasn't a newline.
			// This signifies that the file has a new line at the end of the file,
			// which doesn't have a trailing newline.
			if strings.HasPrefix(lastSeen, "+") && lastSeen != "+" {
				position := offset // Copy it because of &.
				comments = append(comments, &github.DraftReviewComment{
					Path:     &path,
					Position: &position,
					Body: comment.WithCaptionedSuggestion(
						"Add trailing newline:",
						lastSeen[1:]+"\n",
					),
				})
			}

			return visitor.Continue, nil
		})
	if err != nil {
		return nil, err
	}

	return &reviewresult.Payload{
		Name:        botinfo.GetName(),
		Description: `Check for whitespace issues in PRs.`,
		Owner:       rrr.Owner,
		Repository:  rrr.Repository,
		PullRequest: rrr.PullRequest,
		SHA:         rrr.Head.GetSHA(),
		Comments:    comments,
	}, nil
}
