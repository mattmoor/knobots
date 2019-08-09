package handler

import (
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/google/go-github/github"
)

func ParseGithubData(payload []byte, eventType string) (interface{}, error) {
	// TODO(mattmoor): This should be:
	//     eventType := github.WebHookType(r)
	// https://github.com/knative/eventing-sources/issues/120
	// HACK HACK HACK
	eventType = strings.Split(eventType, ".")[4]

	return github.ParseWebHook(eventType, payload)
}

func ParseGithubWebhook(w http.ResponseWriter, r *http.Request) interface{} {
	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("ERROR: no payload: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil
	}

	x, err := ParseGithubData(payload, r.Header.Get("ce-type"))
	if err != nil {
		log.Printf("ERROR: unable to parse webhook: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil
	}
	return x
}
