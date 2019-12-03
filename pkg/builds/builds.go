package builds

import (
	"fmt"
	"log"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	tektonv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tektonclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"knative.dev/pkg/ptr"
)

func Run(bc tektonclientset.Interface, taskrun *tektonv1alpha1.TaskRun, f func(*tektonv1alpha1.TaskRun) error) error {
	taskrun, err := bc.TektonV1alpha1().TaskRuns(taskrun.Namespace).Create(taskrun)
	if err != nil {
		return err
	}
	// Cleanup behind ourselves.
	// TODO(mattmoor): Consider leaving around failed taskruns?
	defer func() {
		bc.TektonV1alpha1().TaskRuns(taskrun.Namespace).Delete(taskrun.Name, &metav1.DeleteOptions{})
	}()

	wi, err := bc.TektonV1alpha1().TaskRuns(taskrun.Namespace).Watch(metav1.ListOptions{
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
			return fmt.Errorf("Timed out waiting for taskrun %s", taskrun.Name)

		case event, ok := <-wi.ResultChan():
			if !ok {
				log.Printf("Unexpected end of watch for taskrun: %s", taskrun.Name)
				return nil
			}
			if event.Type != watch.Modified {
				break
			}
			b := event.Object.(*tektonv1alpha1.TaskRun)
			if b.Name != taskrun.Name {
				// Not our taskrun
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
