package dailybuild

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/ghodss/yaml"
	buildv1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	buildclientset "github.com/knative/build/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/ptr"

	"github.com/mattmoor/knobots/pkg/builds"
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

	var resp handler.Response
	err := builds.Run(gt.BuildClient, rrr.Build, func(b *buildv1alpha1.Build) error {
		c := b.Status.GetCondition("Succeeded")
		switch c.Status {
		case "True":
			log.Printf("Build %s succeeded", b.Name)
			return nil

		case "False":
			// Build failed
			log.Printf("Build %s failed", b.Name)

			byaml, err := yaml.Marshal(rrr.Build)
			if err != nil {
				log.Printf("error serializing build: %v", err)
				return err
			}

			attributes := map[string]string{
				"pod":   fmt.Sprintf("%s/%s", b.Status.Cluster.Namespace, b.Status.Cluster.PodName),
				"build": fmt.Sprintf("\n```\n%s\n```\n", string(byaml)),
			}

			ns, name := b.Status.Cluster.Namespace, b.Status.Cluster.PodName
			pod, err := gt.KubeClient.CoreV1().Pods(ns).Get(name, metav1.GetOptions{})
			if err != nil {
				log.Printf("error fetching pod: %v", err)
				return err
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

			resp = slack.ErrorReport("daily build failed", attributes)
			return nil
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}
