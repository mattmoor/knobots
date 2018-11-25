package review

import (
	"context"

	"github.com/google/go-github/github"

	"github.com/mattmoor/knobots/pkg/client"
	"github.com/mattmoor/knobots/pkg/comment"
)

func CleanupOlder(owner, repo string, number int) error {
	ctx := context.Background()
	ghc := client.New(ctx)

	var ids []int64

	lopt := &github.PullRequestListCommentsOptions{}
	for {
		comments, resp, err := ghc.PullRequests.ListComments(ctx, owner, repo, number, lopt)
		if err != nil {
			return err
		}
		for _, c := range comments {
			if comment.HasSignature(c.GetBody()) {
				ids = append(ids, c.GetID())
			}
		}
		if lopt.Page == resp.NextPage {
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
