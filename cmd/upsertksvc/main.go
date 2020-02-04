package main

import (
	"flag"
	"log"

	"k8s.io/client-go/rest"
	clientset "knative.dev/serving/pkg/client/clientset/versioned"

	"github.com/mattmoor/knobots/pkg/handler"
	"github.com/mattmoor/knobots/pkg/upsertksvc"
)

func main() {
	flag.Parse()

	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Error building kubeconfig: %v", err)
	}

	bc, err := clientset.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Unable to build Tekton client: %v", err)
	}

	handler.Main(upsertksvc.New(bc))
}
