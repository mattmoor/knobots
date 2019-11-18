package review

import (
	"context"

	"github.com/google/go-github/github"

	client "github.com/mattmoor/bindings/pkg/github"
	"github.com/mattmoor/knobots/pkg/comment"
)

func CleanupOlder(ctx context.Context, name, owner, repo string, number int) error {
	ghc, err := client.New(ctx)
	if err != nil {
		return err
	}

	var ids []int64

	lopt := &github.PullRequestListCommentsOptions{}
	for {
		comments, resp, err := ghc.PullRequests.ListComments(ctx, owner, repo, number, lopt)
		if err != nil {
			return err
		}
		for _, c := range comments {
			if comment.HasSignature(name, c.GetBody()) {
				ids = append(ids, c.GetID())
			}
		}
		if resp.NextPage == 0 {
			break
		}
		lopt.Page = resp.NextPage
	}

	for _, id := range ids {
		_, err := ghc.PullRequests.DeleteComment(ctx, owner, repo, id)
		if err != nil {
			return err
		}
	}

	return nil
}
