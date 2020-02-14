package baz

import (
	"context"

	"log"

	cloudevents "github.com/cloudevents/sdk-go"
	"github.com/mattmoor/knobots/pkg/botinfo"

	"github.com/mattmoor/knobots/pkg/handler"
	"github.com/mattmoor/knobots/pkg/slack"
)

type baz struct{}

var _ handler.Interface = (*baz)(nil)

func New(context.Context) handler.Interface {
	return &baz{}
}

func (*baz) GetType() interface{} {
	return &Payload{}
}

func (*baz) Handle(ctx context.Context, x interface{}) (handler.Response, error) {
	p := x.(*Payload)

	h := cloudevents.HTTPTransportContextFrom(ctx).Header
	log.Printf("[%s] GOT HEADERS: %v", botinfo.GetName(), h)

	return slack.ErrorReport(p.Message, map[string]string{}), nil
}

type Payload struct {
	Message string `json:"message"`
}

var _ handler.Response = (*Payload)(nil)

func (*Payload) GetSource() string {
	return "https://github.com/mattmoor/knobots/cmd/baz"
}

func (*Payload) GetType() string {
	return "dev.mattmoor.knobots.baz"
}
