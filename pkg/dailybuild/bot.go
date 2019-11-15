package dailybuild

import (
	"context"

	buildv1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	buildclientset "github.com/knative/build/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"

	"github.com/mattmoor/knobots/pkg/handler"
	"github.com/mattmoor/knobots/pkg/watchbuild"
)

type db struct {
	KubeClient  kubernetes.Interface
	BuildClient buildclientset.Interface
}

var _ handler.Interface = (*db)(nil)

func New(ks kubernetes.Interface, bc buildclientset.Interface) handler.Interface {
	return &db{
		KubeClient:  ks,
		BuildClient: bc,
	}
}

func (*db) GetType() interface{} {
	return &Request{}
}

type Request struct {
	Build *buildv1alpha1.Build `json:"build"`
}

var _ handler.Response = (*Request)(nil)

func (*Request) GetSource() string {
	return "https://github.com/mattmoor/knobots/cmd/periodic"
}

func (*Request) GetType() string {
	return "dev.mattmoor.knobots.dailybuild"
}

func (gt *db) Handle(ctx context.Context, x interface{}) (handler.Response, error) {
	rrr := x.(*Request)

	build, err := gt.BuildClient.BuildV1alpha1().Builds(rrr.Build.Namespace).Create(rrr.Build)
	if err != nil {
		return nil, err
	}

	return &watchbuild.Request{
		Build: build,
	}, nil
}
