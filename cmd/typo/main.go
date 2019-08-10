package main

import (
	"github.com/mattmoor/knobots/pkg/handler"
	"github.com/mattmoor/knobots/pkg/typo"
)

func main() {
	handler.Main(typo.New())
}
