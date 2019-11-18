package main

import (
	"flag"
	"log"

	buildclientset "github.com/knative/build/pkg/client/clientset/versioned"
	"k8s.io/client-go/rest"

	"github.com/mattmoor/knobots/pkg/handler"
	"github.com/mattmoor/knobots/pkg/tweeter"
)

func main() {
	flag.Parse()

	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Error building kubeconfig: %v", err)
	}

	bc, err := buildclientset.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Unable to build Build client: %v", err)
	}

	handler.Main(tweeter.New(bc))
}
