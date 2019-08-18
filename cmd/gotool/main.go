package main

import (
	"flag"
	"log"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/mattmoor/knobots/pkg/gotool"
	"github.com/mattmoor/knobots/pkg/handler"
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

	handler.Main(gotool.New(cfg))
}
