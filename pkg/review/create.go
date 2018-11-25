package review

import (
	"context"

	"github.com/google/go-github/github"

	"github.com/mattmoor/knobots/pkg/client"
	"github.com/mattmoor/knobots/pkg/comment"
)

var Comment = "COMMENT"

func Create(owner, repo string, number int, comments []*github.DraftReviewComment) error {
	ctx := context.Background()
	ghc := client.New(ctx)

	_, _, err := ghc.PullRequests.CreateReview(ctx, owner, repo, number,
		&github.PullRequestReviewRequest{
			Event:    &Comment,
			Body:     comment.WithSignature(""),
			Comments: comments,
		})
	return err
}
