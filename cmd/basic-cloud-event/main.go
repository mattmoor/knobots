package main

import (
	"context"
	"log"
	"net/http"

	"github.com/cloudevents/sdk-go"
	"github.com/google/go-github/github"
	"github.com/kelseyhightower/envconfig"
	"knative.dev/pkg/signals"
)

type envConfig struct {
	// Port on which to listen for cloudevents
	Port int `envconfig:"PORT" default:"8080"`
}

type TestingStuff struct {
	PullRequest int
}

func gotEvent(ctx context.Context, event cloudevents.Event, resp *cloudevents.EventResponse) error {
	pre := github.PullRequestEvent{}
	if err := event.DataAs(&pre); err != nil {
		return err
	}

	pr := pre.GetPullRequest()
	log.Printf("PR: %v", pr.String())

	// Ignore closed PRs
	if pr.GetState() == "closed" {
		return nil
	}

	r := cloudevents.Event{
		Context: cloudevents.EventContextV02{
			Source: *cloudevents.ParseURLRef("/does/not/matter"),
			Type:   "dev.knative.source.github.pull_request.opened",
		}.AsV02(),
		Data: TestingStuff{
			PullRequest: pr.GetNumber(),
		},
	}
	resp.RespondWith(http.StatusOK, &r)
	log.Printf("Response Sent!")

	return nil
}

func main() {
	ctx := signals.NewContext()

	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		log.Fatalf("failed to process env var: %s", err)
	}

	t, err := cloudevents.NewHTTPTransport(
		cloudevents.WithPort(env.Port),
	)
	if err != nil {
		log.Fatalf("failed to create transport: %s", err.Error())
	}

	c, err := cloudevents.NewClient(t,
		cloudevents.WithUUIDs(),
		cloudevents.WithTimeNow(),
	)
	if err != nil {
		log.Fatalf("failed to create client: %s", err.Error())
	}

	if err := c.StartReceiver(ctx, gotEvent); err != nil {
		log.Fatalf("failed to start receiver: %s", err.Error())
	}
}
