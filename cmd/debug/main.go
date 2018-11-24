package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/google/go-github/github"

	"github.com/mattmoor/knobots/pkg/comment"
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

	owner, repo, number := ie.Repo.Owner.GetLogin(), ie.Repo.GetName(), ie.GetIssue().GetNumber()

	if err := comment.CleanupOlder(owner, repo, number); err != nil {
		return err
	}

	return comment.Create(
		owner, repo, number,
		fmt.Sprintf("Issues event: %v", ie.GetAction()))
}

func HandlePullRequest(pre *github.PullRequestEvent) error {
	log.Printf("PR: %v", pre.GetPullRequest().String())

	owner, repo, number := pre.Repo.Owner.GetLogin(), pre.Repo.GetName(), pre.GetNumber()

	if err := comment.CleanupOlder(owner, repo, number); err != nil {
		return err
	}

	return comment.Create(
		owner, repo, number,
		fmt.Sprintf("PR event: %v", pre.GetAction()))
}
