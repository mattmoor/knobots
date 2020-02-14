package reviewresult

import (
	"context"

	"github.com/google/go-github/github"

	"github.com/mattmoor/knobots/pkg/commitstatus"
	"github.com/mattmoor/knobots/pkg/handler"
	"github.com/mattmoor/knobots/pkg/review"
)

type reviewresult struct{}

var _ handler.Interface = (*reviewresult)(nil)

func New(context.Context) handler.Interface {
	return &reviewresult{}
}

func (*reviewresult) GetType() interface{} {
	return &Payload{}
}

func (*reviewresult) Handle(ctx context.Context, x interface{}) (handler.Response, error) {
	p := x.(*Payload)

	if err := review.CleanupOlder(ctx, p.Name, p.Owner, p.Repository, p.PullRequest); err != nil {
		return nil, err
	}

	pr := &commitstatus.Payload{
		Owner:       p.Owner,
		Repository:  p.Repository,
		SHA:         p.SHA,
		Name:        p.Name,
		Description: p.Description,
	}

	// Determine the check state and write our review.
	if len(p.Comments) != 0 || p.Body != "" {
		pr.State = "failure"

		if err := review.Create(ctx, p.Name, p.Owner, p.Repository, p.PullRequest, p.Body, p.Comments); err != nil {
			return nil, err
		}
	} else {
		pr.State = "success"
	}

	return nil, nil

	// TODO(mattmoor): Don't do this for now.
	return pr, nil
}

type Payload struct {
	Owner       string `json:"owner"`
	Repository  string `json:"repository"`
	PullRequest int    `json:"pull_request"`
	SHA         string `json:"sha"`

	Name        string `json:"name"`
	Description string `json:"description"`

	Body     string                       `json:"body,omitempty"`
	Comments []*github.DraftReviewComment `json:"comments"`
}

var _ handler.Response = (*Payload)(nil)

func (*Payload) GetSource() string {
	return "https://github.com/mattmoor/knobots/cmd/reviewresult"
}

func (*Payload) GetType() string {
	return "dev.mattmoor.knobots.reviewresult"
}
