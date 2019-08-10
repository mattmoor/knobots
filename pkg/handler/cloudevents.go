package handler

import (
	"context"
	"log"
	"net/http"

	"github.com/cloudevents/sdk-go"
	"github.com/kelseyhightower/envconfig"
	"knative.dev/pkg/signals"
)

type envConfig struct {
	// Port on which to listen for cloudevents
	Port int `envconfig:"PORT" default:"8080"`
}

type Response interface {
	GetType() string
	GetSource() string
}

type Interface interface {
	GetType() interface{}
	Handle(interface{}) (Response, error)
}

func gotEvent(h Interface) interface{} {
	return func(ctx context.Context, event cloudevents.Event, resp *cloudevents.EventResponse) error {
		data := h.GetType()
		if err := event.DataAs(&data); err != nil {
			return err
		}

		response, err := h.Handle(data)
		if err != nil {
			return err
		}

		if response == nil {
			return nil
		}

		r := cloudevents.NewEvent(cloudevents.VersionV03)
		r.SetType(response.GetType())
		r.SetSource(response.GetSource())
		r.SetData(response)

		resp.RespondWith(http.StatusOK, &r)
		log.Printf("Response Sent!")

		return nil
	}
}

func Main(h Interface) {
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

	if err := c.StartReceiver(ctx, gotEvent(h)); err != nil {
		log.Fatalf("failed to start receiver: %s", err.Error())
	}
}
