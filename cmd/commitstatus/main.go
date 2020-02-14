package main

import (
	"github.com/mattmoor/knobots/pkg/commitstatus"
	"github.com/mattmoor/knobots/pkg/handler"
)

func main() {
	handler.Main(commitstatus.New)
}
