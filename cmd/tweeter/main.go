package main

import (
	"flag"

	"github.com/mattmoor/knobots/pkg/handler"
	"github.com/mattmoor/knobots/pkg/tweeter"
)

func main() {
	flag.Parse()

	handler.Main(tweeter.New())
}
