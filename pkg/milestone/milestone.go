package milestone

import (
	"context"

	"github.com/google/go-github/github"

	"github.com/mattmoor/knobots/pkg/client"
)

func Get(ctx context.Context, owner, repo, title string) (*github.Milestone, error) {
	ghc := client.New(ctx)

	// Walk the pages of milestones looking for one matching our title.
	lopt := &github.MilestoneListOptions{}
	for {
		ms, resp, err := ghc.Issues.ListMilestones(ctx, owner, repo, lopt)
		if err != nil {
			return nil, err
		}
		for _, m := range ms {
			if m.GetTitle() == title {
				return m, nil
			}
		}
		if resp.NextPage == 0 {
			break
		}
		lopt.Page = resp.NextPage
	}
	return nil, nil
}

func Create(ctx context.Context, owner, repo, title string) (*github.Milestone, error) {
	ghc := client.New(ctx)

	m, _, err := ghc.Issues.CreateMilestone(ctx, owner, repo, &github.Milestone{
		Title: &title,
	})
	return m, err
}

func GetOrCreate(ctx context.Context, owner, repo, title string) (*github.Milestone, error) {
	if m, err := Get(ctx, owner, repo, title); err != nil || m != nil {
		return m, err
	}
	return Create(ctx, owner, repo, title)
}
