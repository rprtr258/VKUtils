package vkutils

import (

	// r "github.com/rprtr258/goflow/result"

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

func GetIntersection(client *VKClient, include UserSets) f.Set[UserInfo] {
	// TODO: parallelize
	userIDsStreams := s.CollectToSlice(s.Chain(
		s.Map(s.FromSlice(include.Friends), client.getFriends),
		s.Map(s.FromSlice(include.GroupMembers), client.getGroupMembers),
		s.Map(s.FromSlice(include.Followers), client.getFollowers),
		s.Map(s.FromSlice(include.Likers), func(postID PostID) s.Stream[UserInfo] { return client.getLikes(postID.OwnerID, postID.PostID) }),
	))
	chans := s.FromSlice(userIDsStreams)
	// TODO: find most-intersected user ids?
	first := s.CollectToSet(chans.Next().Unwrap())
	sets := s.Map(chans, s.CollectToSet[UserInfo])
	return s.Reduce(first, f.Intersect[UserInfo], sets)
}
