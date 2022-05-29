package vkutils

import (
	"fmt"
	"net/url"
	"sync"

	f "github.com/primetalk/goio/fun"
	i "github.com/primetalk/goio/io"
	s "github.com/primetalk/goio/stream"
)

type WallPost struct {
	Date        uint `json:"date"`
	PostID      uint `json:"id"`
	CopyHistory []struct {
		PostID  uint   `json:"id"`
		OwnerID UserID `json:"owner_id"`
	} `json:"copy_history"`
}

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

func (client *VKClient) getPosts(ownerIDString string, offset uint, countString string) i.IO[WallPosts] {
	body := apiRequest(client, "wall.get", url.Values{
		"owner_id": []string{ownerIDString},
		"offset":   []string{fmt.Sprint(offset)},
		"count":    []string{countString},
	})
	v := i.FlatMap(body, jsonUnmarshall[WallPosts])
	// v = Recover(v, func(err error) IO[WallPosts] {
	// 	errMsg := err.Error()
	// 	// TODO: change to error structs?
	// 	if errMsg == "Error(15) Access denied: user hid his wall from accessing from outside" ||
	// 		errMsg == "Error(18) User was deleted or banned" ||
	// 		errMsg == "Error(30) This profile is private" {
	// 		return Lift[Repost](NOT_FOUND_REPOST)
	// 	}
	// 	return Fail[Repost](err)
	// })
	return v
}

// TODO: abstract findRepostImplImpl and getUserListImpl cuz they have similar structure and logic
func findRepostImplImpl(client *VKClient, ownerIDString string, offset uint, total uint /*TODO: remove*/) s.Stream[s.Stream[WallPost]] {
	var stepResultIo i.IO[s.StepResult[s.Stream[WallPost]]]
	newOffset := offset + wallGet_count
	if offset == 0 {
		stepResultIo = i.Map[WallPosts, s.StepResult[s.Stream[WallPost]]](
			client.getPosts(ownerIDString, offset, wallGet_countString),
			func(ws WallPosts) s.StepResult[s.Stream[WallPost]] {
				return s.NewStepResult(
					s.FromSlice(ws.Response.Items),
					findRepostImplImpl(client, ownerIDString, newOffset, ws.Response.Count),
				)
			},
		)
	} else if offset >= total {
		stepResultIo = i.Lift(s.NewStepResultFinished[s.Stream[WallPost]]())
	} else {
		stepResultIo = i.Map[WallPosts, s.StepResult[s.Stream[WallPost]]](
			client.getPosts(ownerIDString, newOffset, wallGet_countString),
			func(ws WallPosts) s.StepResult[s.Stream[WallPost]] {
				return s.NewStepResult[s.Stream[WallPost]](
					s.FromSlice(ws.Response.Items),
					findRepostImplImpl(client, ownerIDString, newOffset, ws.Response.Count),
				)
			},
		)
	}
	return s.FromStepResult(stepResultIo)
}

func findRepostImpl(client *VKClient, ownerIDString string) s.Stream[WallPost] {
	vv := findRepostImplImpl(client, ownerIDString, 0, 0)
	return s.Flatten[WallPost](vv)
}

func TakeWhile[A any](sa s.Stream[A], p func(A) bool) s.Stream[A] {
	return s.FromStepResult(i.Map[s.StepResult[A], s.StepResult[A]](
		sa,
		func(a s.StepResult[A]) s.StepResult[A] {
			if p(a.Value) {
				cont := TakeWhile(a.Continuation, p)
				return s.NewStepResult[A](a.Value, cont)
			} else {
				return s.NewStepResultFinished[A]()
			}
		},
	))
}

func isPostRepost(post WallPost, origPost Post) bool {
	copyHistory := post.CopyHistory
	return len(copyHistory) != 0 && copyHistory[0].PostID == origPost.ID && copyHistory[0].OwnerID == origPost.Owner
}

// Returns either found repost or unit signalling that it was not found
// TODO: binary search?
func findRepost(client *VKClient, userID UserID, post Post) i.IO[f.Either[uint, f.Unit]] {
	ownerIDString := fmt.Sprint(userID)
	swp := findRepostImpl(client, ownerIDString)
	pinnedPost := s.Head(swp)
	return i.FlatMap(
		pinnedPost,
		func(w WallPost) i.IO[f.Either[uint, f.Unit]] {
			if isPostRepost(w, post) {
				return i.Lift(f.Left[uint, f.Unit](w.PostID))
			} else {
				ss := TakeWhile(
					swp,
					func(w WallPost) bool {
						return w.Date >= post.Date
					},
				)
				ss = s.Filter(ss, func(ww WallPost) bool {
					return isPostRepost(ww, post)
				})
				return i.Fold(
					s.Head(ss),
					func(www WallPost) i.IO[f.Either[uint, f.Unit]] {
						return i.Lift(f.Left[uint, f.Unit](www.PostID))
					},
					func(e error) i.IO[f.Either[uint, f.Unit]] {
						return i.Lift(f.Right[uint, f.Unit](f.Unit1))
					},
				)
			}
		},
	)
}

