package main

import (
	"github.com/mattmoor/knobots/pkg/bar"
	"github.com/mattmoor/knobots/pkg/handler"
)

func main() {
	handler.Main(bar.New())
}
