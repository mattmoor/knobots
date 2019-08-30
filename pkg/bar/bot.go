package bar

import (
	"context"

	"github.com/cloudevents/sdk-go"
	"github.com/mattmoor/knobots/pkg/botinfo"
	"log"

	"github.com/mattmoor/knobots/pkg/baz"
	"github.com/mattmoor/knobots/pkg/handler"
)

type bar struct{}

var _ handler.Interface = (*bar)(nil)

func New() handler.Interface {
	return &bar{}
}

func (*bar) GetType() interface{} {
	return &Payload{}
}

func (*bar) Handle(ctx context.Context, x interface{}) (handler.Response, error) {
	p := x.(*Payload)

	h := cloudevents.HTTPTransportContextFrom(ctx).Header
	log.Printf("[%s] GOT HEADERS: %v", botinfo.GetName(), h)

	return &baz.Payload{
		Message: p.Message,
	}, nil
}

type Payload struct {
	Message string `json:"message"`
}

var _ handler.Response = (*Payload)(nil)

func (*Payload) GetSource() string {
	return "https://github.com/mattmoor/knobots/cmd/bar"
}

func (*Payload) GetType() string {
	return "dev.mattmoor.knobots.bar"
}
