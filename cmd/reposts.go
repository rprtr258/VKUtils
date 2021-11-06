package cmd

import (
	"fmt"

	f "github.com/rprtr258/go-flow/fun"
	r "github.com/rprtr258/go-flow/result"
	s "github.com/rprtr258/go-flow/stream"
	vk "github.com/rprtr258/vk-utils/pkg"
	"github.com/urfave/cli/v2"
)

var (
	_postURL          string
	repostSearchLimit uint
	repostsCmd        = &cli.Command{
		Name: "reposts",
		Usage: `Find reposters from commenters, group members, likers. Won't find all of reposters.
Example:
	vkutils reposts -u https://vk.com/wall-2158488_651604
`,
		Action: func(*cli.Context) error {
			client.RepostSearchLimit = repostSearchLimit
			sharersStream := r.FlatMap(
				parsePostURL(_postURL),
				func(postID vk.PostID) r.Result[s.Stream[vk.PostID]] {
					return vk.GetReposters(client, postID)
				},
			)
			return r.Fold(
				sharersStream,
				func(ss s.Stream[vk.PostID]) error {
					s.ForEach(
						ss,
						func(s vk.PostID) {
							fmt.Printf("https://vk.com/wall%d_%d\n", s.OwnerID, s.ID)
						},
					)
					return nil
				},
				f.Identity[error],
			)
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "url",
				Aliases:     []string{"u"},
				Required:    true,
				Usage:       "url of vk post",
				Destination: &_postURL,
			},
			&cli.UintFlag{
				Name:        "limit",
				Aliases:     []string{"i"},
				Value:       30000000,
				Usage:       "max diff between post time and repost time to look",
				Destination: &repostSearchLimit,
			},
		},
	}
)

func parsePostURL(url string) r.Result[vk.PostID] {
	var postID vk.PostID
	if _, err := fmt.Sscanf(url, "https://vk.com/wall%d_%d", &postID.OwnerID, &postID.ID); err != nil {
		return r.Err[vk.PostID](err)
	}
	return r.Success(postID)
}
