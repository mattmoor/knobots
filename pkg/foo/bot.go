package foo

import (
	"context"

	"github.com/cloudevents/sdk-go"
	"github.com/mattmoor/knobots/pkg/botinfo"
	"log"

	"github.com/mattmoor/knobots/pkg/bar"
	"github.com/mattmoor/knobots/pkg/handler"
)

type foo struct{}

var _ handler.Interface = (*foo)(nil)

func New() handler.Interface {
	return &foo{}
}

func (*foo) GetType() interface{} {
	return &Payload{}
}

func (*foo) Handle(ctx context.Context, x interface{}) (handler.Response, error) {
	p := x.(*Payload)

	h := cloudevents.HTTPTransportContextFrom(ctx).Header
	log.Printf("[%s] GOT HEADERS: %v", botinfo.GetName(), h)

	return &bar.Payload{
		Message: p.Message,
	}, nil
}

type Payload struct {
	Message string `json:"message"`
}

var _ handler.Response = (*Payload)(nil)

func (*Payload) GetSource() string {
	return "https://github.com/mattmoor/knobots/cmd/foo"
}

func (*Payload) GetType() string {
	return "dev.mattmoor.knobots.foo"
}
