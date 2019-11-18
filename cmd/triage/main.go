package main

import (
	"context"
	"log"
	"net/http"

	"github.com/google/go-github/github"

	client "github.com/mattmoor/bindings/pkg/github"
	"github.com/mattmoor/knobots/pkg/handler"
	"github.com/mattmoor/knobots/pkg/milestone"
)

func main() {
	http.HandleFunc("/", Handler)
	http.ListenAndServe(":8080", nil)
}

func Handler(w http.ResponseWriter, r *http.Request) {
	event := handler.ParseGithubWebhook(w, r)
	if event == nil {
		return
	}

	// The set of events here should line up with what is in
	//   config/one-time/github-source.yaml
	switch event := event.(type) {
	case *github.PullRequestEvent:
		handler.InternalError(w, event, HandlePullRequest(event))
	case *github.IssuesEvent:
		handler.InternalError(w, event, HandleIssues(event))
	}
}

func HandleIssues(ie *github.IssuesEvent) error {
	if ie.GetIssue().Milestone != nil {
		log.Printf("Issue #%v already has a milestone.", ie.GetIssue().GetNumber())
		return nil
	}
	if ie.GetIssue().GetState() == "closed" {
		log.Printf("Issue #%v is closed.", ie.GetIssue().GetNumber())
		return nil
	}

	return needsTriage(ie.Repo.Owner.GetLogin(), ie.Repo.GetName(), ie.GetIssue().GetNumber())
}

func HandlePullRequest(pre *github.PullRequestEvent) error {
	if pre.GetPullRequest().Milestone != nil {
		log.Printf("PR #%v already has a milestone.", pre.GetNumber())
		return nil
	}
	if pre.GetPullRequest().GetState() == "closed" {
		log.Printf("PR #%v is closed.", pre.GetNumber())
		return nil
	}

	return needsTriage(pre.Repo.Owner.GetLogin(), pre.Repo.GetName(), pre.GetNumber())
}

func needsTriage(owner, repo string, number int) error {
	ctx := context.Background()
	m, err := milestone.GetOrCreate(ctx, owner, repo, "Needs Triage")
	if err != nil {
		return err
	}

	ghc, err := client.New(ctx)
	if err != nil {
		return err
	}
	_, _, err = ghc.Issues.Edit(ctx, owner, repo, number, &github.IssueRequest{
		Milestone: m.Number,
	})
	return err
}
