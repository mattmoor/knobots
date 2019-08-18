package builds

import (
	"log"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	buildv1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	buildclientset "github.com/knative/build/pkg/client/clientset/versioned"
	"knative.dev/pkg/ptr"
)

func Run(bc buildclientset.Interface, build *buildv1alpha1.Build, f func(*buildv1alpha1.Build) error) error {
	build, err := bc.BuildV1alpha1().Builds(build.Namespace).Create(build)
	if err != nil {
		return err
	}
	// Cleanup behind ourselves.
	// TODO(mattmoor): Consider leaving around failed builds?
	defer func() {
		bc.BuildV1alpha1().Builds(build.Namespace).Delete(build.Name, &metav1.DeleteOptions{})
	}()

	wi, err := bc.BuildV1alpha1().Builds(build.Namespace).Watch(metav1.ListOptions{
		TimeoutSeconds: ptr.Int64(600),
	})
	if err != nil {
		return err
	}
	defer wi.Stop()

	timeout := time.After(8 * time.Minute)
	for {
		select {
		case <-timeout:
			log.Printf("Timed out waiting for build %s", build.Name)
			return nil
		case event, ok := <-wi.ResultChan():
			if !ok {
				log.Printf("Unexpected end of watch for build: %s", build.Name)
				return nil
			}
			if event.Type != watch.Modified {
				break
			}
			b := event.Object.(*buildv1alpha1.Build)
			if b.Name != build.Name {
				// Not our build
				break
			}
			c := b.Status.GetCondition("Succeeded")
			switch {
			case c == nil, c.Status == "Unknown":
				// Not done.
				break

			case c.Status == "True":
				return f(b)
			case c.Status == "False":
				return f(b)
			}
		}
	}
}
