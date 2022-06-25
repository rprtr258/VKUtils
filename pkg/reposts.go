package vkutils

import (
	"fmt"
	"log"

	f "github.com/rprtr258/go-flow/fun"
	r "github.com/rprtr258/go-flow/result"
	s "github.com/rprtr258/go-flow/stream"
)

type PageSize uint

const (
	wallGetPageSize         = PageSize(100)
	userCheckRepostsThreads = 10
)

// Returns either found (or not found) repost's post id
func findRepost(client VKClient, userID UserID, postID PostID, postDate uint) f.Option[uint] {
	params := MakeUrlValues(map[string]any{
		"owner_id": userID,
		"count":    wallGetPageSize,
	})
	w0Result := client.getWallPosts(params, "offset", "0")
	if w0Result.IsErr() || w0Result.Unwrap().Response.Count == 0 {
		return f.None[uint]()
	}
	w0 := w0Result.Unwrap()
	isRepostPredicate := func(post Post) bool {
		copyHistory := post.CopyHistory
		return len(copyHistory) != 0 && copyHistory[0].ID == postID.ID && copyHistory[0].OwnerID == postID.OwnerID
	}
	if len(w0.Response.Items) == 0 {
		return f.None[uint]()
	}
	if isRepostPredicate(w0.Response.Items[0]) {
		return f.Some(w0.Response.Items[0].ID)
	}
	if w0.Response.Items[len(w0.Response.Items)-1].Date <= postDate {
		for _, post := range w0.Response.Items[1:] {
			if isRepostPredicate(post) {
				return f.Some(post.ID)
			}
		}
		return f.None[uint]()
	}
	log.Printf("SEARCHING REPOST IN USER %d WITH %d ENTRIES\n", userID, w0.Response.Count)
	l := uint(1)
	r := (w0.Response.Count + uint(wallGetPageSize) - 1) / uint(wallGetPageSize)
	for r-l > 1 {
		m := (l + r) / 2
		w0Result = client.getWallPosts(params, "offset", fmt.Sprint(m*uint(wallGetPageSize)))
		if w0Result.IsErr() || w0Result.Unwrap().Response.Count == 0 {
			return f.None[uint]()
		}
		w0 = w0Result.Unwrap()
		if w0.Response.Items[0].Date >= postDate && w0.Response.Items[len(w0.Response.Items)-1].Date <= postDate {
			for _, post := range w0.Response.Items {
				if isRepostPredicate(post) {
					return f.Some(post.ID)
				}
			}
			l--
			break
		} else if w0.Response.Items[0].Date < postDate {
			r = m
		} else {
			l = m
		}
	}
	for l > 0 {
		w0Result = client.getWallPosts(params, "offset", fmt.Sprint(l*uint(wallGetPageSize)))
		if w0Result.IsErr() || w0Result.Unwrap().Response.Count == 0 {
			return f.None[uint]()
		}
		// TODO: give user ability to control search limit
		if w0.Response.Items[0].Date-postDate > 30000000 {
			return f.None[uint]()
		}
		w0 = w0Result.Unwrap()
		for _, post := range w0.Response.Items {
			if isRepostPredicate(post) {
				return f.Some(post.ID)
			}
		}
		l--
	}
	return f.None[uint]()
}

func userInfoToUserID(info User) UserID {
	return info.ID
}

func getPotentialUserIDs(client VKClient, postID PostID) s.Stream[UserID] {
	// scan commenters
	commenters := s.Map(client.GetComments(postID), userInfoToUserID)

	// scan likers
	likers := s.Map(client.getLikes(postID), userInfoToUserID)

	// scan group members/friends of post owner
	var potentialUserIDs s.Stream[UserID]
	if postID.OwnerID < 0 { // owner is group
		potentialUserIDs = s.Map(client.getGroupMembers(postID.OwnerID), userInfoToUserID)
	} else { // owner is user
		potentialUserIDs = s.Map(client.getFriends(postID.OwnerID), userInfoToUserID)
	}

	return s.Gather([]s.Stream[UserID]{commenters, likers, potentialUserIDs})
}

func getCheckedIDs(client VKClient, postID PostID, postDate uint, userIDs s.Stream[UserID]) s.Stream[PostID] {
	findRepost := func(userID UserID) f.Option[PostID] {
		return f.Map(
			findRepost(client, userID, postID, postDate),
			func(postID uint) PostID {
				return PostID{userID, postID}
			},
		)
	}
	return s.Gather(s.CollectToSlice(s.Map(
		s.FromSlice(s.Scatter(userIDs, userCheckRepostsThreads)),
		func(userIDs s.Stream[UserID]) s.Stream[PostID] {
			return s.MapFilter(userIDs, findRepost)
		},
	)))
}

func GetReposters(client VKClient, postID PostID) r.Result[s.Stream[PostID]] {
	return r.Map(
		client.getPostTime(postID),
		func(postDate uint) s.Stream[PostID] {
			uniqueIDs := s.Unique(getPotentialUserIDs(client, postID))
			return getCheckedIDs(client, postID, postDate, uniqueIDs)
		},
	)
}
