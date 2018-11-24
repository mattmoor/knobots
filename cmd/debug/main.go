package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/google/go-github/github"

	"github.com/mattmoor/knobots/pkg/client"
)

func main() {
	http.HandleFunc("/", Handler)
	http.ListenAndServe(":8080", nil)
}

func Handler(w http.ResponseWriter, r *http.Request) {
	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("ERROR: no payload: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO(mattmoor): This should be:
	//     eventType := github.WebHookType(r)
	// https://github.com/knative/eventing-sources/issues/120
	// HACK HACK HACK
	eventType := strings.Split(r.Header.Get("ce-eventtype"), ".")[4]

	event, err := github.ParseWebHook(eventType, payload)
	if err != nil {
		log.Printf("ERROR: unable to parse webhook: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// The set of events here should line up with what is in
	//   config/one-time/github-source.yaml
	switch event := event.(type) {
	case *github.PullRequestEvent:
		if err := HandlePullRequest(event); err != nil {
			log.Printf("Error handling %T: %v", event, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	case *github.IssuesEvent:
		if err := HandleIssues(event); err != nil {
			log.Printf("Error handling %T: %v", event, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	default:
		log.Printf("Unrecognized event: %T", event)
		http.Error(w, "Unknown event", http.StatusBadRequest)
		return
	}
}

func HandleIssues(ie *github.IssuesEvent) error {
	log.Printf("Issue: %v", ie.GetIssue().String())

	ctx := context.Background()
	ghc := client.New(ctx)

	msg := fmt.Sprintf("Issues event: %v", ie.GetAction())
	_, _, err := ghc.Issues.CreateComment(ctx,
		ie.Repo.Owner.GetLogin(), ie.Repo.GetName(), ie.GetIssue().GetNumber(),
		&github.IssueComment{
			Body: &msg,
		})
	return err
}

func HandlePullRequest(pre *github.PullRequestEvent) error {
	log.Printf("PR: %v", pre.GetPullRequest().String())

	ctx := context.Background()
	ghc := client.New(ctx)

	msg := fmt.Sprintf("PR event: %v", pre.GetAction())
	_, _, err := ghc.Issues.CreateComment(ctx,
		pre.Repo.Owner.GetLogin(), pre.Repo.GetName(), pre.GetNumber(),
		&github.IssueComment{
			Body: &msg,
		})
	return err
}
