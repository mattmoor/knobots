package main

import (
	"github.com/mattmoor/knobots/pkg/baz"
	"github.com/mattmoor/knobots/pkg/handler"
)

func main() {
	handler.Main(baz.New)
}
