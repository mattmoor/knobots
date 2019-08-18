package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
	buildv1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	buildclientset "github.com/knative/build/pkg/client/clientset/versioned"
	"github.com/mattmoor/knobots/pkg/builds"
	"k8s.io/client-go/tools/clientcmd"
	"knative.dev/serving/pkg/pool"
)

var (
	kubeconfig = flag.String("kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	masterURL  = flag.String("master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
)

func buildPath(buildClient buildclientset.Interface, path string) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	build := &buildv1alpha1.Build{}
	if err := yaml.Unmarshal(b, build); err != nil {
		return err
	}
	return builds.Run(buildClient, build, func(b *buildv1alpha1.Build) error {
		c := b.Status.GetCondition("Succeeded")
		switch c.Status {
		case "True":
			log.Printf("build %s succeeded.", b.Name)
		case "False":
			log.Printf("build %s failed. %v: %v.", b.Name, c.Reason, c.Message)
			// TODO(mattmoor): Consider returning an error.
		}
		return nil
	})
}

func main() {
	flag.Parse()

	cfg, err := clientcmd.BuildConfigFromFlags(*masterURL, *kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %v", err)
	}

	buildClient, err := buildclientset.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Error building Build clientset: %v", err)
	}

	p := pool.New(5)

	err = filepath.Walk(os.Getenv("KO_DATA_PATH"),
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
				return buildPath(buildClient, path)
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
