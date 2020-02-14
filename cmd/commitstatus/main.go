package main

import (
	"github.com/mattmoor/knobots/pkg/handler"
	"github.com/mattmoor/knobots/pkg/handler/commitstatus"
)

func main() {
	handler.Main(commitstatus.New)
}
