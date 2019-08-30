package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/plugin/ochttp/propagation/b3"
	"go.opencensus.io/trace"
)

func main() {
	// TODO(mattmoor): Send this:
	// curl -v -X POST \
	//   -H "Content-Type: application/json" \
	//   -H "ce-specversion: 0.2" \
	//   -H "ce-source: github.com/mattmoor/knobots/cmd/send-event" \
	//   -H "ce-type: dev.mattmoor.knobots.foo" \
	//   -H "ce-id: 123-abc" \
	//   -H "x-b3-sampled: 1" \
	//   -d '{"message": "Testing..."}' \
	//   http://default-broker.default.svc.cluster.local

	// Having already created your sampler "theSampler"
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})

	// Use an http.RoundTripper that instruments all outgoing requests with stats and tracing.
	client := &http.Client{Transport: &ochttp.Transport{Propagation: &b3.HTTPFormat{}}}

	req := &http.Request{
		Method: http.MethodPost,
		URL: &url.URL{
			Scheme: "http",
			Host:   "default-broker.default.svc.cluster.local",
		},
		Header: http.Header{
			"Content-Type":   []string{"application/json"},
			"ce-specversion": []string{"0.2"},
			"ce-source":      []string{"github.com/mattmoor/knobots/cmd/send-event"},
			"ce-type":        []string{"dev.mattmoor.knobots.foo"},
			"ce-id":          []string{"123-abc"},
		},
		Body: ioutil.NopCloser(bytes.NewBufferString(`{"message": "Testing..."}`)),
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("error posting: %v", err)
	}
	log.Printf("got response code: %d", resp.StatusCode)
}
