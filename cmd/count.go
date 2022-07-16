package cmd

import (
	"fmt"
	"strconv"
	"strings"

	r "github.com/rprtr258/go-flow/result"
	s "github.com/rprtr258/go-flow/stream"
	vk "github.com/rprtr258/vk-utils/pkg"
	"github.com/spf13/cobra"
)

var (
	groups         []string
	postLikers     []string
	friends        []string
	followers      []string
	postCommenters []string
	userProvided   []string
	countCmd       = cobra.Command{
		Use:     "count",
		Short:   "Counts how many sets users belong to. Useful for uniting and intersecting user sets.",
		Args:    cobra.MaximumNArgs(0),
		RunE:    run,
		Example: "vkutils count --friends 168715495 --groups -187839235 --post-likers 107904132_1371",
	}
)

func init() {
	countCmd.Flags().StringSliceVarP(&groups, "groups", "g", []string{}, "group id members of which to scan")
	countCmd.Flags().StringSliceVarP(&friends, "friends", "r", []string{}, "user id friends of which to scan")
	countCmd.Flags().StringSliceVarP(&followers, "followers", "w", []string{}, "user id followers of which to scan")
	countCmd.Flags().StringSliceVarP(&postLikers, "post-likers", "l", []string{}, "post id likers of which to scan")
	countCmd.Flags().StringSliceVarP(&postCommenters, "commenters", "c", []string{}, "post id commenters of which to scan")
	countCmd.Flags().StringSliceVarP(&userProvided, "users", "u", []string{}, "user ids to scan")

	rootCmd.AddCommand(&countCmd)
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

func run(cmd *cobra.Command, args []string) error {
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
}
