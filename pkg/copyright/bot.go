package copyright

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"sourcegraph.com/sourcegraph/go-diff/diff"

	"github.com/mattmoor/knobots/pkg/botinfo"
	"github.com/mattmoor/knobots/pkg/comment"
	"github.com/mattmoor/knobots/pkg/handler"
	"github.com/mattmoor/knobots/pkg/reviewrequest"
	"github.com/mattmoor/knobots/pkg/reviewresult"
	"github.com/mattmoor/knobots/pkg/visitor"
)

const reCopyrightRaw = "Copyright \\d{4} [a-zA-Z0-9 .]+"

var reGoCopyrightLine = regexp.MustCompile("^" + reCopyrightRaw + "$")
var reAnyCopyright = regexp.MustCompile(reCopyrightRaw)

type copyright struct{}

var _ handler.Interface = (*copyright)(nil)

func New(context.Context) handler.Interface {
	return &copyright{}
}

func (*copyright) GetType() interface{} {
	return &reviewrequest.Response{}
}

func updateCopyrightYear(orig string) string {
	if !reGoCopyrightLine.MatchString(orig) {
		return orig
	}

	return string(reGoCopyrightLine.ReplaceAllFunc([]byte(orig), func(in []byte) []byte {
		before := string(in)
		return []byte(fmt.Sprintf("Copyright %d%s",
			time.Now().Year(), before[len("Copyright 2018"):]))
	}))
}

type boilerplate struct {
	Path    string
	Content string
}

// TODO(mattmoor): add caching to this, it likely isn't cheap.
func findBoilerplateFile(ext, owner, repo, sha string) (*boilerplate, error) {
	var content *boilerplate

	expectedFileName := "boilerplate" + ext + ".txt"
	err := visitor.Files(owner, repo, sha,
		func(filename string, reader io.Reader) (visitor.VisitControl, error) {
			if strings.HasPrefix(filename, "vendor/") {
				return visitor.Continue, nil
			}
			if filepath.Base(filename) != expectedFileName {
				return visitor.Continue, nil
			}

			// We found the boilerplate file, so extract it and quit the walk.
			b, err := ioutil.ReadAll(reader)
			if err != nil {
				return visitor.Break, err
			}
			content = &boilerplate{
				Path:    filename,
				Content: string(b),
			}
			return visitor.Break, nil
		})
	if err != nil {
		return nil, err
	}
	return content, nil
}

func handleSh(path string, hs []*diff.Hunk, rrr *reviewrequest.Response) (comments []*github.DraftReviewComment) {
	// Each hunk header @@ takes a line.
	// For subsequent hunks, this is covered by the trailing `\n`
	// in each hunk, but the first needs to start at offset 1.
	offset := 1
	// Only the first Hunk should contain the Copyright.
	hunk := hs[0]

	// The hunk starting at line 1 MUST contain the Copyright statement.
	if hunk.NewStartLine != 1 {
		log.Print("Skipping hunk, doesn't include file header.")
		return
	}

	body := string(hunk.Body)
	if !reAnyCopyright.MatchString(body) {
		// If there isn't any Copyright keyword in the first hunk, then attempt to
		// look up the boilerplate file for this file extension.
		bp, err := findBoilerplateFile(".sh", rrr.Owner, rrr.Repository, rrr.Head.GetSHA())
		if err != nil {
			log.Printf("Error finding boilerplate: %v", err)
			return
		} else if bp == nil {
			log.Print("Unable to find boilerplate")
			return
		}

		// Find the first added line to comment on.
		lines := strings.Split(body, "\n")
		for _, line := range lines {
			if !strings.HasPrefix(line, "+") {
				log.Printf("line not added: %q", line)
				offset++
				continue
			}
			orig := line[1:]

			position := offset // Copy it because of &.
			comments = append(comments, &github.DraftReviewComment{
				Path:     &path,
				Position: &position,
				Body: comment.WithCaptionedSuggestion(
					"Missing copyright header:",
					strings.Join([]string{
						bp.Content,
						orig,
					}, "\n"),
				),
			})
			return
		}
	}
	return
}

