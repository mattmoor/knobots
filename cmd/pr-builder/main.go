package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/google/go-github/github"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"

	"github.com/mattmoor/knobots/pkg/client"
	"github.com/mattmoor/knobots/pkg/comment"
)

var (
	username = os.Getenv("GITHUB_USERNAME")
	password = os.Getenv("GITHUB_ACCESS_TOKEN")

	owner  = flag.String("organization", "", "The Github organization to which we're sending a PR")
	repo   = flag.String("repository", "", "The Github repository to which we're sending a PR")
	branch = flag.String("branch", "master", "The branch we are building a PR against.")

	// TODO(mattmoor): Figure out how to dodge CLA bot...
	signature = &object.Signature{
		Name:  "Matt Moore (via sockpuppet)",
		Email: "mattmoor+sockpuppet@google.com",
		When:  time.Now(),
	}

	title = flag.String("title", "", "The title of the PR to send.")
	body  = flag.String("body", "", "The body of the PR to send.")
	token = flag.String("token", "", "The random token for identifying this PR's provenance.")
)

func main() {
	flag.Parse()

	// Clean up older PRs as the first thing we do so that if the latest batch of
	// changes needs nothing we don't leave old PRs around.
	err := cleanupOlderPRs(*title, *owner, *repo)
	if err != nil {
		log.Fatalf("Error cleaning up PRs: %v", err)
	}

	r, err := git.PlainOpen("/workspace")
	if err != nil {
		log.Fatalf("Error opening /workspace: %v", err)
	}

	// First, build the worktree.
	wt, err := r.Worktree()
	if err != nil {
		log.Fatalf("Error fetching worktree: %v", err)
	}

	// Check the status of the worktree, and if there aren't any changes
	// bail out we're done.
	st, err := wt.Status()
	if err != nil {
		log.Fatalf("Error fetching worktree status: %v", err)
	}
	if len(st) == 0 {
		log.Println("No changes")
		return
	}
	// Display any changed we do find: `git status --porcelain`
	log.Printf("%v", st)

	nonGopkgCount := 0
	for p := range st {
		if path.Base(p) != "Gopkg.lock" {
			nonGopkgCount++
		}
		_, err = wt.Add(p)
		if err != nil {
			log.Fatalf("Error staging %q: %v", p, err)
		}
	}
	if nonGopkgCount == 0 {
		log.Println("Only Gopkg.lock files changed (skipping PR).")
		return
	}

	commitMessage := *title + "\n\n" + *body

	// Commit the staged changes to the repo.
	if _, err := wt.Commit(commitMessage, &git.CommitOptions{Author: signature}); err != nil {
		log.Fatalf("Error committing changes: %v", err)
	}

	// We use the pod name (injected by downward API) as the
	// branch name so that it is pseudo-randomized and so that
	// we can trace opened PRs back to logs.
	branchName := os.Getenv("POD_NAME")

	// Create and checkout a new branch from the commit of the HEAD reference.
	// This should be roughly equivalent to `git checkout -b {new-branch}`
	headRef, err := r.Head()
	if err != nil {
		log.Fatalf("Error fetching workspace HEAD: %v", err)
	}
	newBranchName := plumbing.NewBranchReferenceName(branchName)
	if err := wt.Checkout(&git.CheckoutOptions{
		Hash:   headRef.Hash(),
		Branch: newBranchName,
		Create: true,
		Force:  true,
	}); err != nil {
		log.Fatalf("Error checkout out new branch: %v", err)
	}

	// Push the branch to a remote to which we have write access.
	// TODO(mattmoor): What if the fork doesn't exist, or has another name?
	remote, err := r.CreateRemote(&config.RemoteConfig{
		Name: username,
		URLs: []string{fmt.Sprintf("https://github.com/%s/%s.git", username, *repo)},
	})
	if err != nil {
		log.Fatalf("Error creating new remote: %v", err)
	}

	// Publish all local branches to the remote.
	rs := config.RefSpec(fmt.Sprintf("%s:%s", newBranchName, newBranchName))
	err = remote.Push(&git.PushOptions{
		RemoteName: username,
		RefSpecs:   []config.RefSpec{rs},
		Auth: &http.BasicAuth{
			Username: username, // This can be anything.
			Password: password,
		},
	})
	if err != nil {
		log.Fatalf("Error pushing to remote: %v", err)
	}

	ctx := context.Background()
	ghc := client.New(ctx)

	// Head has the form source-owner:branch, per the Github API docs.
	head := fmt.Sprintf("%s:%s", username, branchName)

	// Inject the token (if specified) into the body of the PR, so
	// that we can identify it's provenance.
	bodyWithToken := comment.WithSignature(*token, *body)

	pr, _, err := ghc.PullRequests.Create(ctx, *owner, *repo, &github.NewPullRequest{
		Title: title,
		// Inject a signature into the body that will help us clean up matching older PRs.
		Body: comment.WithSignature(*title, *bodyWithToken),
		Head: &head,
		Base: branch,
	})
	if err != nil {
		log.Fatalf("Error creating PR: %v", err)
	}

	log.Printf("Created PR: #%d", pr.GetNumber())
}

func cleanupOlderPRs(name, owner, repo string) error {
	ctx := context.Background()
	ghc := client.New(ctx)

	closed := "closed"
	lopt := &github.PullRequestListOptions{
		Base: *branch,
	}
	for {
		prs, resp, err := ghc.PullRequests.List(ctx, owner, repo, lopt)
		if err != nil {
			return err
		}
		for _, pr := range prs {
			if comment.HasSignature(name, pr.GetBody()) {
				_, _, err := ghc.PullRequests.Edit(ctx, owner, repo, pr.GetNumber(), &github.PullRequest{
					State: &closed,
				})
				if err != nil {
					return err
				}
			}
		}
		if resp.NextPage == 0 {
			break
		}
		lopt.Page = resp.NextPage
	}

	return nil
}
