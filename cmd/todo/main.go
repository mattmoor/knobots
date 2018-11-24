package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

func main() {
	http.HandleFunc("/", Handler)
	http.ListenAndServe(":8080", nil)
}

// This should be unique per bot.
const MagicString = `<!--TODO BOT-->`

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

func HasMagicString(comment string) bool {
	return strings.Contains(comment, MagicString)
}

func WithMagicString(comment string) *string {
	both := MagicString + "\n" + comment
	return &both
}

func CleanupOldComments(owner, repo string, number int) error {
	ctx := context.Background()
	ghc := GetClient(ctx)

	var ids []int64

	lopt := &github.IssueListCommentsOptions{}
	for {
		comments, resp, err := ghc.Issues.ListComments(ctx, owner, repo, number, lopt)
		if err != nil {
			return err
		}
		for _, comment := range comments {
			if HasMagicString(comment.GetBody()) {
				ids = append(ids, comment.GetID())
			}
		}
		if lopt.Page == resp.NextPage {
			break
		}
		lopt.Page = resp.NextPage
	}

	for _, id := range ids {
		_, err := ghc.Issues.DeleteComment(ctx, owner, repo, id)
		if err != nil {
			return err
		}
	}

	return nil
}

type match struct {
	filename string
	text     string
}

func FindIssueTodos(owner, repo, sha string, issues []int) ([]match, error) {
	reIssue := IssueRegexp(issues)

	var hits []match
	err := FileWalker(owner, repo, sha, func(filename string, reader io.Reader) error {
		if strings.HasPrefix(filename, "vendor/") {
			return nil
		}
		body, err := ioutil.ReadAll(reader)
		if err != nil {
			return err
		}
		ms := reIssue.FindAll(body, 5)
		if ms == nil {
			return nil
		}
		for _, m := range ms {
			hits = append(hits, match{
				filename: filename,
				// TODO(mattmoor): Find a way to include position to comment on it?
				text: string(m),
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return hits, nil
}

func CommentWithProlog(prolog string, owner, repo string, number int, hits []match) error {
	ctx := context.Background()
	ghc := GetClient(ctx)

	parts := []string{prolog}
	for _, hit := range hits {
		parts = append(parts, fmt.Sprintf(" * `%s` contains: `%s`", hit.filename, hit.text))
	}

	msg := strings.Join(parts, "\n")
	_, _, err := ghc.Issues.CreateComment(ctx,
		owner, repo, number,
		&github.IssueComment{
			Body: WithMagicString(msg),
		})
	return err
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
		return CleanupOldComments(owner, repo, pr.GetNumber())
	}
	// TODO(mattmoor): Check to see if each of the numbers is
	// actually an issue that's open.

	hits, err := FindIssueTodos(owner, repo, pr.GetHead().GetSHA(), fixedIssues)
	if err != nil {
		return err
	}

	if err := CleanupOldComments(owner, repo, pr.GetNumber()); err != nil {
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
	ghc := GetClient(ctx)

	// Determine the SHA of the default branch.
	br, _, err := ghc.Repositories.GetBranch(ctx, owner, repo, ie.Repo.GetDefaultBranch())
	if err != nil {
		return err
	}

	hits, err := FindIssueTodos(owner, repo, br.Commit.GetSHA(), []int{issue.GetNumber()})
	if err != nil {
		return err
	}

	if err := CleanupOldComments(owner, repo, issue.GetNumber()); err != nil {
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

type FileVisitor func(filename string, reader io.Reader) error

func FileWalker(owner, repo, sha string, v FileVisitor) error {
	// TODO(mattmoor): Maybe this should use this:
	// https://godoc.org/github.com/google/go-github/github#RepositoriesService.GetArchiveLink
	url := fmt.Sprintf("https://codeload.github.com/%s/%s/tar.gz/%s", owner, repo, sha)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}

	tr := tar.NewReader(gr)

	// All of the files in the archive should have the following prefix
	prefix := fmt.Sprintf("%s-%s/", repo, sha)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		if !strings.HasPrefix(header.Name, prefix) {
			log.Printf("File without prefix: %s", header.Name)
			continue
		}
		if !header.FileInfo().Mode().IsRegular() {
			log.Printf("Ignoring file (not regular): %s", header.Name)
			continue
		}
		stripped := header.Name[len(prefix):]
		if err := v(stripped, tr); err != nil {
			return err
		}
	}

	return nil
}

func GetClient(ctx context.Context) *github.Client {
	return github.NewClient(
		oauth2.NewClient(ctx,
			oauth2.StaticTokenSource(
				&oauth2.Token{
					AccessToken: os.Getenv("GITHUB_ACCESS_TOKEN"),
				},
			),
		),
	)
}
