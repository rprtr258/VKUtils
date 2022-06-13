package vkutils

import (
	"fmt"
	"log"
	"net/url"

	f "github.com/rprtr258/goflow/fun"
	r "github.com/rprtr258/goflow/result"
	s "github.com/rprtr258/goflow/stream"
)

type PageSize uint

const (
	wallGetPageSize         = PageSize(100)
	userCheckRepostsThreads = 10
)

type findRepostPager struct {
	client *VKClient
	offset uint
	total  f.Option[uint]
	params url.Values
}

// TODO: binary search
func (pager *findRepostPager) NextPage() r.Result[f.Option[[]Post]] {
	if pager.total.IsSome() && pager.offset >= pager.total.Unwrap() {
		return r.Success(f.None[[]Post]())
	}
	wallPosts := r.TryRecover(
		pager.client.getWallPosts(pager.params, "offset", fmt.Sprint(pager.offset)),
		func(err error) r.Result[WallPosts] {
			if errMsg, ok := err.(ApiCallError); ok {
				switch errMsg.vkError.Code {
				case accessDenied, userWasDeletedOrBanned, profileIsPrivate:
					return r.Success(WallPosts{Response: wallPostsResponse{
						Count: 0,
						Items: []Post{},
					}})
				}
			}
			log.Printf("Error getting user posts: %T(%[1]v)\n", err)
			return r.Err[WallPosts](err)
		},
	)
	return r.Map(
		wallPosts,
		func(page WallPosts) f.Option[[]Post] {
			// if pager.offset == 0 {
			// 	log.Printf("SEARCHING REPOST IN USER WITH %d ENTRIES\n", page.Response.Count)
			// }
			pager.offset += uint(wallGetPageSize)
			pager.total = f.Some(page.Response.Count)
			return f.Some(page.Response.Items)
		},
	)
}

func findRepostImpl(client *VKClient, ownerID UserID) s.Stream[Post] {
	return getPaged[Post](&findRepostPager{
		client: client,
		offset: 0,
		total:  f.None[uint](),
		params: MakeUrlValues(map[string]any{
			"owner_id": ownerID,
			"count":    wallGetPageSize,
		}),
	})
}

// Returns either found (or not found) repost's post id
func findRepost(client *VKClient, userID UserID, postID PostID, postDate uint) f.Option[uint] {
	allPosts := findRepostImpl(client, userID)
	pinnedPostMaybe := s.Head(allPosts)
	if pinnedPostMaybe.IsNone() {
		return f.None[uint]()
	}
	pinnedPost := pinnedPostMaybe.Unwrap()
	isRepostPredicate := func(post Post) bool {
		copyHistory := post.CopyHistory
		return len(copyHistory) != 0 && copyHistory[0].ID == postID.ID && copyHistory[0].OwnerID == postID.OwnerID
	}
	if isRepostPredicate(pinnedPost) {
		return f.Some(pinnedPost.ID)
	}
	remainingPosts := s.TakeWhile(
		allPosts,
		func(w Post) bool {
			return w.Date >= postDate
		},
	)
	repostMaybe := s.Find(remainingPosts, isRepostPredicate)
	return f.Map(repostMaybe, func(w Post) uint { return w.ID })
}

func userInfoToUserID(info User) UserID {
	return info.ID
}

func getPotentialUserIDs(client *VKClient, postID PostID) s.Stream[UserID] {
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

	return s.Chain(commenters, likers, potentialUserIDs)
}

func getCheckedIDs(client *VKClient, postID PostID, postDate uint, userIDs s.Stream[UserID]) s.Stream[PostID] {
	tasks := s.MapFilter(
		userIDs,
		func(userID UserID) f.Option[f.Task[PostID]] {
			return f.Map(
				findRepost(client, userID, postID, postDate),
				f.ToTaskFactory(func(postID uint) PostID {
					return PostID{userID, postID}
				}),
			)
		},
	)
	return s.Parallel(userCheckRepostsThreads, tasks)
}

func GetReposters(client *VKClient, postID PostID) r.Result[s.Stream[PostID]] {
	return r.Map(
		client.getPostTime(postID),
		func(postDate uint) s.Stream[PostID] {
			uniqueIDs := s.Unique(getPotentialUserIDs(client, postID))
			return getCheckedIDs(client, postID, postDate, uniqueIDs)
		},
	)
}
