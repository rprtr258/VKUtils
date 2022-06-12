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

func parseUserIDsList(ls []string) r.Result[[]vk.UserID] {
	res := make([]vk.UserID, 0, len(ls))
	for _, id := range ls {
		userID, err := strconv.ParseInt(id, 10, 32)
		if err != nil {
			return r.Err[[]vk.UserID](err)
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
			return r.Err[[]vk.PostID](err)
		}
		res = append(res, vk.PostID{
			OwnerID: ownerID,
			PostID:  postID,
		})
	}
	return r.Success(res)
}

func parsePostURL(url string) r.Result[f.Pair[vk.UserID, uint]] {
	var (
		ownerID vk.UserID
		postID  uint
	)
	if _, err := fmt.Sscanf(url, "https://vk.com/wall%d_%d", &ownerID, &postID); err != nil {
		return r.Err[f.Pair[vk.UserID, uint]](err)
	}
	return r.Success(f.NewPair(ownerID, postID))
}

func parseGroupURL(groupURL string) r.Result[string] {
	var groupName string
	if _, err := fmt.Sscanf(groupURL, "https://vk.com/%s", &groupName); err != nil {
		return r.Err[string](err)
	}
	return r.Success(groupName)
}

func main() {
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

	var postURL string
	repostsCmd := cobra.Command{
		Use:   "reposts -u",
		Short: "Find reposters.",
		Long:  `Find reposters from commenters, group members, likers. Won't find all of reposters.`,
		Args:  cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			sharersStream := r.FlatMap(
				parsePostURL(postURL),
				func(postID f.Pair[vk.UserID, uint]) r.Result[s.Stream[vk.Sharer]] {
					return vk.GetReposters(&client, postID.Left, postID.Right)
				},
			)
			return r.Fold(
				sharersStream,
				func(ss s.Stream[vk.Sharer]) error {
					s.ForEach(
						s.Take(ss, 1), // TODO: remove
						func(s vk.Sharer) {
							fmt.Printf("FOUND REPOST: https://vk.com/wall%d_%d\n", s.UserID, s.RepostID)
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
	rootCmd.AddCommand(&repostsCmd)

	// TODO: dump groups/profiles posts into database (own format?)
	var groupURL string
	revPostsUrl := cobra.Command{ // TODO: rename to dump posts
		Use:   "revposts",
		Short: "List group posts in reversed order (from old to new).",
		Args:  cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			groupName := parseGroupURL(groupURL)
			r.FoldConsume(
				vk.GetReversedPosts(&client, groupName.Unwrap()),
				func(x s.Stream[vk.Post]) {
					s.ForEach(
						x,
						func(p vk.Post) {
							fmt.Printf("https://vk.com/wall%d_%d\n", p.Owner, p.ID)
							fmt.Println("Date: ", time.Unix(int64(p.Date), 0))
							fmt.Println(p.Text)
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
	revPostsUrl.Flags().StringVarP(&groupURL, "url", "u", "", "url of vk group")
	revPostsUrl.MarkFlagRequired("url")
	rootCmd.AddCommand(&revPostsUrl)

	var (
		groups     []string
		postLikers []string
		friends    []string
		followers  []string
	)
	intersectionCmd := cobra.Command{
		Use:   "intersection",
		Short: "Find users sets intersection.",
		Args:  cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			var errors []string

			groupIDs := parseUserIDsList(groups)
			if groupIDs.IsErr() {
				errors = append(errors, fmt.Sprintf("error parsing group ids: %v", groupIDs.UnwrapErr()))
			}

			friendIDs := parseUserIDsList(friends)
			if friendIDs.IsErr() {
				errors = append(errors, fmt.Sprintf("error parsing friend ids: %v", friendIDs.UnwrapErr()))
			}

			followerIDs := parseUserIDsList(followers)
			if followerIDs.IsErr() {
				errors = append(errors, fmt.Sprintf("error parsing follower ids: %v", followerIDs.UnwrapErr()))
			}

			postLikerIDs := parsePostsList(postLikers)
			if postLikerIDs.IsErr() {
				errors = append(errors, fmt.Sprintf("error parsing post likers ids: %v", postLikerIDs.UnwrapErr()))
			}

			if errors != nil {
				return fmt.Errorf(strings.Join(errors, "\n"))
			}

			for userInfo := range vk.GetIntersection(&client, vk.UserSets{
				GroupMembers: groupIDs.Unwrap(),
				Friends:      friendIDs.Unwrap(),
				Followers:    followerIDs.Unwrap(),
				Likers:       postLikerIDs.Unwrap(),
			}) {
				fmt.Printf("%d: %s %s\n", userInfo.ID, userInfo.FirstName, userInfo.SecondName)
			}
			return nil
		},
		Example: "vkutils intersection --friends 168715495 --groups -187839235 --post-likers 107904132_1371",
	}
	intersectionCmd.Flags().StringSliceVarP(&groups, "groups", "g", []string{}, "group ids members of which to ")
	intersectionCmd.Flags().StringSliceVarP(&postLikers, "post-likers", "l", []string{}, "group ids members of which to ")
	intersectionCmd.Flags().StringSliceVarP(&friends, "friends", "r", []string{}, "group ids members of which to ")
	intersectionCmd.Flags().StringSliceVarP(&followers, "followers", "w", []string{}, "group ids members of which to ")
	rootCmd.AddCommand(&intersectionCmd)

	start := time.Now()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
	log.Printf("Time elapsed %v", time.Since(start))
}
