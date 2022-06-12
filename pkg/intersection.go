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

func count[A comparable](xs s.Stream[A]) map[A]int {
	res := make(map[A]int)
	s.ForEach(xs, func(a A) { res[a]++ })
	return res
}

func sumCounters[A comparable](a, b map[A]int) map[A]int {
	res := make(map[A]int, len(a))
	for k, v := range a {
		res[k] += v
	}
	for k, v := range b {
		res[k] += v
	}
	return res
}

func MembershipCount(client *VKClient, include UserSets) []f.Pair[UserInfo, int] {
	// TODO: parallelize
	chans := s.Chain(
		s.Map(s.FromSlice(include.Friends), client.getFriends),
		s.Map(s.FromSlice(include.GroupMembers), client.getGroupMembers),
		s.Map(s.FromSlice(include.Followers), client.getFollowers),
		s.Map(s.FromSlice(include.Likers), func(postID PostID) s.Stream[UserInfo] { return client.getLikes(postID.OwnerID, postID.PostID) }),
	)
	mp := s.Reduce(map[UserInfo]int{}, sumCounters[UserInfo], s.Map(chans, count[UserInfo]))
	res := make([]f.Pair[UserInfo, int], 0, len(mp))
	for k, v := range mp {
		res = append(res, f.NewPair(k, v))
	}
	sort.Slice(res, func(i, j int) bool { return res[i].Right > res[j].Right })
	return res
}
