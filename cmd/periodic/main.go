package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/mattmoor/knobots/pkg/dailybuild"
	"github.com/mattmoor/knobots/pkg/handler"
	tektonv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"knative.dev/pkg/system"
	"knative.dev/serving/pkg/pool"
)

func buildPath(path string) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	log.Printf("Got(%q): %s", path, string(b))
	taskrun := &tektonv1alpha1.TaskRun{}
	if err := yaml.Unmarshal(b, taskrun); err != nil {
		return err
	}
	taskrun.Namespace = system.Namespace()
	log.Printf("Send(%q): %+v", path, taskrun)
	return handler.Send(&dailybuild.Request{
		TaskRun: taskrun,
	})
}

func main() {
	flag.Parse()

	p := pool.New(5)

	err := filepath.Walk(os.Getenv("KO_DATA_PATH"),
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if filepath.Ext(path) != ".yaml" {
				return nil
			}
			// Use a pool to execute a limited number of these, wait for them
			// to complete, clean them up, signal on errors, etc.
			p.Go(func() error {
				return buildPath(path)
			})
			return nil
		})
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}

	if err := p.Wait(); err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}
}
