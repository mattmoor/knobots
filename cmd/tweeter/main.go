package main

import (
	"github.com/mattmoor/knobots/pkg/handler"
	"github.com/mattmoor/knobots/pkg/tweeter"
)

func main() {
	handler.Main(tweeter.New)
}
