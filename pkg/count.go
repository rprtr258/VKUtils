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

func MembershipCount(client VKClient, include UserSets) []f.Pair[User, uint] {
	chans := s.Gather([]s.Stream[s.Stream[User]]{
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
		s.Map(s.FromSlice(include.Likers), client.getLikes),
		s.Map(s.FromSlice(include.Commenters), client.GetComments),
	})
	counters := s.Map(chans, s.CollectCounter[User])
	resCounter := s.Reduce(f.NewCounter[User](), f.CounterPlus[User], counters)
	res := slice.FromMap(resCounter)
	sort.Slice(res, func(i, j int) bool { return res[i].Right > res[j].Right })
	return res
}
