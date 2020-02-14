package dailybuild

import (
	"context"

	tektonv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tektonclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/injection/client"
	"k8s.io/client-go/kubernetes"
	kubeclient "knative.dev/pkg/client/injection/kube/client"

	"github.com/mattmoor/knobots/pkg/handler"
	"github.com/mattmoor/knobots/pkg/watchbuild"
)

type db struct {
	KubeClient kubernetes.Interface
	Client     tektonclientset.Interface
}

var _ handler.Interface = (*db)(nil)

func New(ctx context.Context) handler.Interface {
	return &db{
		KubeClient: kubeclient.Get(ctx),
		Client:     tektonclient.Get(ctx),
	}
}

func (*db) GetType() interface{} {
	return &Request{}
}

type Request struct {
	TaskRun *tektonv1alpha1.TaskRun `json:"taskRun"`
}

var _ handler.Response = (*Request)(nil)

func (*Request) GetSource() string {
	return "https://github.com/mattmoor/knobots/cmd/periodic"
}

func (*Request) GetType() string {
	return "dev.mattmoor.knobots.dailytaskrun"
}

func (gt *db) Handle(ctx context.Context, x interface{}) (handler.Response, error) {
	rrr := x.(*Request)

	taskrun, err := gt.Client.TektonV1alpha1().TaskRuns(rrr.TaskRun.Namespace).Create(rrr.TaskRun)
	if err != nil {
		return nil, err
	}

	return &watchbuild.Request{
		TaskRun: taskrun,
	}, nil
}
