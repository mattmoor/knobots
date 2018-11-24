package milestone

import (
	"context"

	"github.com/google/go-github/github"

	"github.com/mattmoor/knobots/pkg/client"
)

func Get(owner, repo, title string) (*github.Milestone, error) {
	ctx := context.Background()
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
		if lopt.Page == resp.NextPage {
			break
		}
		lopt.Page = resp.NextPage
	}
	return nil, nil
}

func Create(owner, repo, title string) (*github.Milestone, error) {
	ctx := context.Background()
	ghc := client.New(ctx)

	m, _, err := ghc.Issues.CreateMilestone(ctx, owner, repo, &github.Milestone{
		Title: &title,
	})
	return m, err
}

func GetOrCreate(owner, repo, title string) (*github.Milestone, error) {
	if m, err := Get(owner, repo, title); err != nil || m != nil {
		return m, err
	}
	return Create(owner, repo, title)
}
