package review

import (
	"context"

	"github.com/google/go-github/github"

	"github.com/mattmoor/knobots/pkg/client"
	"github.com/mattmoor/knobots/pkg/comment"
)

var Comment = "COMMENT"

func Create(ctx context.Context, name, owner, repo string, number int, body string, comments []*github.DraftReviewComment) error {
	ghc := client.New(ctx)

	_, _, err := ghc.PullRequests.CreateReview(ctx, owner, repo, number,
		&github.PullRequestReviewRequest{
			Event:    &Comment,
			Body:     comment.WithSignature(name, body),
			Comments: comments,
		})
	return err
}
