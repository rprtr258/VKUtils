package cmd

import (
	"fmt"

	f "github.com/rprtr258/go-flow/fun"
	r "github.com/rprtr258/go-flow/result"
	s "github.com/rprtr258/go-flow/stream"
	vk "github.com/rprtr258/vk-utils/pkg"
	"github.com/spf13/cobra"
)

var (
	postURL    string
	repostsCmd = cobra.Command{
		Use:   "reposts -u",
		Short: "Find reposters.",
		Long:  `Find reposters from commenters, group members, likers. Won't find all of reposters.`,
		Args:  cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			sharersStream := r.FlatMap(
				parsePostURL(postURL),
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
		Example: "vkutils reposts -u https://vk.com/wall-2158488_651604",
	}
)

func init() {
	repostsCmd.Flags().StringVarP(&postURL, "url", "u", "", "url of vk post")

	rootCmd.AddCommand(&repostsCmd)
}

func parsePostURL(url string) r.Result[vk.PostID] {
	var (
		ownerID vk.UserID
		postID  uint
	)
	if _, err := fmt.Sscanf(url, "https://vk.com/wall%d_%d", &ownerID, &postID); err != nil {
		return r.Err[vk.PostID](err)
	}
	return r.Success(vk.PostID{OwnerID: ownerID, ID: postID})
}
