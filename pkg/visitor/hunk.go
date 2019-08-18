package visitor

import (
	"context"

	"github.com/google/go-github/github"
	"sourcegraph.com/sourcegraph/go-diff/diff"

	"github.com/mattmoor/knobots/pkg/client"
)

type HunkCallback func(filename string, hunks []*diff.Hunk) (VisitControl, error)

func Hunks(ctx context.Context, owner, repo string, num int, v HunkCallback) error {
	ghc := client.New(ctx)

	lopt := &github.ListOptions{}
	for {
		cfs, resp, err := ghc.PullRequests.ListFiles(ctx, owner, repo, num, lopt)
		if err != nil {
			return err
		}
		for _, cf := range cfs {
			if vc, err := visitHunks(cf, v); err != nil {
				return err
			} else if vc == Break {
				return nil
			}
		}
		if resp.NextPage == 0 {
			break
		}
		lopt.Page = resp.NextPage
	}

	return nil
}

func visitHunks(cf *github.CommitFile, hc HunkCallback) (VisitControl, error) {
	hs, err := diff.ParseHunks([]byte(cf.GetPatch()))
	if err != nil {
		return Break, err
	}

	return hc(cf.GetFilename(), hs)
}
