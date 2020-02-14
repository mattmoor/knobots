package main

import (
	"github.com/mattmoor/knobots/pkg/handler"
	"github.com/mattmoor/knobots/pkg/handler/tweeter"
)

func main() {
	handler.Main(tweeter.New)
}
