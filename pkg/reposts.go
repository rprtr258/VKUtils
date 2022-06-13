package vkutils

import (
	"fmt"
	"net/url"

	f "github.com/rprtr258/goflow/fun"
	r "github.com/rprtr258/goflow/result"
	s "github.com/rprtr258/goflow/stream"
)

type PageSize uint

const (
	wallGetPageSize = PageSize(100)
	// wallGetPageSizeString = "100" // TODO: fmt.Sprint(wallGetPageSize) (init?)
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
	// log.Println("GET USER LIST", xs.offset, xs.total)
	wallPosts := pager.client.getWallPosts(pager.params, "offset", fmt.Sprint(pager.offset))
	// onePageOfPosts = TryRecover(onePageOfPosts, func(err error) IO[WallPosts] {
	// 	errMsg := err.Error()
	// 	// TODO: change to error structs
	// 	if errMsg == "Error(15) Access denied: user hid his wall from accessing from outside" ||
	// 		errMsg == "Error(18) User was deleted or banned" ||
	// 		errMsg == "Error(30) This profile is private" {
	// 		return []Post{}
	// 	}
	// 	return Fail[Repost](err)
	// })
	return r.Map(
		wallPosts,
		func(ul WallPosts) f.Option[[]Post] {
			pager.offset += uint(wallGetPageSize)
			pager.total = f.Some(ul.Response.Count)
			return f.Some(ul.Response.Items)
		},
	)
}

// TODO: limit extracted fields.
func findRepostImpl(client *VKClient, ownerID UserID) s.Stream[Post] {
	return getPaged[Post](&findRepostPager{
		client: client,
		offset: 0,
		total:  f.None[uint](),
		params: url.Values{
			"owner_id": []string{fmt.Sprint(ownerID)},
			"count":    []string{fmt.Sprint(wallGetPageSize)},
		},
	})
}

// Returns either found (or not found) repost's post id
func findRepost(client *VKClient, userID UserID, origPost Post) f.Option[uint] {
	allPosts := findRepostImpl(client, userID)
	pinnedPostMaybe := s.Head(allPosts)
	if pinnedPostMaybe.IsNone() {
		return f.None[uint]()
	}
	pinnedPost := pinnedPostMaybe.Unwrap()
	isRepostPredicate := func(post Post) bool {
		copyHistory := post.CopyHistory
		return len(copyHistory) != 0 && copyHistory[0].PostID == origPost.ID && copyHistory[0].OwnerID == origPost.Owner
	}
	if isRepostPredicate(pinnedPost) {
		return f.Some(pinnedPost.ID)
	}
	remainingPosts := s.TakeWhile(
		allPosts,
		func(w Post) bool {
			return w.Date >= origPost.Date
		},
	)
	repostMaybe := s.Find(remainingPosts, isRepostPredicate)
	return f.Map(repostMaybe, func(w Post) uint { return w.ID })
}

func userInfoToUserID(info User) UserID {
	return info.ID
}

func getPotentialUserIDs(client *VKClient, ownerID UserID, postID uint) s.Stream[UserID] {
	// TODO: add commenters?

	// scan likers
	likers := s.Map(client.getLikes(ownerID, postID), userInfoToUserID)

	// TODO: "Error(15) Access denied: group hide members"
	// scan group members/friends of post owner
	var potentialUserIDs s.Stream[UserID]
	if ownerID < 0 { // owner is group
		potentialUserIDs = s.Map(client.getGroupMembers(ownerID), userInfoToUserID)
	} else { // owner is user
		potentialUserIDs = s.Map(client.getFriends(ownerID), userInfoToUserID)
	}

	return s.Chain(likers, potentialUserIDs)
}

// Sharer is post shared user.
type Sharer struct {
	UserID   UserID
	RepostID uint // TODO: uint?
}

func getCheckedIDs(client *VKClient, post Post, userIDs s.Stream[UserID]) s.Stream[Sharer] {
	tasks := s.MapFilter(
		userIDs,
		func(userID UserID) f.Option[f.Task[Sharer]] {
			return f.Map(
				findRepost(client, userID, post),
				f.ToTaskFactory(func(postID uint) Sharer {
					return Sharer{userID, postID}
				}),
			)
		},
	)
	return s.Parallel(userCheckRepostsThreads, tasks)
}

func GetReposters(client *VKClient, ownerID UserID, postID uint) r.Result[s.Stream[Sharer]] {
	// TODO: separate modification of post and creation of result
	return r.Map(
		client.getPostTime(ownerID, postID),
		func(postDate uint) s.Stream[Sharer] {
			uniqueIDs := s.Unique(getPotentialUserIDs(client, ownerID, postID))
			return getCheckedIDs(client, Post{
				Owner: ownerID,
				ID:    postID,
				Date:  postDate,
			}, uniqueIDs)
		},
	)
}
