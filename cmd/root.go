package cmd

import (
	"log"
	"time"

	vk "github.com/rprtr258/vk-utils/pkg"
	"github.com/urfave/cli/v2"
)

var (
	client   vk.VKClient
	_verbose bool
	_vkToken string
	start    time.Time
	RootCmd  = &cli.App{
		Name:  "vkutils",
		Usage: "VK data extraction tools. Need VK_ACCESS_TOKEN env var to work with VK api.",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "verbose",
				Aliases:     []string{"v"},
				Value:       false,
				Usage:       "log api calls",
				Destination: &_verbose,
			},
			&cli.StringFlag{
				Name:        "token",
				Usage:       "VK api token",
				Required:    true,
				EnvVars:     []string{"VK_ACCESS_TOKEN"},
				Destination: &_vkToken,
			},
		},
		Commands: []*cli.Command{
			dumpCmd,
			repostsCmd,
			countCmd,
		},
		Before: func(ctx *cli.Context) error {
			client = vk.NewVKClient(_vkToken, _verbose)
			start = time.Now()
			return nil
		},
		After: func(ctx *cli.Context) error {
			log.Printf("Time elapsed %v", time.Since(start))
			return nil
		},
	}
)
