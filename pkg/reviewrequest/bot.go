package reviewrequest

import (
	"context"
	"log"

	"github.com/google/go-github/github"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/mattmoor/knobots/pkg/handler"
)

type reviewrequest struct{}

var _ handler.Interface = (*reviewrequest)(nil)

func New(context.Context) handler.Interface {
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

	labels := sets.NewString()
	for _, l := range pr.Labels {
		labels.Insert(l.GetName())
	}

	return &Response{
		Owner:       pre.Repo.Owner.GetLogin(),
		Repository:  pre.Repo.GetName(),
		PullRequest: pre.GetNumber(),
		Head:        pr.GetHead(),
		Labels:      labels.List(),
	}, nil
}

type Response struct {
	Owner       string                    `json:"owner"`
	Repository  string                    `json:"repository"`
	PullRequest int                       `json:"pull_request"`
	Head        *github.PullRequestBranch `json:"head,omitempty"`
	Labels      []string                  `json:"labels,omitempty"`
}

var _ handler.Response = (*Response)(nil)

func (*Response) GetSource() string {
	return "https://github.com/mattmoor/knobots/cmd/reviewrequest"
}

func (*Response) GetType() string {
	return "dev.mattmoor.knobots.reviewrequest"
}
