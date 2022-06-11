package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"

	i "github.com/rprtr258/goflow/result"
	s "github.com/rprtr258/goflow/stream"
	vkutils "github.com/rprtr258/vk-utils/pkg"
)

func main() {
	var client vkutils.VKClient
	rootCmd := cobra.Command{
		Use:   "vkutils",
		Short: "VK data extraction tools. Need VK_ACCESS_TOKEN env var to work with VK api.",
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			if _, presented := os.LookupEnv("VK_ACCESS_TOKEN"); !presented {
				return errors.New("VK_ACCESS_TOKEN was not found in env vars")
			}
			client = vkutils.NewVKClient(os.Getenv("VK_ACCESS_TOKEN"))
			return nil
		},
	}

	var postURL string
	repostsCmd := cobra.Command{
		Use:   "reposts -u",
		Short: "Find reposters.",
		Long:  `Find reposters from commenters, group members, likers. Won't find all of reposters.`,
		Args:  cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			i.FoldConsume(
				vkutils.GetRepostersByPostURL(&client, postURL),
				func(ss s.Stream[vkutils.Sharer]) {
					s.ForEach(
						s.Take(ss, 1),
						func(s vkutils.Sharer) {
							fmt.Printf("FOUND REPOST: https://vk.com/wall%d_%d\n", s.UserID, s.RepostID)
						},
					)
				},
				func(err error) {
					log.Println("Error: ", err)
				},
			)
			return nil
		},
		Example: "vkutils reposts -u https://vk.com/wall-2158488_651604",
	}
	repostsCmd.Flags().StringVarP(&postURL, "url", "u", "", "url of vk post")
	rootCmd.AddCommand(&repostsCmd)

	// // TODO: dump to db/sqlite like query, filter by date range, reversed flag, in text, etc.
	// // TODO: search in different groups, profiles
	// // https://vk.com/app3876642
	// // https://vk.com/wall-2158488_651604
	// var groupUrl string
	// revPostsUrl := cobra.Command{ // TODO: rename to dump posts
	// 	Use:   "revposts",
	// 	Short: "List group posts in reversed order (from old to new).",
	// 	Args:  cobra.MaximumNArgs(0),
	// 	RunE: func(cmd *cobra.Command, args []string) error {
	// 		res, err := vkutils.GetReversedPosts(&client, groupUrl)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		log.Println(res)
	// 		return nil
	// 	},
	// 	Example: "fimgs cluster -n 4 girl.png",
	// }
	// revPostsUrl.Flags().StringVarP(&groupUrl, "url", "u", "", "url of vk group")
	// revPostsUrl.MarkFlagRequired("url")
	// rootCmd.AddCommand(&revPostsUrl)

	// intersectionCmd := cobra.Command{
	// 	Use:   "intersection",
	// 	Short: "Find users sets intersection.",
	// 	Args:  cobra.MaximumNArgs(0),
	// 	RunE: func(cmd *cobra.Command, args []string) error {
	// 		res := vkutils.GetIntersection(&client, &json.Decoder{})
	// 		log.Println(res)
	// 		return nil
	// 	},
	// 	Example: "fimgs cluster -n 4 girl.png",
	// }
	// intersectionCmd.Flags().StringVarP(&groupUrl, "url", "u", "", "url of vk group")
	// intersectionCmd.MarkFlagRequired("url")
	// rootCmd.AddCommand(&intersectionCmd)

	start := time.Now()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
	log.Printf("Time elapsed %v", time.Since(start))
}
