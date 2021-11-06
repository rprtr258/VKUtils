package main

import (
	"log"
	"os"

	"github.com/rprtr258/vk-utils/cmd"
)

func main() {
	if err := cmd.RootCmd.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
