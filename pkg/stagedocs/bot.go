package stagedocs

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/ptr"
	"knative.dev/serving/pkg/apis/serving/v1alpha1"
	"knative.dev/serving/pkg/apis/serving/v1beta1"

	tektonv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tektonclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"

	"github.com/mattmoor/knobots/pkg/botinfo"
	"github.com/mattmoor/knobots/pkg/commitstatus"
	"github.com/mattmoor/knobots/pkg/handler"
	"github.com/mattmoor/knobots/pkg/reviewrequest"
	"github.com/mattmoor/knobots/pkg/upsertksvc"
	"github.com/mattmoor/knobots/pkg/watchbuild"
)

type stagedocs struct {
	Client tektonclientset.Interface
}

var _ handler.Interface = (*stagedocs)(nil)

func New(bc tektonclientset.Interface) handler.Interface {
	return &stagedocs{Client: bc}
}

func (*stagedocs) GetType() interface{} {
	return &reviewrequest.Response{}
}

func (gt *stagedocs) Handle(ctx context.Context, x interface{}) (handler.Response, error) {
	rrr := x.(*reviewrequest.Response)

	// Only run this on knative/docs
	if rrr.Owner != "knative" && rrr.Repository != "docs" {
		return nil, nil
	}

	if rrr.Head.GetUser().GetLogin() != "mattmoor" {
		return nil, nil
	}

	// We need to use a different image tag per build, or the service
	// updates won't change anything.
	image := fmt.Sprintf(
		"gcr.io/mattmoor-knative/docs-on-the-rocks:%d-%s",
		rrr.PullRequest, *rrr.Head.SHA)

	taskrun := &tektonv1alpha1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "docs-on-the-rocks-",
			// TODO(mattmoor): Namespace: system.Namespace()?  Needs downward API :(
			Namespace: "default",
		},
		Spec: tektonv1alpha1.TaskRunSpec{
			TaskRef: &tektonv1alpha1.TaskRef{
				Name: "docs-image",
			},
			Inputs: tektonv1alpha1.TaskRunInputs{
				Resources: []tektonv1alpha1.TaskResourceBinding{{
					PipelineResourceBinding: tektonv1alpha1.PipelineResourceBinding{
						Name: "docs-source",
						ResourceSpec: &tektonv1alpha1.PipelineResourceSpec{
							Type: tektonv1alpha1.PipelineResourceTypeGit,
							Params: []tektonv1alpha1.ResourceParam{{
								Name:  "url",
								Value: rrr.Head.GetRepo().GetCloneURL(),
							}, {
								Name:  "revision",
								Value: rrr.Head.GetRef(),
							}},
						},
					},
				}, {
					PipelineResourceBinding: tektonv1alpha1.PipelineResourceBinding{
						Name: "website-source",
						ResourceSpec: &tektonv1alpha1.PipelineResourceSpec{
							Type: tektonv1alpha1.PipelineResourceTypeGit,
							Params: []tektonv1alpha1.ResourceParam{{
								Name:  "url",
								Value: "https://github.com/knative/website.git",
							}},
						},
					},
				}},
				Params: []tektonv1alpha1.Param{{
					Name: "IMAGE",
					Value: tektonv1alpha1.ArrayOrString{
						Type:      tektonv1alpha1.ParamTypeString,
						StringVal: image,
					},
				}},
			},
		},
	}
	taskrun, err := gt.Client.TektonV1alpha1().TaskRuns(taskrun.Namespace).Create(taskrun)
	if err != nil {
		return nil, err
	}

	response := &upsertksvc.Payload{
		Service: &v1alpha1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("docs-on-the-rocks-%d", rrr.PullRequest),
				Namespace: "default",
			},
			Spec: v1alpha1.ServiceSpec{
				ConfigurationSpec: v1alpha1.ConfigurationSpec{
					Template: &v1alpha1.RevisionTemplateSpec{
						Spec: v1alpha1.RevisionSpec{
							RevisionSpec: v1beta1.RevisionSpec{
								PodSpec: corev1.PodSpec{
									Containers: []corev1.Container{{
										Image: image,
										Ports: []corev1.ContainerPort{{
											ContainerPort: 1313,
										}},
									}},
								},
								// This is so things scale to zero faster.
								TimeoutSeconds: ptr.Int64(30),
							},
						},
					},
				},
			},
		},
		Result: commitstatus.Payload{
			Name:        botinfo.GetName(),
			Description: `Stage the website.`,
			Owner:       rrr.Owner,
			Repository:  rrr.Repository,
			SHA:         rrr.Head.GetSHA(),
		},
	}

	c, err := handler.ToContinuation(response)
	if err != nil {
		return nil, err
	}

	return &watchbuild.Request{
		TaskRun:      taskrun,
		Continuation: c,
	}, nil
}
