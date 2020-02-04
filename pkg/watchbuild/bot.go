package watchbuild

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/ghodss/yaml"
	tektonv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tektonclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/ptr"

	"github.com/mattmoor/knobots/pkg/handler"
	"github.com/mattmoor/knobots/pkg/slack"
)

type db struct {
	KubeClient kubernetes.Interface
	Client     tektonclientset.Interface
}

var _ handler.Interface = (*db)(nil)

func New(ks kubernetes.Interface, bc tektonclientset.Interface) handler.Interface {
	return &db{
		KubeClient: ks,
		Client:     bc,
	}
}

func (*db) GetType() interface{} {
	return &Request{}
}

type Request struct {
	TaskRun      *tektonv1alpha1.TaskRun `json:"taskRun"`
	Count        int                     `json:"count"`
	Continuation *handler.Continuation   `json:"continuation,omitempty"`
}

var _ handler.Response = (*Request)(nil)

func (*Request) GetSource() string {
	return "https://github.com/mattmoor/knobots/cmd/watchbuild"
}

func (*Request) GetType() string {
	return "dev.mattmoor.knobots.watchtaskrun"
}

func (gt *db) Handle(ctx context.Context, x interface{}) (handler.Response, error) {
	rrr := x.(*Request)

	log.Printf("Watching taskrun: %s (count: %d)", rrr.TaskRun.Name, rrr.Count)

	wi, err := gt.Client.TektonV1alpha1().TaskRuns(rrr.TaskRun.Namespace).Watch(metav1.ListOptions{
		TimeoutSeconds: ptr.Int64(600),
	})
	if err != nil {
		return nil, err
	}
	defer wi.Stop()

	timeout := time.After(1 * time.Minute)
	for {
		select {
		case <-timeout:
			return &Request{
				TaskRun:      rrr.TaskRun,
				Count:        rrr.Count + 1,
				Continuation: rrr.Continuation,
			}, nil

		case event, ok := <-wi.ResultChan():
			if !ok {
				log.Printf("Unexpected end of watch for taskrun: %s (trying again)", rrr.TaskRun.Name)
				return &Request{
					TaskRun:      rrr.TaskRun,
					Count:        rrr.Count + 1,
					Continuation: rrr.Continuation,
				}, nil
			}
			if event.Type != watch.Modified {
				break
			}
			b := event.Object.(*tektonv1alpha1.TaskRun)
			if b.Name != rrr.TaskRun.Name {
				// Not our taskrun
				break
			}

			c := b.Status.GetCondition("Succeeded")
			switch {
			case c == nil, c.Status == "Unknown":
				// Not done.
				break

			case c.Status == "True":
				log.Printf("TaskRun %s succeeded", b.Name)
				err := gt.Client.TektonV1alpha1().TaskRuns(rrr.TaskRun.Namespace).Delete(
					rrr.TaskRun.Name,
					&metav1.DeleteOptions{},
				)
				if err != nil {
					return nil, err
				}
				if rrr.Continuation == nil {
					return nil, nil
				}
				return rrr.Continuation.AsResponse(), nil

			case c.Status == "False":
				// TaskRun failed
				log.Printf("TaskRun %s failed", b.Name)

				byaml, err := yaml.Marshal(rrr.TaskRun)
				if err != nil {
					log.Printf("error serializing taskrun: %v", err)
					return nil, err
				}

				attributes := map[string]string{
					"pod":     fmt.Sprintf("%s/%s", b.Namespace, b.Status.PodName),
					"taskrun": fmt.Sprintf("\n```\n%s\n```\n", string(byaml)),
				}

				ns, name := b.Namespace, b.Status.PodName
				pod, err := gt.KubeClient.CoreV1().Pods(ns).Get(name, metav1.GetOptions{})
				if err != nil {
					log.Printf("error fetching pod: %v", err)
					return nil, err
				}

				for _, c := range append(pod.Spec.InitContainers, pod.Spec.Containers...) {
					options := &corev1.PodLogOptions{
						Container: c.Name,
						// Get everything
						SinceSeconds: ptr.Int64(100000),
					}

					stream, err := gt.KubeClient.CoreV1().Pods(ns).GetLogs(name, options).Stream()
					if err != nil {
						attributes[c.Name] = fmt.Sprintf("`%v`", err)
						continue
					}
					defer stream.Close()
					b, err := ioutil.ReadAll(stream)
					if err != nil {
						attributes[c.Name] = fmt.Sprintf("`%v`", err)
						continue
					}
					attributes[c.Name] = fmt.Sprintf("\n```\n%s\n```\n", string(b))
				}

				return slack.ErrorReport(
						"taskrun failed",
						attributes,
					), gt.Client.TektonV1alpha1().TaskRuns(rrr.TaskRun.Namespace).Delete(
						rrr.TaskRun.Name,
						&metav1.DeleteOptions{},
					)
			}
		}
	}
}
