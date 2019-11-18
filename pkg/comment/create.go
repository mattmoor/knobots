package comment

import (
	"context"

	"github.com/google/go-github/github"

	client "github.com/mattmoor/bindings/pkg/github"
	"github.com/mattmoor/knobots/pkg/botinfo"
)

func Create(ctx context.Context, owner, repo string, number int, body string) error {
	ghc, err := client.New(ctx)
	if err != nil {
		return err
	}

	_, _, err = ghc.Issues.CreateComment(ctx,
		owner, repo, number, &github.IssueComment{
			Body: WithSignature(botinfo.GetName(), body),
		})
	return err
}
