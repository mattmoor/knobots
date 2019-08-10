package reviewresult

import (
	"github.com/google/go-github/github"

	"github.com/mattmoor/knobots/pkg/commitstatus"
	"github.com/mattmoor/knobots/pkg/handler"
	"github.com/mattmoor/knobots/pkg/review"
)

type reviewresult struct{}

var _ handler.Interface = (*reviewresult)(nil)

func New() handler.Interface {
	return &reviewresult{}
}

func (*reviewresult) GetType() interface{} {
	return &Payload{}
}

func (*reviewresult) Handle(x interface{}) (handler.Response, error) {
	p := x.(*Payload)

	if err := review.CleanupOlder(p.Name, p.Owner, p.Repository, p.PullRequest); err != nil {
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
	if len(p.Comments) != 0 {
		pr.State = "failure"

		if err := review.Create(p.Name, p.Owner, p.Repository, p.PullRequest, p.Comments); err != nil {
			return nil, err
		}
	} else {
		pr.State = "success"
	}

	return pr, nil
}

type Payload struct {
	Owner       string `json:"owner"`
	Repository  string `json:"repository"`
	PullRequest int    `json:"pull_request"`
	SHA         string `json:"sha"`

	Name        string `json:"name"`
	Description string `json:"description"`

	Comments []*github.DraftReviewComment `json:"comments"`
}

var _ handler.Response = (*Payload)(nil)

func (*Payload) GetSource() string {
	return "https://github.com/mattmoor/knobots/cmd/reviewresult"
}

func (*Payload) GetType() string {
	return "dev.mattmoor.knobots.reviewresult"
}
