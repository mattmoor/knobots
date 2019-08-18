package main

import (
	"context"

	"github.com/google/go-github/github"

	"github.com/mattmoor/knobots/pkg/handler"
)

type Thing struct{}

var _ handler.Interface = (*Thing)(nil)

func (t *Thing) GetType() interface{} {
	return github.PullRequestEvent{}
}

func (t *Thing) Handle(ctx context.Context, x interface{}) (handler.Response, error) {
	pre := x.(github.PullRequestEvent)

	pr := pre.GetPullRequest()
	if pr.GetState() == "closed" {
		return nil, nil
	}

	return &ThingResponse{PullRequest: pr.GetNumber()}, nil
}

type ThingResponse struct {
	PullRequest int
}

var _ handler.Response = (*ThingResponse)(nil)

func (tr *ThingResponse) GetSource() string {
	return "https://github.com/mattmoor/knobots/cmd/basic-cloud-event"
}

func (tr *ThingResponse) GetType() string {
	return "dev.knative.source.github.pull_request"
}

func main() {
	handler.Main(&Thing{})
}
