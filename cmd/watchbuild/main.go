package main

import (
	"github.com/mattmoor/knobots/pkg/handler"
	"github.com/mattmoor/knobots/pkg/watchbuild"
)

func main() {
	handler.Main(watchbuild.New)
}
