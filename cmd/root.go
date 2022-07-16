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
	client  vk.VKClient
	rootCmd = cobra.Command{
		Use:   "vkutils",
		Short: "VK data extraction tools. Need VK_ACCESS_TOKEN env var to work with VK api.",
	}
)

func init() {
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "log api calls")
}

func Execute() error {
	if _, presented := os.LookupEnv("VK_ACCESS_TOKEN"); !presented {
		return errors.New("VK_ACCESS_TOKEN was not found in env vars")
	}
	verbose, err := rootCmd.PersistentFlags().GetBool("verbose")
	if err != nil {
		return err
	}
	client = vk.NewVKClient(os.Getenv("VK_ACCESS_TOKEN"), verbose)
	start := time.Now()
	if err := rootCmd.Execute(); err != nil {
		return err
	}
	log.Printf("Time elapsed %v", time.Since(start))
	return nil
}
