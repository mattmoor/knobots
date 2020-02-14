package main

import (
	"github.com/mattmoor/knobots/pkg/handler"
	"github.com/mattmoor/knobots/pkg/handler/donotsubmit"
)

func main() {
	handler.Main(donotsubmit.New)
}
