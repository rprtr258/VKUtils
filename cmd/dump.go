package cmd

import (
	"fmt"
	"time"

	r "github.com/rprtr258/goflow/result"
	s "github.com/rprtr258/goflow/stream"
	vk "github.com/rprtr258/vk-utils/pkg"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newDumpCommand(client))
}

func parseGroupURL(groupURL string) r.Result[string] {
	var groupName string
	if _, err := fmt.Sscanf(groupURL, "https://vk.com/%s", &groupName); err != nil {
		return r.Err[string](err)
	}
	return r.Success(groupName)
}

func newDumpCommand(client *vk.VKClient) *cobra.Command {
	// TODO: dump groups/profiles posts into database (own format?)
	var groupURL string
	dumpCmd := cobra.Command{
		Use:   "dumpwall",
		Short: "List group posts in reversed order (from old to new).",
		Args:  cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			groupName := parseGroupURL(groupURL)
			r.FoldConsume(
				vk.GetPosts(client, groupName.Unwrap()),
				func(x s.Stream[vk.Post]) {
					s.ForEach(
						x,
						func(p vk.Post) {
							fmt.Printf("https://vk.com/wall%d_%d\n", p.Owner, p.ID)
							fmt.Println("Date: ", time.Unix(int64(p.Date), 0))
							fmt.Println(p.Text)
							if len(p.CopyHistory) > 0 {
								fmt.Println("Repost: ", p.CopyHistory[0])
							}
							fmt.Println()
						},
					)
				},
				func(err error) {
					fmt.Printf("error: %v\n", err)
				},
			)
			return nil
		},
		Example: "vkutils revposts https://vk.com/abobus_official",
	}
	dumpCmd.Flags().StringVarP(&groupURL, "url", "u", "", "url of vk group")
	dumpCmd.MarkFlagRequired("url")
	return &dumpCmd
}
