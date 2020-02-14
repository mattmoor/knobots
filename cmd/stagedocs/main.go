package main

import (
	"github.com/mattmoor/knobots/pkg/handler"
	"github.com/mattmoor/knobots/pkg/handler/stagedocs"
)

func main() {
	handler.Main(stagedocs.New)
}
