package vkutils

import (

	// r "github.com/rprtr258/goflow/result"

	f "github.com/rprtr258/goflow/fun"
	s "github.com/rprtr258/goflow/stream"
)

type UserSet = f.Set[UserID]

// func getFollowers(client *VKClient, userId UserID) s.Stream[UserID] {
// 	return client.getUserList("users.getFollowers", url.Values{
// 		"user_id": []string{fmt.Sprint(userId)},
// 	}, 1000)
// }

func intersectChans(chans s.Stream[s.Stream[UserID]]) UserSet {
	first := s.CollectToSet(chans.Next().Unwrap())
	sets := s.Map(chans, s.CollectToSet[UserID])
	return s.Reduce(first, f.Intersect[UserID], sets)
}

type PostID struct {
	OwnerID UserID
	PostID  uint
}

type UserSets struct {
	GroupMembers []UserID
	Friends      []UserID
	Followers    []UserID
	Likers       []PostID
	Sharers      []PostID // TODO: check inexactly
	// TODO: user provided
	// TODO: commenters
}

// TODO: also return users info if api gives it anyway (e.g. names)
func GetIntersection(client *VKClient, include UserSets) f.Set[UserID] {
	// TODO: parallelize
	userIDsStreams := make(
		[]s.Stream[UserID],
		0,
		len(include.GroupMembers)+len(include.Friends)+len(include.Followers)+len(include.Likers)+len(include.Sharers),
	)
	for _, userID := range include.Friends {
		userIDsStreams = append(userIDsStreams, client.getFriends(userID))
	}
	for _, groupID := range include.GroupMembers {
		userIDsStreams = append(userIDsStreams, client.getGroupMembers(groupID))
	}
	// groupSet   GroupMembers client.getGroupMembers(groupSet.GroupId)
	// profileSet Followers    getFollowers(client, profileSet.UserId)
	// postSet    Likers       client.getLikes(postSet.OwnerId, postSet.PostId)
	// postSet    Sharers      getSharers(client, postSet.OwnerId, postSet.PostId).Unwrap()

	// TODO: differentiate between empty map and empty intersection
	// TODO: find most-intersected user ids?
	// TODO: iterate over something different, maybe change algo
	return intersectChans(s.FromSlice(userIDsStreams))
}
