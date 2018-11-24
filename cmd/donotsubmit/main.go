package main

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/google/go-github/github"
	"sourcegraph.com/sourcegraph/go-diff/diff"

	"github.com/mattmoor/knobots/pkg/botinfo"
	"github.com/mattmoor/knobots/pkg/client"
	"github.com/mattmoor/knobots/pkg/visitor"
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
	default:
		log.Printf("Unrecognized event: %T", event)
		http.Error(w, "Unknown event", http.StatusBadRequest)
		return
	}
}

var (
	botName        = botinfo.GetName()
	botDescription = `Check for "DO NOT SUBMIT" in added lines.`
)

func HandlePullRequest(pre *github.PullRequestEvent) error {
	pr := pre.GetPullRequest()
	log.Printf("PR: %v", pr.String())

	// Ignore closed PRs
	if pr.GetState() == "closed" {
		return nil
	}

	owner, repo, number := pre.Repo.Owner.GetLogin(), pre.Repo.GetName(), pre.GetNumber()

	found := false
	err := visitor.Hunks(owner, repo, number,
		func(_ string, hunk *diff.Hunk) (visitor.VisitControl, error) {
			s := string(hunk.Body)
			lines := strings.Split(s, "\n")
			for _, line := range lines {
				if !strings.HasPrefix(line, "+") {
					continue
				}
				if strings.Contains(line, "DO NOT SUBMIT") {
					// Break after the first occurrence we find.
					// TODO(mattmoor): Track occurrence locations and comment on them.
					found = true
					return visitor.Break, nil
				}
			}
			return visitor.Continue, nil
		})
	if err != nil {
		return err
	}

	// Determine the check state.
	var state string
	if found {
		state = "failure"
	} else {
		state = "success"
	}

	sha := pre.GetPullRequest().GetHead().GetSHA()

	ctx := context.Background()
	ghc := client.New(ctx)
	_, _, err = ghc.Repositories.CreateStatus(ctx, owner, repo, sha, &github.RepoStatus{
		Context:     &botName,
		State:       &state,
		Description: &botDescription,
		// TODO(mattmoor): Consider adding a target URL for where we found the string.
	})

	return err
}
