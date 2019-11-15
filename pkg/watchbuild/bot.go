package watchbuild

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/ghodss/yaml"
	buildv1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	buildclientset "github.com/knative/build/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/ptr"

	"github.com/mattmoor/knobots/pkg/handler"
	"github.com/mattmoor/knobots/pkg/slack"
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
	Count int                  `json:"count"`
}

var _ handler.Response = (*Request)(nil)

func (*Request) GetSource() string {
	return "https://github.com/mattmoor/knobots/cmd/periodic"
}

func (*Request) GetType() string {
	return "dev.mattmoor.knobots.watchbuild"
}

func (gt *db) Handle(ctx context.Context, x interface{}) (handler.Response, error) {
	rrr := x.(*Request)

	log.Printf("Watching build: %s (count: %d)", rrr.Build.Name, rrr.Count)

	wi, err := gt.BuildClient.BuildV1alpha1().Builds(rrr.Build.Namespace).Watch(metav1.ListOptions{
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
				Build: rrr.Build,
				Count: rrr.Count + 1,
			}, nil

		case event, ok := <-wi.ResultChan():
			if !ok {
				log.Printf("Unexpected end of watch for build: %s (trying again)", rrr.Build.Name)
				return &Request{
					Build: rrr.Build,
					Count: rrr.Count + 1,
				}, nil
			}
			if event.Type != watch.Modified {
				break
			}
			b := event.Object.(*buildv1alpha1.Build)
			if b.Name != rrr.Build.Name {
				// Not our build
				break
			}

			c := b.Status.GetCondition("Succeeded")
			switch {
			case c == nil, c.Status == "Unknown":
				// Not done.
				break

			case c.Status == "True":
				log.Printf("Build %s succeeded", b.Name)
				return nil, gt.BuildClient.BuildV1alpha1().Builds(rrr.Build.Namespace).Delete(
					rrr.Build.Name,
					&metav1.DeleteOptions{},
				)

			case c.Status == "False":
				// Build failed
				log.Printf("Build %s failed", b.Name)

				byaml, err := yaml.Marshal(rrr.Build)
				if err != nil {
					log.Printf("error serializing build: %v", err)
					return nil, err
				}

				attributes := map[string]string{
					"pod":   fmt.Sprintf("%s/%s", b.Status.Cluster.Namespace, b.Status.Cluster.PodName),
					"build": fmt.Sprintf("\n```\n%s\n```\n", string(byaml)),
				}

				ns, name := b.Status.Cluster.Namespace, b.Status.Cluster.PodName
				pod, err := gt.KubeClient.CoreV1().Pods(ns).Get(name, metav1.GetOptions{})
				if err != nil {
					log.Printf("error fetching pod: %v", err)
					return nil, err
				}

				for _, c := range pod.Spec.InitContainers {
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
						"daily build failed",
						attributes,
					), gt.BuildClient.BuildV1alpha1().Builds(rrr.Build.Namespace).Delete(
						rrr.Build.Name,
						&metav1.DeleteOptions{},
					)
			}
		}
	}
}
