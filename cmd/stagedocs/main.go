package main

import (
	"github.com/mattmoor/knobots/pkg/handler"
	"github.com/mattmoor/knobots/pkg/stagedocs"
)

func main() {
	handler.Main(stagedocs.New)
}