func getUniqueIDs(client *VKClient, ownerID UserID, postID uint) s.Stream[UserID] {
	wasChecked := make(map[UserID]struct{})
	toCheckQueue := make(chan UserID)

	// TODO: add commenters?
	// scan likers
	likersIo := s.Collect(
		client.getLikes(ownerID, postID),
		func(userID UserID) error {
			if _, has := wasChecked[userID]; !has {
				wasChecked[userID] = struct{}{}
			}
			return nil
		},
	)

	// TODO: "Error(15) Access denied: group hide members"
	// scan group members/friends of post owner
	var potentialUserIDs s.Stream[UserID]
	if ownerID < 0 { // owner is group
		potentialUserIDs = client.getGroupMembers(ownerID)
	} else { // owner is user
		potentialUserIDs = client.getFriends(UserID(ownerID))
	}
	potentialUsersIo := s.Collect(
		potentialUserIDs,
		func(userID UserID) error {
			if _, has := wasChecked[userID]; !has {
				wasChecked[userID] = struct{}{}
			}
			return nil
		},
	)

	return i.FlatMap[f.Unit, s.StepResult[UserID]](
		likersIo,
		func(_ f.Unit) i.IO[s.StepResult[UserID]] {
			return i.FlatMap[f.Unit, s.StepResult[UserID]](
				potentialUsersIo,
				func(_ f.Unit) i.IO[s.StepResult[UserID]] {
					slice := make([]UserID, 0, len(wasChecked))
					for k, _ := range wasChecked {
						slice = append(slice, k)
					}
					return s.FromSlice(slice)
				},
			)
		},
	)
}

// TODO: does it need to have json tags?
type Sharer struct {
	UserID   UserID `json:"user_id"`
	RepostID int    `json:"repost_id"` // TODO: uint?
}

// TODO: remove NOT_FOUND_REPOST const
// TODO: consider if len(res.Reposters) == res.TotalReposts { break } // which is highly unlikely
func getCheckedIDs(client *VKClient, post Post, userIDs s.Stream[UserID]) s.Stream[Sharer] {
	a := s.Map(
		userIDs,
		func(userID UserID) i.IO[f.Pair[UserID, f.Either[uint, f.Unit]]] {
			return i.Map(
				findRepost(client, userID, post),
				func(x f.Either[uint, f.Unit]) f.Pair[UserID, f.Either[uint, f.Unit]] {
					return f.NewPair(userID, x)
				},
			)
		},
	)
	slice := make([]i.IO[f.Pair[UserID, f.Either[uint, f.Unit]]], 0)
	ioSliceIoEitherUintUnit := i.FlatMap(
		s.AppendToSlice(a, slice),
		i.Sequence[f.Pair[UserID, f.Either[uint, f.Unit]]],
	)
	v := s.FromStepResult(
		i.FlatMap(
			ioSliceIoEitherUintUnit,
			func(us []f.Pair[UserID, f.Either[uint, f.Unit]]) i.IO[s.StepResult[f.Pair[UserID, f.Either[uint, f.Unit]]]] {
				return s.FromSlice(us)
			},
		),
	)
	vv := s.Filter(
		v,
		func(x f.Pair[UserID, f.Either[uint, f.Unit]]) bool {
			return f.IsLeft(x.V2)
		},
	)
	return s.Map(
		vv,
		func(leftSurely f.Pair[UserID, f.Either[uint, f.Unit]]) Sharer {
			return Sharer{leftSurely.V1, int(leftSurely.V2.Left)}
		},
	)
}

func drainErrorChan(dest chan error, source <-chan error) {
	for err := range source {
		dest <- err
	}
}

func getSharersAndReposts(client *VKClient, ownerId UserID, postId uint) (<-chan Sharer, <-chan error) {
	// TODO: expand to two vars/change to simpler structure
	post := Post{
		Owner: ownerId,
		ID:    postId,
	}
	// TODO: separate modification of post and creation of result
	postDate, err := client.getPostTime(post)
	if err != nil {
		shares := make(chan Sharer)
		close(shares)
		errors := make(chan error)
		go func(err error) {
			errors <- err
			close(errors)
		}(err)
		return shares, errors
	}
	post.Date = postDate
	errors := make(chan error)
	uniqueIDs, errorsUnique := getUniqueIDs(client, ownerId, postId)
	checkedIDs, errorsChecking := getCheckedIDs(client, post, uniqueIDs)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		drainErrorChan(errors, errorsUnique)
		wg.Done()
	}()
	go func() {
		drainErrorChan(errors, errorsChecking)
		wg.Done()
	}()
	go func() {
		wg.Wait()
		close(errors)
	}()
	return checkedIDs, errors
}

func getSharers(client *VKClient, ownerId UserID, postId uint) (users <-chan UserID, errors <-chan error) {
	usersChan := make(chan UserID)
	reposts, errors := getSharersAndReposts(client, ownerId, postId)
	for repost := range reposts {
		usersChan <- repost.UserID
	}
	users = usersChan
	return
}

type RepostersResult struct {
	Reposts []Sharer `json:"reposts"`
	Errs    []string `json:"errors"`
}

func GetRepostersByPostUrl(client *VKClient, postUrl string) RepostersResult {
	var ownerId UserID
	var postId uint
	fmt.Sscanf(postUrl, "https://vk.com/wall%d_%d", &ownerId, &postId)
	var res RepostersResult
	res.Reposts = make([]Sharer, 0)
	res.Errs = make([]string, 0)
	sharers, errors := getSharersAndReposts(client, ownerId, postId)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		for share := range sharers {
			res.Reposts = append(res.Reposts, share)
		}
		wg.Done()
	}()
	go func() {
		for err := range errors {
			res.Errs = append(res.Errs, err.Error())
		}
		wg.Done()
	}()
	wg.Wait()
	return res
}
