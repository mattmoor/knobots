package github

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/google/go-github/github"
)

// formatRequest generates ascii representation of a request
// from: https://medium.com/doing-things-right/pretty-printing-http-requests-in-golang-a918d5aaa000
func formatRequest(r *http.Request) string {
	// Create return string
	var request []string
	// Add the request string
	url := fmt.Sprintf("%v %v %v", r.Method, r.URL, r.Proto)
	request = append(request, url)
	// Add the host
	request = append(request, fmt.Sprintf("Host: %v", r.Host))
	// Loop through headers
	for name, headers := range r.Header {
		name = strings.ToLower(name)
		for _, h := range headers {
			request = append(request, fmt.Sprintf("%v: %v", name, h))
		}
	}

	// If this is a POST, add post data
	if r.Method == "POST" {
		r.ParseForm()
		request = append(request, "\n")
		request = append(request, r.Form.Encode())
	}
	// Return the request as a string
	return strings.Join(request, "\n")
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
		log.Printf("ERROR: unable to parse webhook: %v\n%v", err, formatRequest(r))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	handleErr := func(event interface{}, err error) {
		if err == nil {
			fmt.Fprintf(w, "Handled %T", event)
			return
		}
		log.Printf("Error handling %T: %v", event, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// The set of events here should line up with what is in
	//   config/one-time/github-source.yaml
	switch event := event.(type) {
	case *github.PullRequestEvent:
		handleErr(event, HandleOther(event))
	case *github.PullRequestReviewEvent:
		handleErr(event, HandleOther(event))
	case *github.PullRequestReviewCommentEvent:
		handleErr(event, HandleOther(event))
	case *github.IssueCommentEvent:
		handleErr(event, HandleIssue(event))
	default:
		log.Printf("Unrecognized event: %T", event)
		http.Error(w, "Unknown event", http.StatusBadRequest)
		return
	}
}

func HandleIssue(ice *github.IssueCommentEvent) error {
	log.Printf("Comment from %s on #%d: %q",
		ice.Sender.GetLogin(),
		ice.Issue.GetNumber(),
		ice.Comment.GetBody())

	// TODO(mattmoor): Is ice.Repo.Owner.Login reliable for organizations, or do we
	// have to parse the FullName?
	//    Owner: mattmoor, Repo: kontext, Fullname: mattmoor/kontext
	// log.Printf("Owner: %s, Repo: %s, Fullname: %s", *ice.Repo.Owner.Login, *ice.Repo.Name,
	// 	*ice.Repo.FullName)

	if strings.Contains(*ice.Comment.Body, "Hello there.") {
		ctx := context.Background()
		ghc := GetClient(ctx)

		msg := fmt.Sprintf("Hello @%s", ice.Sender.GetLogin())

		_, _, err := ghc.Issues.CreateComment(ctx,
			ice.Repo.Owner.GetLogin(), ice.Repo.GetName(), ice.Issue.GetNumber(),
			&github.IssueComment{
				Body: &msg,
			})
		return err
	}

	return nil
}

func HandleOther(event interface{}) error {
	log.Printf("TODO %T: %#v", event, event)
	return nil
}
