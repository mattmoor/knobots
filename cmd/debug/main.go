package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/google/go-github/github"

	"github.com/mattmoor/knobots/pkg/botinfo"
	"github.com/mattmoor/knobots/pkg/comment"
	"github.com/mattmoor/knobots/pkg/handler"
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
	log.Printf("Issue: %v", ie.GetIssue().String())

	owner, repo, number := ie.Repo.Owner.GetLogin(), ie.Repo.GetName(), ie.GetIssue().GetNumber()

	if err := comment.CleanupOlder(context.Background(), botinfo.GetName(), owner, repo, number); err != nil {
		return err
	}

	return comment.Create(
		context.Background(),
		owner, repo, number,
		fmt.Sprintf("Issues event: %v", ie.GetAction()))
}

func HandlePullRequest(pre *github.PullRequestEvent) error {
	log.Printf("PR: %v", pre.GetPullRequest().String())

	owner, repo, number := pre.Repo.Owner.GetLogin(), pre.Repo.GetName(), pre.GetNumber()

	if err := comment.CleanupOlder(context.Background(), botinfo.GetName(), owner, repo, number); err != nil {
		return err
	}

	return comment.Create(
		context.Background(),
		owner, repo, number,
		fmt.Sprintf("PR event: %v", pre.GetAction()))
}
