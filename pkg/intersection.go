package vkutils

import (
	"sort"

	f "github.com/rprtr258/goflow/fun"
	s "github.com/rprtr258/goflow/stream"
)

// PostID is pair of post author ID and post index.
type PostID struct {
	OwnerID UserID
	PostID  uint
}

type UserSets struct {
	GroupMembers []UserID
	Friends      []UserID
	Followers    []UserID
	Likers       []PostID
	// TODO: user provided
	// TODO: commenters
}

func MembershipCount(client *VKClient, include UserSets) []f.Pair[UserInfo, int] {
	// TODO: parallelize
	chans := s.Chain(
		s.Map(s.FromSlice(include.Friends), client.getFriends),
		s.Map(s.FromSlice(include.GroupMembers), client.getGroupMembers),
		s.Map(s.FromSlice(include.Followers), client.getFollowers),
		s.Map(s.FromSlice(include.Likers), func(postID PostID) s.Stream[UserInfo] { return client.getLikes(postID.OwnerID, postID.PostID) }),
	)
	mp := s.Reduce(f.NewEmptyCounter[UserInfo](), f.CounterPlus[UserInfo], s.Map(chans, s.CollectCounter[UserInfo]))
	res := f.FromMap(mp)
	sort.Slice(res, func(i, j int) bool { return res[i].Right > res[j].Right })
	return res
}
