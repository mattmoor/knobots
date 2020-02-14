package main

import (
	"github.com/mattmoor/knobots/pkg/handler"
	"github.com/mattmoor/knobots/pkg/handler/copyright"
)

func main() {
	handler.Main(copyright.New)
}
