package main

import (
	"github.com/mattmoor/knobots/pkg/handler"
	"github.com/mattmoor/knobots/pkg/handler/dailybuild"
)

func main() {
	handler.Main(dailybuild.New)
}
