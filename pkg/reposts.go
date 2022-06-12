package vkutils

import (
	"log"

	f "github.com/rprtr258/goflow/fun"
	r "github.com/rprtr258/goflow/result"
	s "github.com/rprtr258/goflow/stream"
)

const (
	wallGetCount       = 100
	wallGetCountString = "100" // TODO: fmt.Sprint(wallGet_count) (init?)
	// userCheckRepostsThreads = 10
)

type findRepostImplImpl struct {
	client  *VKClient
	ownerID UserID
	offset  uint
	total   f.Option[uint]
	count   uint

	curPage s.Stream[Post]
}

// TODO: abstract findRepostImplImpl and getUserListImpl cuz they have similar structure and logic.
// TODO: limit extracted fields.
func (xs *findRepostImplImpl) Next() f.Option[Post] {
	// TODO: move out to flatten
	if x := xs.curPage.Next(); x.IsSome() {
		return x
	}

	// log.Println("REPOST CHECK", xs.offset)
	if /*xs.offset > 0*/ xs.total.IsSome() && xs.offset >= xs.total.Unwrap() {
		return f.None[Post]()
	}
	// body := apiRequest(xs.client, "wall.get", url.Values{
	// 	"owner_id": []string{xs.ownerIDString},
	// 	"offset":   []string{fmt.Sprint(xs.offset)},
	// 	"count":    []string{xs.countString},
	// })
	onePageOfPosts := xs.client.getWallPosts(xs.offset, xs.count, xs.ownerID)
	// onePageOfPosts = TryRecover(onePageOfPosts, func(err error) IO[WallPosts] {
	// 	errMsg := err.Error()
	// 	// TODO: change to error structs?
	// 	if errMsg == "Error(15) Access denied: user hid his wall from accessing from outside" ||
	// 		errMsg == "Error(18) User was deleted or banned" ||
	// 		errMsg == "Error(30) This profile is private" {
	// 		return Lift[Repost](NOT_FOUND_REPOST)
	// 	}
	// 	return Fail[Repost](err)
	// })
	return r.Fold(
		onePageOfPosts,
		func(totalAndFirstBatch WallPosts) f.Option[Post] {
			xs.total, xs.curPage = f.Some(totalAndFirstBatch.Response.Count), s.FromSlice(totalAndFirstBatch.Response.Items)
			// if xs.offset == 0 {
			// 	log.Printf("CHECKING USER %s WITH %d POSTS\n", xs.ownerIDString, totalAndFirstBatch.Response.Count)
			// }
			xs.offset += xs.count
			return xs.curPage.Next()
		},
		func(err error) f.Option[Post] {
			log.Println("ERROR WHILE GETTING PAGE: ", err)
			return f.None[Post]()
		},
	)
}

func findRepostImpl(client *VKClient, ownerID UserID) s.Stream[Post] {
	return &findRepostImplImpl{
		client:  client,
		ownerID: ownerID,
		offset:  0,
		total:   f.None[uint](),
		count:   wallGetCount,
		curPage: s.NewStreamEmpty[Post](),
	}
}

// Returns either found (or not found) repost's post id
// TODO: binary search?
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

func userInfoToUserID(info UserInfo) UserID {
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

// TODO: remove NOT_FOUND_REPOST const
// TODO: consider if len(res.Reposters) == res.TotalReposts { break } // which is highly unlikely
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
	return s.Parallel(10, tasks)
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
