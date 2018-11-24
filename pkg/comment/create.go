package comment

import (
	"context"

	"github.com/google/go-github/github"

	"github.com/mattmoor/knobots/pkg/client"
)

func Create(owner, repo string, number int, body string) error {
	ctx := context.Background()
	ghc := client.New(ctx)

	_, _, err := ghc.Issues.CreateComment(ctx,
		owner, repo, number, &github.IssueComment{
			Body: WithSignature(body),
		})
	return err
}
