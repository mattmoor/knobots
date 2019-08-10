package main

import (
	"github.com/mattmoor/knobots/pkg/copyright"
	"github.com/mattmoor/knobots/pkg/handler"
)

func main() {
	handler.Main(copyright.New())
}
