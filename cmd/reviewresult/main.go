package main

import (
	"github.com/mattmoor/knobots/pkg/handler"
	"github.com/mattmoor/knobots/pkg/handler/reviewresult"
)

func main() {
	handler.Main(reviewresult.New)
}
