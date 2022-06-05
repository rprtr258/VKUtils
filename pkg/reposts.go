package vkutils

import (
	"fmt"
	"log"
	"net/url"

	f "github.com/rprtr258/vk-utils/flow/fun"
	i "github.com/rprtr258/vk-utils/flow/io"
	s "github.com/rprtr258/vk-utils/flow/stream"
)

type WallPost struct {
	Date        uint `json:"date"`
	PostID      uint `json:"id"`
	CopyHistory []struct {
		PostID  uint   `json:"id"`
		OwnerID UserID `json:"owner_id"`
	} `json:"copy_history"`
}

// TODO: replace with f.Pair[uint, s.Stream[WallPost]]
type WallPosts struct {
	Response struct {
		Count uint       `json:"count"`
		Items []WallPost `json:"items"`
	} `json:"response"`
}

const (
	wallGet_count              = 100
	wallGet_countString        = "100" // TODO: fmt.Sprint(wallGet_count) (init?)
	USER_CHECK_REPOSTS_THREADS = 10
)

type findRepostImplImpl struct {
	client        *VKClient
	ownerIDString string
	offset        uint
	total         f.Option[uint]
	countString   string // TODO: remove to urlValues?
	count         uint

	curPage s.Stream[WallPost]
}

// TODO: abstract findRepostImplImpl and getUserListImpl cuz they have similar structure and logic
// TODO: limit extracted fields
func (xs *findRepostImplImpl) Next() f.Option[WallPost] {
	// TODO: move out to flatten
	if x := xs.curPage.Next(); x.IsSome() {
		return x
	}

	// log.Println("REPOST CHECK", xs.offset)
	if /*xs.offset == 0*/ xs.total.IsNone() || xs.offset < xs.total.Unwrap() {
		body := apiRequest(xs.client, "wall.get", url.Values{
			"owner_id": []string{xs.ownerIDString},
			"offset":   []string{fmt.Sprint(xs.offset)},
			"count":    []string{xs.countString},
		})
		onePageOfPosts := i.FlatMap(body, jsonUnmarshall[WallPosts])
		// onePageOfPosts = Recover(onePageOfPosts, func(err error) IO[WallPosts] {
		// 	errMsg := err.Error()
		// 	// TODO: change to error structs?
		// 	if errMsg == "Error(15) Access denied: user hid his wall from accessing from outside" ||
		// 		errMsg == "Error(18) User was deleted or banned" ||
		// 		errMsg == "Error(30) This profile is private" {
		// 		return Lift[Repost](NOT_FOUND_REPOST)
		// 	}
		// 	return Fail[Repost](err)
		// })
		return i.Fold(
			onePageOfPosts,
			func(totalAndFirstBatch WallPosts) f.Option[WallPost] {
				xs.total, xs.curPage = f.Some(totalAndFirstBatch.Response.Count), s.FromSlice(totalAndFirstBatch.Response.Items)
				// if xs.offset == 0 {
				// 	log.Printf("CHECKING USER %s WITH %d POSTS\n", xs.ownerIDString, totalAndFirstBatch.Response.Count)
				// }
				xs.offset += xs.count
				return xs.curPage.Next()
			},
			func(err error) f.Option[WallPost] {
				log.Println("ERROR WHILE GETTING PAGE: ", err)
				return f.None[WallPost]()
			},
		)
	} else {
		return f.None[WallPost]()
	}
}

func findRepostImpl(client *VKClient, ownerIDString string) s.Stream[WallPost] {
	return &findRepostImplImpl{
		client:        client,
		ownerIDString: ownerIDString,
		offset:        0,
		total:         f.None[uint](),
		count:         wallGet_count,
		countString:   fmt.Sprint(wallGet_count),
		curPage:       s.NewStreamEmpty[WallPost](),
	}
}

