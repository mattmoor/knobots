package comment

import (
	"context"

	"github.com/google/go-github/github"

	"github.com/mattmoor/knobots/pkg/botinfo"
	"github.com/mattmoor/knobots/pkg/client"
)

func Create(ctx context.Context, owner, repo string, number int, body string) error {
	ghc := client.New(ctx)

	_, _, err := ghc.Issues.CreateComment(ctx,
		owner, repo, number, &github.IssueComment{
			Body: WithSignature(botinfo.GetName(), body),
		})
	return err
}
