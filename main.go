package main

import (
	"log"

	"github.com/rprtr258/vk-utils/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
