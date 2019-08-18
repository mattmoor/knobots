package reviewrequest

import (
	"context"
	"log"

	"github.com/google/go-github/github"

	"github.com/mattmoor/knobots/pkg/handler"
)

type reviewrequest struct{}

var _ handler.Interface = (*reviewrequest)(nil)

func New() handler.Interface {
	return &reviewrequest{}
}

func (*reviewrequest) GetType() interface{} {
	return &github.PullRequestEvent{}
}

func (*reviewrequest) Handle(ctx context.Context, x interface{}) (handler.Response, error) {
	pre := x.(*github.PullRequestEvent)

	pr := pre.GetPullRequest()
	log.Printf("PR: %v", pr.String())

	// Ignore closed PRs
	if pr.GetState() == "closed" {
		return nil, nil
	}

	// Only fire on a handful of "actions".
	switch pre.GetAction() {
	case "opened", "reopened", "synchronize":
		// Fire on these.
	default:
		log.Printf("Skipping action: %s", pre.GetAction())
		return nil, nil
	}

	return &Response{
		Owner:       pre.Repo.Owner.GetLogin(),
		Repository:  pre.Repo.GetName(),
		PullRequest: pre.GetNumber(),
		Head:        pr.GetHead(),
	}, nil
}

type Response struct {
	Owner       string                    `json:"owner"`
	Repository  string                    `json:"repository"`
	PullRequest int                       `json:"pull_request"`
	Head        *github.PullRequestBranch `json:"head"`
}

var _ handler.Response = (*Response)(nil)

func (*Response) GetSource() string {
	return "https://github.com/mattmoor/knobots/cmd/reviewrequest"
}

func (*Response) GetType() string {
	return "dev.mattmoor.knobots.reviewrequest"
}
