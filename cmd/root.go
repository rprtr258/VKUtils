package cmd

import (
	"errors"
	"log"
	"os"
	"time"

	vk "github.com/rprtr258/vk-utils/pkg"
	"github.com/spf13/cobra"
)

var (
	client  *vk.VKClient
	rootCmd = cobra.Command{
		Use:   "vkutils",
		Short: "VK data extraction tools. Need VK_ACCESS_TOKEN env var to work with VK api.",
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			if _, presented := os.LookupEnv("VK_ACCESS_TOKEN"); !presented {
				return errors.New("VK_ACCESS_TOKEN was not found in env vars")
			}
			tmp := vk.NewVKClient(os.Getenv("VK_ACCESS_TOKEN"))
			client = &tmp
			return nil
		},
	}
)

func Execute() {
	start := time.Now()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
	log.Printf("Time elapsed %v", time.Since(start))
}
