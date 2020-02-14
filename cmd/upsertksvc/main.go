package main

import (
	"github.com/mattmoor/knobots/pkg/handler"
	"github.com/mattmoor/knobots/pkg/handler/upsertksvc"
)

func main() {
	handler.Main(upsertksvc.New)
}
