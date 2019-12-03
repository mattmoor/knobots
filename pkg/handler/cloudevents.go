package handler

import (
	"context"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/compute/metadata"
	"contrib.go.opencensus.io/exporter/stackdriver"
	cloudevents "github.com/cloudevents/sdk-go"
	"github.com/cloudevents/sdk-go/pkg/cloudevents/client"
	transporthttp "github.com/cloudevents/sdk-go/pkg/cloudevents/transport/http"
	"github.com/kelseyhightower/envconfig"
	"go.opencensus.io/plugin/ochttp/propagation/b3"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"go.opencensus.io/trace/propagation"
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
	Handle(context.Context, interface{}) (Response, error)
}

func responseContext(ctx context.Context, pf propagation.HTTPFormat) *cloudevents.HTTPTransportResponseContext {
	sc, ok := pf.SpanContextFromRequest(&http.Request{
		Header: cloudevents.HTTPTransportContextFrom(ctx).Header,
	})
	if !ok {
		return nil
	}
	req := &http.Request{
		Header: http.Header{},
	}
	pf.SpanContextToRequest(sc, req)

	log.Printf("SENDING RESPONSE HEADERS: %v", req.Header)

	return &cloudevents.HTTPTransportResponseContext{
		Header: req.Header,
	}
}

func gotEvent(h Interface) interface{} {
	return func(ctx context.Context, event cloudevents.Event, resp *cloudevents.EventResponse) error {
		data := h.GetType()
		if err := event.DataAs(&data); err != nil {
			return err
		}

		response, err := h.Handle(ctx, data)
		if err != nil {
			log.Printf("handle returned error: %v", err)
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
		resp.Context = responseContext(ctx, &b3.HTTPFormat{})
		log.Printf("Response Sent!")

		return nil
	}
}

func Send(resp Response) error {
	// Use an http.RoundTripper that instruments all outgoing requests with stats and tracing.
	// client := &http.Client{Transport: &ochttp.Transport{Propagation: &b3.HTTPFormat{}}}
	client, err := cloudevents.NewDefaultClient()
	if err != nil {
		return err
	}

	r := cloudevents.NewEvent(cloudevents.VersionV1)
	r.SetType(resp.GetType())
	r.SetSource(resp.GetSource())
	r.SetDataContentType("application/json")
	r.SetData(resp)

	ctx := cloudevents.ContextWithTarget(context.Background(), os.Getenv("K_SINK"))
	_, _, err = client.Send(ctx, r)
	return err
}

func Main(h Interface) {
	ctx := signals.NewContext()

	projectID, err := metadata.ProjectID()
	if err != nil {
		log.Fatalf("unable to fetch GCP ProjectID: %v", err)
	}

	// Create and register a OpenCensus Stackdriver Trace exporter.
	exporter, err := stackdriver.NewExporter(stackdriver.Options{
		ProjectID: projectID,
	})
	if err != nil {
		log.Fatalf("stackdriver.NewExporter() = %v", err)
	}
	trace.RegisterExporter(exporter)

	if err := view.Register(
		client.LatencyView,
		transporthttp.LatencyView,
	); err != nil {
		log.Fatalf("failed to register views: %v", err)
	}

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
