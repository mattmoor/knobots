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
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfig = flag.String("kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	masterURL  = flag.String("master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
)

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

	err = filepath.Walk(os.Getenv("KO_DATA_PATH"),
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if filepath.Ext(path) != ".yaml" {
				return nil
			}
			b, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			build := &buildv1alpha1.Build{}
			if err := yaml.Unmarshal(b, build); err != nil {
				return err
			}
			ns := "default"
			if build.Namespace != "" {
				ns = build.Namespace
			}
			newb, err := buildClient.BuildV1alpha1().Builds(ns).Create(build)
			if err != nil {
				return err
			}
			log.Printf("Created build: %v/%v", newb.Namespace, newb.Name)
			return nil
		})
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}
}
