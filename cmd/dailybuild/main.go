package main

import (
	"github.com/mattmoor/knobots/pkg/dailybuild"
	"github.com/mattmoor/knobots/pkg/handler"
)

func main() {
	handler.Main(dailybuild.New)
}
