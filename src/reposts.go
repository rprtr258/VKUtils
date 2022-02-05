package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
)

type Repost struct {
	Response struct {
		Count int `json:"count"` // TODO: remove
		Items []struct {
			Date        uint `json:"date"`
			PostID      uint `json:"id"`
			CopyHistory []struct {
				PostID  uint   `json:"id"`
				OwnerID UserID `json:"owner_id"`
			} `json:"copy_history"`
		} `json:"items"`
	} `json:"response"`
}

const (
	NOT_FOUND_REPOST           = -1 // scanned all posts, not found repost
	wallGet_count              = 100
	wallGet_countString        = "100"
	USER_CHECK_REPOSTS_THREADS = 10
)

func (client *VKClient) getPosts(ownerIDString string, offset uint, countString string) (v Repost, err error) {
	body, err := client.apiRequest("wall.get", url.Values{
		"owner_id": []string{ownerIDString},
		"offset":   []string{fmt.Sprint(offset)},
		"count":    []string{countString},
	})
	if err != nil {
		return
	}
	err = json.Unmarshal(body, &v)
	return
}

func findRepost(client *VKClient, userID UserID, post Post) (repostID int, err error) {
	ownerIDString := fmt.Sprint(userID)
	v, err := client.getPosts(ownerIDString, 0, wallGet_countString)
	if err != nil {
		errMsg := err.Error()
		// TODO: change to error structs?
		if errMsg == "Error(15) Access denied: user hid his wall from accessing from outside" ||
			errMsg == "Error(18) User was deleted or banned" ||
			errMsg == "Error(30) This profile is private" {
			return NOT_FOUND_REPOST, nil
		}
		return
	}
	postsCount := v.Response.Count
	if postsCount == 0 {
		repostID = NOT_FOUND_REPOST
		return
	}
	// check first/pinned post and first 99
	for i, item := range v.Response.Items {
		// first post is unchecked regardless if it is pinned or not
		if i > 0 && item.Date < post.Date {
			repostID = NOT_FOUND_REPOST
			return
		}
		copyHistory := item.CopyHistory
		if len(copyHistory) != 0 && copyHistory[0].PostID == post.ID && copyHistory[0].OwnerID == post.Owner {
			repostID = int(item.PostID)
			return
		}
	}
	// check remaining posts
	var offset uint = wallGet_count
	for offset < uint(postsCount) {
		v, err = client.getPosts(ownerIDString, offset, wallGet_countString)
		if err != nil {
			return
		}
		for _, item := range v.Response.Items {
			if item.Date < post.Date {
				repostID = NOT_FOUND_REPOST
				return
			}
			copyHistory := item.CopyHistory
			if len(copyHistory) != 0 && copyHistory[0].PostID == post.ID && copyHistory[0].OwnerID == post.Owner {
				repostID = int(item.PostID)
				return
			}
		}
		offset += wallGet_count
	}
	repostID = NOT_FOUND_REPOST
	return
}

func getUniqueIDs(client *VKClient, ownerID UserID, postID uint) (<-chan UserID, <-chan error) {
	var wg sync.WaitGroup
	users := make(chan UserID)
	errors := make(chan error)

	// TODO: add commenters?
	wg.Add(1)
	go func() {
		likers, errorsLikers := client.getLikes(ownerID, postID)
		go drainErrorChan(errors, errorsLikers)
		for userID := range likers {
			users <- userID
		}
		wg.Done()
	}()

	// TODO: "Error(15) Access denied: group hide members"
	wg.Add(1)
	go func() {
		var potentialUserIDs <-chan UserID
		var errorsUsers <-chan error
		if ownerID < 0 { // owner is group
			potentialUserIDs, errorsUsers = client.getGroupMembers(ownerID)
		} else { // owner is user
			potentialUserIDs, errorsUsers = client.getFriends(UserID(ownerID))
		}
		go drainErrorChan(errors, errorsUsers)
		for userID := range potentialUserIDs {
			users <- userID
		}
		wg.Done()
	}()

	go func() {
		wg.Wait()
		close(users)
		close(errors)
	}()

	wasChecked := make(map[UserID]bool)
	toCheckQueue := make(chan UserID)
	go func() {
		for userID := range users {
			if !wasChecked[userID] {
				toCheckQueue <- userID
				wasChecked[userID] = true
			}
		}
		close(toCheckQueue)
	}()
	return toCheckQueue, errors
}

type Sharer struct {
	UserID   UserID `json:"user_id"`
	RepostID int    `json:"repost_id"`
}

func getCheckedIDs(client *VKClient, post Post, userIDs <-chan UserID) (<-chan Sharer, <-chan error) {
	sharers := make(chan Sharer)
	errors := make(chan error)
	var wg sync.WaitGroup
	wg.Add(USER_CHECK_REPOSTS_THREADS)
	for i := 0; i < USER_CHECK_REPOSTS_THREADS; i++ {
		go func() {
			for userID := range userIDs {
				repostID, err := findRepost(client, userID, post)
				if err != nil {
					errors <- err
					continue
				}
				if repostID != NOT_FOUND_REPOST {
					sharers <- Sharer{userID, repostID}
					// TODO: consider if len(res.Reposters) == res.TotalReposts { break } // which is highly unlikely
				}
			}
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		close(sharers)
		close(errors)
	}()
	return sharers, errors
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

func getRepostersByPostUrl(client *VKClient, postUrl string) RepostersResult {
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
