package main

import (
	"github.com/mattmoor/knobots/pkg/handler"
	"github.com/mattmoor/knobots/pkg/handler/gotool"
)

func main() {
	handler.Main(gotool.New)
}
