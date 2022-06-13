package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	f "github.com/rprtr258/goflow/fun"
	r "github.com/rprtr258/goflow/result"
	s "github.com/rprtr258/goflow/stream"
	vk "github.com/rprtr258/vk-utils/pkg"
)

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

func parseGroupURL(groupURL string) r.Result[string] {
	var groupName string
	if _, err := fmt.Sscanf(groupURL, "https://vk.com/%s", &groupName); err != nil {
		return r.Err[string](err)
	}
	return r.Success(groupName)
}

func newRepostCommand(client *vk.VKClient) *cobra.Command {
	var postURL string
	repostsCmd := cobra.Command{
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
	repostsCmd.Flags().StringVarP(&postURL, "url", "u", "", "url of vk post")
	return &repostsCmd
}

func parseUserIDsList(ls []string) r.Result[[]vk.UserID] {
	res := make([]vk.UserID, 0, len(ls))
	for _, id := range ls {
		userID, err := strconv.ParseInt(id, 10, 32)
		if err != nil {
			return r.Err[[]vk.UserID](fmt.Errorf("error parsing user id: %s", id))
		}
		res = append(res, vk.UserID(userID))
	}
	return r.Success(res)
}

func parsePostsList(ls []string) r.Result[[]vk.PostID] {
	res := make([]vk.PostID, 0, len(ls))
	for _, id := range ls {
		var (
			ownerID vk.UserID
			postID  uint
		)
		if _, err := fmt.Sscanf(id, "%d_%d", &ownerID, &postID); err != nil {
			return r.Err[[]vk.PostID](fmt.Errorf("error parsing post id: %s", id))
		}
		res = append(res, vk.PostID{
			OwnerID: ownerID,
			ID:      postID,
		})
	}
	return r.Success(res)
}

func appendIfError[A any](errors *[]error, x r.Result[A]) {
	if x.IsErr() {
		*errors = append(*errors, x.UnwrapErr())
	}
}

func newCountCmd(client *vk.VKClient) *cobra.Command {
	var (
		groups         []string
		postLikers     []string
		friends        []string
		followers      []string
		postCommenters []string
		userProvided   []string
	)
	countCmd := cobra.Command{
		Use:   "count",
		Short: "Counts how many sets users belong to. Useful for uniting and intersecting user sets.",
		Args:  cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			var errors []error

			groupIDs := parseUserIDsList(groups)
			appendIfError(&errors, groupIDs)
			friendIDs := parseUserIDsList(friends)
			appendIfError(&errors, friendIDs)
			followerIDs := parseUserIDsList(followers)
			appendIfError(&errors, followerIDs)
			userIDs := parseUserIDsList(userProvided)
			appendIfError(&errors, userIDs)
			postLikerIDs := parsePostsList(postLikers)
			appendIfError(&errors, postLikerIDs)
			postCommenterIDs := parsePostsList(postCommenters)
			appendIfError(&errors, postCommenterIDs)

			if errors != nil {
				return fmt.Errorf(strings.Join(s.CollectToSlice(s.Map(s.FromSlice(errors), (error).Error)), "\n"))
			}

			for _, userInfoCount := range vk.MembershipCount(client, vk.UserSets{
				GroupMembers: groupIDs.Unwrap(),
				Friends:      friendIDs.Unwrap(),
				Followers:    followerIDs.Unwrap(),
				Users:        userIDs.Unwrap(),
				Likers:       postLikerIDs.Unwrap(),
				Commenters:   postCommenterIDs.Unwrap(),
			}) {
				fmt.Printf("%d: %s %s - %d\n", userInfoCount.Left.ID, userInfoCount.Left.FirstName, userInfoCount.Left.SecondName, userInfoCount.Right)
			}
			return nil
		},
		Example: "vkutils count --friends 168715495 --groups -187839235 --post-likers 107904132_1371",
	}
	countCmd.Flags().StringSliceVarP(&groups, "groups", "g", []string{}, "group id members of which to scan")
	countCmd.Flags().StringSliceVarP(&friends, "friends", "r", []string{}, "user id friends of which to scan")
	countCmd.Flags().StringSliceVarP(&followers, "followers", "w", []string{}, "user id followers of which to scan")
	countCmd.Flags().StringSliceVarP(&postLikers, "post-likers", "l", []string{}, "post id likers of which to scan")
	countCmd.Flags().StringSliceVarP(&postCommenters, "commenters", "c", []string{}, "post id commenters of which to scan")
	countCmd.Flags().StringSliceVarP(&userProvided, "users", "u", []string{}, "user ids to scan")
	return &countCmd
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

func newRootCmd() *cobra.Command {
	var client vk.VKClient
	rootCmd := cobra.Command{
		Use:   "vkutils",
		Short: "VK data extraction tools. Need VK_ACCESS_TOKEN env var to work with VK api.",
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			if _, presented := os.LookupEnv("VK_ACCESS_TOKEN"); !presented {
				return errors.New("VK_ACCESS_TOKEN was not found in env vars")
			}
			client = vk.NewVKClient(os.Getenv("VK_ACCESS_TOKEN"))
			return nil
		},
	}
	rootCmd.AddCommand(newRepostCommand(&client))
	rootCmd.AddCommand(newDumpCommand(&client))
	rootCmd.AddCommand(newCountCmd(&client))
	return &rootCmd
}

func main() {
	rootCmd := newRootCmd()

	start := time.Now()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
	log.Printf("Time elapsed %v", time.Since(start))
}