func handleGo(path string, hs []*diff.Hunk, rrr *reviewrequest.Response) (comments []*github.DraftReviewComment) {
	// Each hunk header @@ takes a line.
	// For subsequent hunks, this is covered by the trailing `\n`
	// in each hunk, but the first needs to start at offset 1.
	offset := 1
	// Only the first Hunk should contain the Copyright.
	hunk := hs[0]

	// The hunk starting at line 1 MUST contain the Copyright statement.
	if hunk.NewStartLine != 1 {
		log.Print("Skipping hunk, doesn't include file header.")
		return
	}

	body := string(hunk.Body)
	if !reAnyCopyright.MatchString(body) {
		// If there isn't any Copyright keyword in the first hunk, then attempt to
		// look up the boilerplate file for this file extension.
		bp, err := findBoilerplateFile(".go", rrr.Owner, rrr.Repository, rrr.Head.GetSHA())
		if err != nil {
			log.Printf("Error finding boilerplate: %v", err)
			return
		} else if bp == nil {
			log.Print("Unable to find boilerplate")
			return
		}

		// Find the first added line to comment on.
		lines := strings.Split(body, "\n")
		for _, line := range lines {
			if !strings.HasPrefix(line, "+") {
				log.Printf("line not added: %q", line)
				offset++
				continue
			}
			orig := line[1:]

			position := offset // Copy it because of &.
			comments = append(comments, &github.DraftReviewComment{
				Path:     &path,
				Position: &position,
				Body: comment.WithCaptionedSuggestion(
					"Missing copyright header:",
					strings.Join([]string{
						bp.Content,
						orig,
					}, "\n"),
				),
			})
			return
		}
	}

	lines := strings.Split(body, "\n")
	for _, line := range lines {
		if !strings.HasPrefix(line, "+") {
			log.Printf("line not added: %q", line)
			offset++
			continue
		}
		orig := line[1:]
		if !reGoCopyrightLine.MatchString(orig) {
			log.Printf("line not copyright: %q", orig)
			offset++
			continue
		}
		updated := updateCopyrightYear(orig)
		if updated != orig {
			position := offset // Copy it because of &.
			comments = append(comments, &github.DraftReviewComment{
				Path:     &path,
				Position: &position,
				Body: comment.WithCaptionedSuggestion(
					"Incorrect copyright year:",
					updated,
				),
			})
		}
		// Increase our offset for each line we see.
		offset++
	}
	return
}

func (*copyright) Handle(ctx context.Context, x interface{}) (handler.Response, error) {
	rrr := x.(*reviewrequest.Response)

	var comments []*github.DraftReviewComment
	err := visitor.Hunks(ctx, rrr.Owner, rrr.Repository, rrr.PullRequest,
		func(path string, hs []*diff.Hunk) (visitor.VisitControl, error) {
			// TODO(mattmoor): Base this on .gitattributes (we should build a library).
			if strings.HasPrefix(path, "vendor/") {
				return visitor.Continue, nil
			}

			// TODO(mattmoor): other file types (yaml?).
			switch filepath.Ext(path) {
			case ".go":
				comments = append(comments, handleGo(path, hs, rrr)...)
			case ".sh":
				comments = append(comments, handleSh(path, hs, rrr)...)
			}

			return visitor.Continue, nil
		})
	if err != nil {
		return nil, err
	}

	return &reviewresult.Payload{
		Name:        botinfo.GetName(),
		Description: `Check for incorrect year in Copyright headers.`,
		Owner:       rrr.Owner,
		Repository:  rrr.Repository,
		PullRequest: rrr.PullRequest,
		SHA:         rrr.Head.GetSHA(),
		Comments:    comments,
	}, nil
}
