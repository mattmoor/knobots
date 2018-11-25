package main

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/google/go-github/github"
	"sourcegraph.com/sourcegraph/go-diff/diff"

	"github.com/mattmoor/knobots/pkg/botinfo"
	"github.com/mattmoor/knobots/pkg/client"
	"github.com/mattmoor/knobots/pkg/comment"
	"github.com/mattmoor/knobots/pkg/handler"
	"github.com/mattmoor/knobots/pkg/review"
	"github.com/mattmoor/knobots/pkg/visitor"
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

	var comments []*github.DraftReviewComment
	err := visitor.Hunks(owner, repo, number,
		func(path string, hs []*diff.Hunk) (visitor.VisitControl, error) {
			// TODO(mattmoor): Base this on .gitattributes (we should build a library).
			if strings.HasPrefix(path, "vendor/") {
				return visitor.Continue, nil
			}
			// Each hunk header @@ takes a line.
			// For subsequent hunks, this is covered by the trailing `\n`
			// in each hunk, but the first needs to start at offset 1.
			offset := 1
			for _, hunk := range hs {
				lines := strings.Split(string(hunk.Body), "\n")
				for _, line := range lines {
					if strings.HasPrefix(line, "+") {
						if strings.Contains(line, "DO NOT SUBMIT") {
							position := offset // Copy it.
							comments = append(comments, &github.DraftReviewComment{
								Path:     &path,
								Position: &position,
								Body:     comment.WithSignature(`Found "DO NOT SUBMIT".`),
							})
						}
					}
					// Increase our offset for each line we see.
					offset++
				}
			}
			return visitor.Continue, nil
		})
	if err != nil {
		return err
	}

	if err := review.CleanupOlder(owner, repo, number); err != nil {
		return err
	}

	// Determine the check state.
	var state string
	if len(comments) != 0 {
		state = "failure"

		if err := review.Create(owner, repo, number, comments); err != nil {
			return err
		}
	} else {
		state = "success"
	}

	sha := pr.GetHead().GetSHA()
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
