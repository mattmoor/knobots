package client

import (
	"context"
	"os"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

func New(ctx context.Context) *github.Client {
	return github.NewClient(
		oauth2.NewClient(ctx,
			oauth2.StaticTokenSource(
				&oauth2.Token{
					AccessToken: os.Getenv("GITHUB_ACCESS_TOKEN"),
				},
			),
		),
	)
}
