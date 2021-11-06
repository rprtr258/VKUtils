package cmd

import (
	"fmt"
	"time"

	r "github.com/rprtr258/go-flow/result"
	s "github.com/rprtr258/go-flow/stream"
	vk "github.com/rprtr258/vk-utils/pkg"
	"github.com/urfave/cli/v2"
)

var (
	_groupURL string
	dumpCmd   = &cli.Command{
		Name: "dumpwall",
		Usage: `List group posts in reversed order (from old to new).
Example:
	vkutils revposts https://vk.com/abobus_official
`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "url",
				Aliases:     []string{"u"},
				Required:    true,
				Usage:       "url of vk group",
				Destination: &_groupURL,
			},
		},
		Action: func(*cli.Context) error {
			groupName := parseGroupURL(_groupURL)
			vk.GetPosts(client, groupName.Unwrap()).Consume(
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
	}
)

func parseGroupURL(groupURL string) r.Result[string] {
	var groupName string
	if _, err := fmt.Sscanf(groupURL, "https://vk.com/%s", &groupName); err != nil {
		return r.Err[string](err)
	}
	return r.Success(groupName)
}
