package vkutils

import (
	"sort"

	f "github.com/rprtr258/goflow/fun"
	"github.com/rprtr258/goflow/slice"
	s "github.com/rprtr258/goflow/stream"
)

type UserSets struct {
	GroupMembers []UserID
	Friends      []UserID
	Followers    []UserID
	Likers       []PostID
	// TODO: user provided
	// TODO: commenters
}

func MembershipCount(client *VKClient, include UserSets) []f.Pair[User, int] {
	// TODO: parallelize
	chans := s.Chain(
		s.Map(s.FromSlice(include.Friends), client.getFriends),
		s.Map(s.FromSlice(include.GroupMembers), client.getGroupMembers),
		s.Map(s.FromSlice(include.Followers), client.getFollowers),
		s.Map(s.FromSlice(include.Likers), func(postID PostID) s.Stream[User] { return client.getLikes(postID.OwnerID, postID.PostID) }),
	)
	mp := s.Reduce(f.NewEmptyCounter[User](), f.CounterPlus[User], s.Map(chans, s.CollectCounter[User]))
	res := slice.FromMap(mp)
	sort.Slice(res, func(i, j int) bool { return res[i].Right > res[j].Right })
	return res
}
