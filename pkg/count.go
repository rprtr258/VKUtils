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
	Users        []UserID
	Likers       []PostID
	Commenters   []PostID
}

func MembershipCount(client VKClient, include UserSets) []f.Pair[User, int] {
	// TODO: parallelize
	chans := s.Chain(
		s.Map(s.FromSlice(include.Friends), client.getFriends),
		s.Map(s.FromSlice(include.GroupMembers), client.getGroupMembers),
		s.Map(s.FromSlice(include.Followers), client.getFollowers),
		s.Map(s.FromSlice(include.Users), func(userID UserID) s.Stream[User] {
			return s.Once(User{
				ID:         userID,
				FirstName:  "UNKNOWN",
				SecondName: "UNKNOWN",
			})
		}),
		s.Map(s.FromSlice(include.Likers), func(postID PostID) s.Stream[User] { return client.getLikes(postID) }),
		s.Map(s.FromSlice(include.Commenters), func(postID PostID) s.Stream[User] { return client.GetComments(postID) }),
	)
	mp := s.Reduce(f.NewEmptyCounter[User](), f.CounterPlus[User], s.Map(chans, s.CollectCounter[User]))
	res := slice.FromMap(mp)
	sort.Slice(res, func(i, j int) bool { return res[i].Right > res[j].Right })
	return res
}
