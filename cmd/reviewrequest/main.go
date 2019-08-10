package main

import (
	"github.com/mattmoor/knobots/pkg/handler"
	"github.com/mattmoor/knobots/pkg/reviewrequest"
)

func main() {
	handler.Main(reviewrequest.New())
}
