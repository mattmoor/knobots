package commitstatus

import (
	"context"

	"github.com/google/go-github/github"

	githubbinding "github.com/mattmoor/bindings/pkg/github"
	"github.com/mattmoor/knobots/pkg/handler"
)

type commitstatus struct{}

var _ handler.Interface = (*commitstatus)(nil)

func New() handler.Interface {
	return &commitstatus{}
}

func (*commitstatus) GetType() interface{} {
	return &Payload{}
}

func (*commitstatus) Handle(ctx context.Context, x interface{}) (handler.Response, error) {
	p := x.(*Payload)

	ghc, err := githubbinding.New(ctx)
	if err != nil {
		return nil, err
	}

	_, _, err = ghc.Repositories.CreateStatus(ctx, p.Owner, p.Repository, p.SHA, &github.RepoStatus{
		Context:     &p.Name,
		State:       &p.State,
		Description: &p.Description,
		TargetURL:   p.URL,
	})

	return nil, err
}

type Payload struct {
	Owner      string `json:"owner"`
	Repository string `json:"repository"`
	SHA        string `json:"sha"`

	Name        string  `json:"name"`
	Description string  `json:"description"`
	State       string  `json:"state"`
	URL         *string `json:"url"`
}

var _ handler.Response = (*Payload)(nil)

func (*Payload) GetSource() string {
	return "https://github.com/mattmoor/knobots/cmd/commitstatus"
}

func (*Payload) GetType() string {
	return "dev.mattmoor.knobots.commitstatus"
}
