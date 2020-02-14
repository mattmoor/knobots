package main

import (
	"github.com/mattmoor/knobots/pkg/donotsubmit"
	"github.com/mattmoor/knobots/pkg/handler"
)

func main() {
	handler.Main(donotsubmit.New)
}
