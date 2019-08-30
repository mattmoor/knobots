package main

import (
	"github.com/mattmoor/knobots/pkg/foo"
	"github.com/mattmoor/knobots/pkg/handler"
)

func main() {
	handler.Main(foo.New())
}
