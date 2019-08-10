package main

import (
	"github.com/mattmoor/knobots/pkg/handler"
	"github.com/mattmoor/knobots/pkg/reviewresult"
)

func main() {
	handler.Main(reviewresult.New())
}
