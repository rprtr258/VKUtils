package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
)

func drainErrorChan(dest chan error, source <-chan error) {
	for err := range source {
		dest <- err
	}
}

func (client *VKClient) getTotalUsers(method string, params url.Values) (total uint, err error) {
	var v UserList
	params.Set("offset", "0")
	params.Set("count", "1")
	body, err := client.apiRequest(method, params)
	if err != nil {
		return
	}
	err = json.Unmarshal(body, &v)
	if err != nil {
		return
	}
	total = v.Response.Count
	return
}

func (client *VKClient) getUserList(method string, params url.Values, count uint) (<-chan UserID, <-chan error) {
	users := make(chan UserID)
	errors := make(chan error)
	go func() {
		defer func() {
			close(users)
		}()
		total, err := client.getTotalUsers(method, params) // TODO: remove
		if err != nil {
			go func(err error) {
				errors <- err
				close(errors)
			}(err)
			return
		}
		countString := fmt.Sprint(count)
		var wg sync.WaitGroup
		const THREADS = 10
		wg.Add(THREADS)
		STEP := count * THREADS
		for i := uint(0); i < THREADS; i++ {
			go func(start uint) {
				urlParams := make(url.Values)
				for k, v := range params {
					urlParams[k] = v
				}
				offset := start
				for offset < total {
					var v UserList
					urlParams.Set("offset", fmt.Sprint(offset))
					urlParams.Set("count", countString)
					body, err := client.apiRequest(method, urlParams)
					if err != nil {
						go func(err error) {
							errors <- err
							close(errors)
						}(err)
						return
					}
					err = json.Unmarshal(body, &v)
					if err != nil {
						go func(err error) {
							errors <- err
							close(errors)
						}(err)
						return
					} else {
						for _, userID := range v.Response.Items {
							users <- userID
						}
					}
					offset += STEP
				}
				wg.Done()
			}(count * i)
		}
		wg.Wait()
		close(errors)
	}()
	return users, errors
}

// TODO: move client to args
func (client *VKClient) getGroupMembers(groupID UserID) (<-chan UserID, <-chan error) {
	return client.getUserList("groups.getMembers", url.Values{
		"group_id": []string{fmt.Sprint(-groupID)},
	}, 1000)
}

func (client *VKClient) getFriends(userID UserID) (<-chan UserID, <-chan error) {
	return client.getUserList("friends.get", url.Values{
		"user_id": []string{fmt.Sprint(userID)},
	}, 5000)
}

func (client *VKClient) getLikes(ownerId UserID, postId uint) (<-chan UserID, <-chan error) {
	return client.getUserList("likes.getList", url.Values{
		"type":     []string{"post"},
		"owner_id": []string{fmt.Sprint(ownerId)},
		"item_id":  []string{fmt.Sprint(postId)},
		"skip_own": []string{"0"},
	}, 1000)
}

func (client *VKClient) getPostTime(post Post) (time uint, err error) {
	body, err := client.apiRequest("wall.getById", url.Values{
		"posts": []string{fmt.Sprintf("%d_%d", post.Owner, post.ID)},
	})
	if err != nil {
		return
	}
	var v struct {
		Response []struct {
			Date    uint `json:"date"`
			Reposts struct {
				Count int `json:"count"`
			} `json:"reposts"`
		} `json:"response"`
	}
	err = json.Unmarshal(body, &v)
	if err != nil {
		return
	}
	if len(v.Response) != 1 {
		err = fmt.Errorf("Post %d_%d is hidden", post.Owner, post.ID)
		return
	}
	time = v.Response[0].Date
	return
}

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

// TODO: check post id can't be negative or 0, if 0 change to uint
const (
	NOT_FOUND = -1 // scanned all posts, not found repost
)

func findRepost(client *VKClient, userID UserID, post Post) (repostID int, err error) {
	ownerIDString := fmt.Sprint(userID)
	var v Repost
	const COUNT uint = 100
	countString := fmt.Sprint(COUNT)
	body, err := client.apiRequest("wall.get", url.Values{
		"owner_id": []string{ownerIDString},
		"offset":   []string{"0"},
		"count":    []string{countString},
	})
	if err != nil {
		errMsg := err.Error()
		// TODO: change to error structs?
		if errMsg == "Error(15) Access denied: user hid his wall from accessing from outside" ||
			errMsg == "Error(18) User was deleted or banned" ||
			errMsg == "Error(30) This profile is private" {
			return NOT_FOUND, nil
		}
		return
	}
	err = json.Unmarshal(body, &v)
	if err != nil {
		return
	}
	postsCount := v.Response.Count
	if postsCount == 0 {
		repostID = NOT_FOUND
		return
	}
	// check first/pinned post and first 99
	for i, item := range v.Response.Items {
		// first post is unchecked regardless if it is pinned or not
		if i > 0 && item.Date < post.Date {
			repostID = NOT_FOUND
			return
		}
		copyHistory := item.CopyHistory
		if len(copyHistory) != 0 && copyHistory[0].PostID == post.ID && copyHistory[0].OwnerID == post.Owner {
			repostID = int(item.PostID)
			return
		}
	}
	// check remaining posts
	var offset uint = COUNT
	for offset < uint(postsCount) {
		body, err = client.apiRequest("wall.get", url.Values{
			"owner_id": []string{ownerIDString},
			"offset":   []string{fmt.Sprint(offset)},
			"count":    []string{countString},
		})
		if err != nil {
			return
		}
		err = json.Unmarshal(body, &v)
		if err != nil {
			return
		}
		for _, item := range v.Response.Items {
			if item.Date < post.Date {
				repostID = NOT_FOUND
				return
			}
			copyHistory := item.CopyHistory
			if len(copyHistory) != 0 && copyHistory[0].PostID == post.ID && copyHistory[0].OwnerID == post.Owner {
				repostID = int(item.PostID)
				return
			}
		}
		offset += COUNT
	}
	repostID = NOT_FOUND
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
	const THREADS = 10
	wg.Add(THREADS)
	for i := 0; i < THREADS; i++ {
		go func() {
			for userID := range userIDs {
				repostID, err := findRepost(client, userID, post)
				if err != nil {
					errors <- err
					continue
				}
				if repostID != NOT_FOUND {
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
		errors := make(chan error)
		go func(err error) { errors <- err }(err)
		close(errors)
		close(shares)
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
