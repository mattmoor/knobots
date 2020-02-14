package upsertksvc

import (
	"context"
	"log"

	"k8s.io/apimachinery/pkg/api/equality"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/serving/pkg/apis/serving/v1alpha1"
	clientset "knative.dev/serving/pkg/client/clientset/versioned"
	servingclient "knative.dev/serving/pkg/client/injection/client"

	"github.com/mattmoor/knobots/pkg/commitstatus"
	"github.com/mattmoor/knobots/pkg/handler"
)

type upsertksvc struct {
	Client clientset.Interface
}

var _ handler.Interface = (*upsertksvc)(nil)

func New(ctx context.Context) handler.Interface {
	return &upsertksvc{
		Client: servingclient.Get(ctx),
	}
}

func (*upsertksvc) GetType() interface{} {
	return &Payload{}
}

func (uk *upsertksvc) Handle(ctx context.Context, x interface{}) (handler.Response, error) {
	p := x.(*Payload)

	log.Printf("Handling iteration %d of %s", p.Count, p.Service.Name)

	ksvc, err := uk.Client.ServingV1alpha1().Services(p.Service.Namespace).Get(
		p.Service.Name, metav1.GetOptions{})
	if apierrs.IsNotFound(err) {
		ksvc, err = uk.Client.ServingV1alpha1().Services(p.Service.Namespace).Create(p.Service)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		// Avoid spin updating due to defaulting.
		p.Service.SetDefaults(context.Background())

		if !equality.Semantic.DeepEqual(ksvc.Spec, p.Service.Spec) {
			ksvc.Spec = p.Service.Spec
			ksvc, err = uk.Client.ServingV1alpha1().Services(ksvc.Namespace).Update(ksvc)
			if err != nil {
				return nil, err
			}
		}
	}

	if ksvc.Generation == ksvc.Status.ObservedGeneration {
		c := ksvc.Status.GetCondition("Ready")
		switch {
		case c == nil:
		case c.Status == "False":
			response := &p.Result
			url := ksvc.Status.URL.String()
			response.URL = &url
			response.State = "failure"
			return response, nil
		case c.Status == "True":
			response := &p.Result
			url := ksvc.Status.URL.String()
			response.URL = &url
			response.State = "success"
			return response, nil
		}
	}

	// Round and round we go.
	return &Payload{
		Service: p.Service,
		Count:   p.Count + 1,
		Result:  p.Result,
	}, nil
}

type Payload struct {
	Service *v1alpha1.Service
	Count   int
	Result  commitstatus.Payload
}

var _ handler.Response = (*Payload)(nil)

func (*Payload) GetSource() string {
	return "https://github.com/mattmoor/knobots/cmd/upsertksvc"
}

func (*Payload) GetType() string {
	return "dev.mattmoor.knobots.upsertksvc"
}
