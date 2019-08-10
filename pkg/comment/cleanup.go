package comment

import (
	"context"

	"github.com/google/go-github/github"

	"github.com/mattmoor/knobots/pkg/client"
)

func CleanupOlder(name, owner, repo string, number int) error {
	ctx := context.Background()
	ghc := client.New(ctx)

	var ids []int64

	lopt := &github.IssueListCommentsOptions{}
	for {
		comments, resp, err := ghc.Issues.ListComments(ctx, owner, repo, number, lopt)
		if err != nil {
			return err
		}
		for _, comment := range comments {
			if HasSignature(name, comment.GetBody()) {
				ids = append(ids, comment.GetID())
			}
		}
		if lopt.Page == resp.NextPage {
			break
		}
		lopt.Page = resp.NextPage
	}

	for _, id := range ids {
		_, err := ghc.Issues.DeleteComment(ctx, owner, repo, id)
		if err != nil {
			return err
		}
	}

	return nil
}
