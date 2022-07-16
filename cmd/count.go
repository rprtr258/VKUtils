package cmd

import (
	"fmt"
	"strconv"
	"strings"

	r "github.com/rprtr258/go-flow/result"
	s "github.com/rprtr258/go-flow/stream"
	vk "github.com/rprtr258/vk-utils/pkg"
	"github.com/urfave/cli/v2"
)

var (
	_groups         = cli.NewStringSlice()
	_postLikers     = cli.NewStringSlice()
	_friends        = cli.NewStringSlice()
	_followers      = cli.NewStringSlice()
	_postCommenters = cli.NewStringSlice()
	_userProvided   = cli.NewStringSlice()
	countCmd        = &cli.Command{
		Name: "count",
		Usage: `Counts how many sets users belong to. Useful for uniting and intersecting user sets.
Example:
	vkutils count --friends 168715495 --groups -187839235 --post-likers 107904132_1371
`,
		Action: run,
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Destination: _groups,
				Name:        "groups",
				Aliases:     []string{"g"},
				Usage:       "group id members of which to scan",
			},
			&cli.StringSliceFlag{
				Destination: _friends,
				Name:        "friends",
				Aliases:     []string{"r"},
				Usage:       "user id friends of which to scan",
			},
			&cli.StringSliceFlag{
				Destination: _followers,
				Name:        "followers",
				Aliases:     []string{"w"},
				Usage:       "user id followers of which to scan",
			},
			&cli.StringSliceFlag{
				Destination: _postLikers,
				Name:        "post-likers",
				Aliases:     []string{"l"},
				Usage:       "post id likers of which to scan",
			},
			&cli.StringSliceFlag{
				Destination: _postCommenters,
				Name:        "commenters",
				Aliases:     []string{"c"},
				Usage:       "post id commenters of which to scan",
			},
			&cli.StringSliceFlag{
				Destination: _userProvided,
				Name:        "users",
				Aliases:     []string{"u"},
				Usage:       "user ids to scan",
			},
		},
	}
)

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

func run(*cli.Context) error {
	var errors []error

	groupIDs := parseUserIDsList(_groups.Value())
	appendIfError(&errors, groupIDs)
	friendIDs := parseUserIDsList(_friends.Value())
	appendIfError(&errors, friendIDs)
	followerIDs := parseUserIDsList(_followers.Value())
	appendIfError(&errors, followerIDs)
	userIDs := parseUserIDsList(_userProvided.Value())
	appendIfError(&errors, userIDs)
	postLikerIDs := parsePostsList(_postLikers.Value())
	appendIfError(&errors, postLikerIDs)
	postCommenterIDs := parsePostsList(_postCommenters.Value())
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