// Returns either found (or not found) repost's post id
// TODO: binary search?
func findRepost(client *VKClient, userID UserID, origPost Post) f.Option[uint] {
	ownerIDString := fmt.Sprint(userID)
	allPosts := findRepostImpl(client, ownerIDString)
	pinnedPostMaybe := s.Head(allPosts)
	if pinnedPostMaybe.IsNone() {
		return f.None[uint]()
	}
	pinnedPost := pinnedPostMaybe.Unwrap()
	isRepostPredicate := func(post WallPost) bool {
		copyHistory := post.CopyHistory
		return len(copyHistory) != 0 && copyHistory[0].PostID == origPost.ID && copyHistory[0].OwnerID == origPost.Owner
	}
	if isRepostPredicate(pinnedPost) {
		return f.Some(pinnedPost.PostID)
	}
	remainingPosts := s.TakeWhile(
		allPosts,
		func(w WallPost) bool {
			return w.Date >= origPost.Date
		},
	)
	repostMaybe := s.Find(remainingPosts, isRepostPredicate)
	return f.Map(repostMaybe, func(w WallPost) uint { return w.PostID })
}

func getUniqueIDs(client *VKClient, ownerID UserID, postID uint) s.Stream[UserID] {
	// TODO: add commenters?

	// scan likers
	likers := client.getLikes(ownerID, postID)

	// TODO: "Error(15) Access denied: group hide members"
	// scan group members/friends of post owner
	var potentialUserIDs s.Stream[UserID]
	if ownerID < 0 { // owner is group
		potentialUserIDs = client.getGroupMembers(ownerID)
	} else { // owner is user
		potentialUserIDs = client.getFriends(UserID(ownerID))
	}

	return s.Unique(s.Chain(likers, potentialUserIDs))
}

// TODO: does it need to have json tags?
type Sharer struct {
	UserID   UserID `json:"user_id"`
	RepostID int    `json:"repost_id"` // TODO: uint?
}

// TODO: remove NOT_FOUND_REPOST const
// TODO: consider if len(res.Reposters) == res.TotalReposts { break } // which is highly unlikely
// TODO: is there a simpler way to transform Stream[IO[A]] to IO[Stream[A]] (which in turn is Stream[A])?
func getCheckedIDs(client *VKClient, post Post, userIDs s.Stream[UserID]) s.Stream[Sharer] {
	// TODO: change to mapfilter
	a := s.NewPool[f.Pair[UserID, f.Option[uint]]](10)(s.Map(
		userIDs,
		func(userID UserID) func() f.Pair[UserID, f.Option[uint]] {
			return func() f.Pair[UserID, f.Option[uint]] {
				return f.NewPair(userID, findRepost(client, userID, post))
			}
		},
	))
	vv := s.Filter(
		a,
		func(x f.Pair[UserID, f.Option[uint]]) bool {
			return x.Right.IsSome()
		},
	)
	return s.Map(
		vv,
		func(leftSurely f.Pair[UserID, f.Option[uint]]) Sharer {
			return Sharer{
				UserID:   leftSurely.Left,
				RepostID: int(leftSurely.Right.Unwrap()),
			}
		},
	)
}

func getSharersAndReposts(client *VKClient, ownerId UserID, postId uint) s.Stream[Sharer] {
	// TODO: expand to two vars/change to simpler structure
	post := Post{
		Owner: ownerId,
		ID:    postId,
	}
	// TODO: separate modification of post and creation of result
	i.Map(
		client.getPostTime(post), // TODO: signature without struct
		func(postDate uint) Post {
			return Post{
				Owner: ownerId,
				ID:    postId,
				Date:  postDate,
			}
		},
	)
	uniqueIDs := getUniqueIDs(client, ownerId, postId)
	checkedIDs := getCheckedIDs(client, post, uniqueIDs)
	return checkedIDs
}

// func getSharers(client *VKClient, ownerId UserID, postId uint) s.Stream[UserID] {
// 	reposts := getSharersAndReposts(client, ownerId, postId)
// 	return s.Map(
// 		reposts,
// 		func(h Sharer) UserID {
// 			return h.UserID
// 		},
// 	)
// }

func parsePostUrl(url string) (ownerId UserID, postId uint) {
	fmt.Sscanf(url, "https://vk.com/wall%d_%d", &ownerId, &postId)
	return
}

func GetRepostersByPostUrl(client *VKClient, postUrl string) s.Stream[Sharer] {
	ownerId, postId := parsePostUrl(postUrl)
	return getSharersAndReposts(client, ownerId, postId)
}
