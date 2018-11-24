package main

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/google/go-github/github"
	"sourcegraph.com/sourcegraph/go-diff/diff"

	"github.com/mattmoor/knobots/pkg/client"
)

var (
	botName        = "DO NOT SUBMIT"
	botDescription = `Check for "DO NOT SUBMIT" in added lines.`
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

// TODO(mattmoor): For bonus points, return the position to comment on.
func HasDoNotSubmit(cf *github.CommitFile) bool {
	hs, err := diff.ParseHunks([]byte(cf.GetPatch()))
	if err != nil {
		log.Printf("ERROR PARSING HUNKS: %v", err)
		return false
	}

	// Search the lines of each diff "hunk" for an addition line containing
	// the words "DO NOT SUBMIT".
	for _, hunk := range hs {
		s := string(hunk.Body)
		lines := strings.Split(s, "\n")
		for _, line := range lines {
			if !strings.HasPrefix(line, "+") {
				continue
			}
			if strings.Contains(line, "DO NOT SUBMIT") {
				return true
			}
		}
	}

	return false
}

// Determine whether we need a `/hold` on this PR.
func NeedsHold(ctx context.Context, pre *github.PullRequestEvent) (bool, error) {
	ghc := client.New(ctx)

	owner, repo := pre.Repo.Owner.GetLogin(), pre.Repo.GetName()

	lopt := &github.ListOptions{}
	for {
		cfs, resp, err := ghc.PullRequests.ListFiles(ctx, owner, repo, pre.GetNumber(), lopt)
		if err != nil {
			return false, err
		}
		for _, cf := range cfs {
			if HasDoNotSubmit(cf) {
				return true, nil
			}
		}
		if lopt.Page == resp.NextPage {
			break
		}
		lopt.Page = resp.NextPage
	}

	return false, nil
}

func HandlePullRequest(pre *github.PullRequestEvent) error {
	pr := pre.GetPullRequest()
	log.Printf("PR: %v", pr.String())

	// Ignore closed PRs
	if pr.GetState() == "closed" {
		return nil
	}
	ctx := context.Background()
	ghc := client.New(ctx)

	want, err := NeedsHold(ctx, pre)
	if err != nil {
		return err
	}

	owner, repo := pre.Repo.Owner.GetLogin(), pre.Repo.GetName()

	// Determine the check state.
	var state string
	if want {
		state = "failure"
	} else {
		state = "success"
	}

	sha := pre.GetPullRequest().GetHead().GetSHA()

	_, _, err = ghc.Repositories.CreateStatus(ctx, owner, repo, sha, &github.RepoStatus{
		Context:     &botName,
		State:       &state,
		Description: &botDescription,
		// TODO(mattmoor): Consider adding a target URL for where we found the string.
	})

	return err
}
