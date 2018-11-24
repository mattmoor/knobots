package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/go-github/github"

	"github.com/mattmoor/knobots/pkg/client"
	"github.com/mattmoor/knobots/pkg/comment"
	"github.com/mattmoor/knobots/pkg/handler"
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
	case *github.IssuesEvent:
		handler.InternalError(w, event, HandleIssues(event))
	}
}

func FixesRegexp(owner, repo string) *regexp.Regexp {
	// The following are all legitimate ways to close an issue via a PR or commit message:
	//   Fixes: #1234
	//   fixes https://github.com/{owner}/{repo}/issues/1234

	// Use the owner/repo to construct the base URL for issue URLs within this repo.
	issuesURL := fmt.Sprintf("https://github.com/%s/%s/issues/", owner, repo)

	// Allow either capitalization of "Fixes" with an optional trailing colon.
	keywordPattern := "[Ff]ixes[:]?"

	// TODO(mattmoor): Do better than this.
	whitespacePattern := "[ ]+"

	// Allow either a '#' or the issues URL (above) to precede a sequence of digits,
	// which is our issue number that we capture.
	issuePattern := fmt.Sprintf("(?:%s|%s)([0-9]+)",
		regexp.QuoteMeta("#"),
		regexp.QuoteMeta(issuesURL))

	return regexp.MustCompile(
		// Match the keyword followed by whitespace followed by an issue.
		keywordPattern + whitespacePattern + issuePattern,
	)
}

func GetFixedIssues(owner, repo string, pr *github.PullRequest) ([]int, error) {
	reFixes := FixesRegexp(owner, repo)

	matches := reFixes.FindAllSubmatch([]byte(pr.GetBody()), -1)
	if matches == nil {
		return nil, nil
	}

	var issues []int
	for _, match := range matches {
		// Shed the first index because it is the full match.
		for _, capture := range match[1:] {
			text := string(capture)
			num, err := strconv.ParseInt(text, 10, 32)
			if err != nil {
				return nil, err
			}
			issues = append(issues, int(num))
		}
	}

	// TODO(mattmoor): Check commit messages as well.

	return issues, nil
}

func IssueRegexp(issues []int) *regexp.Regexp {
	var asStrings []string
	for _, iss := range issues {
		asStrings = append(asStrings, fmt.Sprintf("%d", iss))
	}

	// Expect issues to have the form:
	//   TODO(#1234):
	return regexp.MustCompile(strings.Join([]string{
		regexp.QuoteMeta("TODO(#"),
		"(", strings.Join(asStrings, "|"), ")",
		regexp.QuoteMeta("):"),
	}, ""))
}

type match struct {
	filename string
	text     string
}

func FindIssueTodos(owner, repo, sha string, issues []int) ([]match, error) {
	reIssue := IssueRegexp(issues)

	var hits []match
	err := visitor.Files(owner, repo, sha,
		func(filename string, reader io.Reader) (visitor.VisitControl, error) {
			// TODO(mattmoor): Base this on .gitattributes (we should build a library).
			if strings.HasPrefix(filename, "vendor/") {
				return visitor.Continue, nil
			}
			body, err := ioutil.ReadAll(reader)
			if err != nil {
				return visitor.Break, err
			}
			// TODO(mattmoor): We should group the output by file and make it clear that
			// we only show the first 5 results per file.
			ms := reIssue.FindAll(body, 5)
			if ms == nil {
				return visitor.Continue, nil
			}
			for _, m := range ms {
				hits = append(hits, match{
					filename: filename,
					// TODO(mattmoor): Find a way to include position to comment on it?
					text: string(m),
				})
			}
			return visitor.Continue, nil
		})
	if err != nil {
		return nil, err
	}
	return hits, nil
}

func CommentWithProlog(prolog string, owner, repo string, number int, hits []match) error {
	parts := []string{prolog}
	for _, hit := range hits {
		parts = append(parts, fmt.Sprintf(" * `%s` contains: `%s`", hit.filename, hit.text))
	}

	return comment.Create(owner, repo, number, strings.Join(parts, "\n"))
}

func HandlePullRequest(pre *github.PullRequestEvent) error {
	pr := pre.GetPullRequest()
	log.Printf("PR: %v", pr.String())

	if pr.GetState() == "closed" {
		return nil
	}
	owner, repo := pre.Repo.Owner.GetLogin(), pre.Repo.GetName()

	fixedIssues, err := GetFixedIssues(owner, repo, pr)
	if err != nil {
		return err
	} else if len(fixedIssues) == 0 {
		// This doesn't fix any issues, so nothing to do.
		return comment.CleanupOlder(owner, repo, pr.GetNumber())
	}
	// TODO(mattmoor): Check to see if each of the numbers is
	// actually an issue that's open.

	hits, err := FindIssueTodos(owner, repo, pr.GetHead().GetSHA(), fixedIssues)
	if err != nil {
		return err
	}

	if err := comment.CleanupOlder(owner, repo, pr.GetNumber()); err != nil {
		return err
	}

	if len(hits) == 0 {
		log.Printf("No leftover comments for issues: %v", fixedIssues)
		return nil
	}

	return CommentWithProlog(
		"**The following fixed issues have outstanding TODOs:**",
		owner, repo, pre.GetNumber(),
		hits)
}

func HandleIssues(ie *github.IssuesEvent) error {
	log.Printf("Issue: %v", ie.GetIssue().String())

	owner, repo := ie.Repo.Owner.GetLogin(), ie.Repo.GetName()
	issue := ie.GetIssue()
	// If the issue isn't closed, then just cleanup old comments
	if issue.GetState() != "closed" {
		// Don't clean up here or we'll immediately remove
		// our own comment below.
		return nil
	}

	// If the issue is closed, then:
	//  1. Determine the SHA of the repositories default branch
	//  2. Find any open issue TODOs at that commit.
	//  3. If we find any, then reopen the issue and leave a comment

	ctx := context.Background()
	ghc := client.New(ctx)

	// Determine the SHA of the default branch.
	br, _, err := ghc.Repositories.GetBranch(ctx, owner, repo, ie.Repo.GetDefaultBranch())
	if err != nil {
		return err
	}

	hits, err := FindIssueTodos(owner, repo, br.Commit.GetSHA(), []int{issue.GetNumber()})
	if err != nil {
		return err
	}

	if err := comment.CleanupOlder(owner, repo, issue.GetNumber()); err != nil {
		return err
	}

	if len(hits) == 0 {
		log.Printf("No leftover comments for: %v", issue.GetNumber())
		return nil
	}

	if err := CommentWithProlog(
		"**Reopening due to the following outstanding comments:**",
		owner, repo, issue.GetNumber(), hits); err != nil {
		return err
	}

	// Reopen the issue
	opened := "opened"
	_, _, err = ghc.Issues.Edit(ctx, owner, repo, issue.GetNumber(), &github.IssueRequest{
		State: &opened,
	})
	return err
}
