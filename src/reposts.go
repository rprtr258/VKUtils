package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
)

type Post struct {
	Owner UserID
	ID    int
}

type RepostSearchResult struct {
	Likes        int      `json:"likes"`
	TotalReposts int      `json:"totalReposts"`
	Reposters    []UserID `json:"reposters"`
}

func (client *VKClient) getTotalUsers(method string, params url.Values) uint {
	var v UserList
	params.Set("offset", "0")
	params.Set("count", "0")
	body := client.apiRequest(method, params)
	err := json.Unmarshal(body, &v)
	if err != nil {
		panic(err)
	}
	return v.Response.Count
}

func (client *VKClient) getUserList(method string, params url.Values, count uint) <-chan UserID {
	res := make(chan UserID)
	go func() {
		total := client.getTotalUsers(method, params)
		countString := fmt.Sprint(count)
		var wg sync.WaitGroup
		const THREADS = 10
		wg.Add(THREADS)
		for i := uint(0); i < THREADS; i++ {
			go func(start, step uint) {
				urlParams := make(url.Values)
				for k, v := range params {
					urlParams[k] = v
				}
				offset := start
				for offset < total {
					var v UserList
					urlParams.Set("offset", fmt.Sprint(offset))
					urlParams.Set("count", countString)
					body := client.apiRequest(method, urlParams)
					err := json.Unmarshal(body, &v)
					if err != nil {
						panic(err)
					}
					for _, userID := range v.Response.Items {
						res <- userID
					}
					offset += step
				}
				wg.Done()
			}(count*i, count*THREADS)
		}
		wg.Wait()
		close(res)
	}()
	return res
}

// TODO: move client to args
func (client *VKClient) getGroupMembers(groupID int) <-chan UserID {
	return client.getUserList("groups.getMembers", url.Values{
		"group_id": []string{fmt.Sprint(groupID)},
	}, 1000)
}

func (client *VKClient) getFriends(userID UserID) <-chan UserID {
	return client.getUserList("friends.get", url.Values{
		"user_id": []string{fmt.Sprint(userID)},
	}, 5000)
}

func (client *VKClient) getLikes(post Post) <-chan UserID {
	return client.getUserList("likes.getList", url.Values{
		"type":     []string{"post"},
		"owner_id": []string{fmt.Sprint(post.Owner)},
		"item_id":  []string{fmt.Sprint(post.ID)},
		"skip_own": []string{"0"},
	}, 1000)
}

func (client *VKClient) getPostRepostsCount(post Post) int {
	body := client.apiRequest("wall.getById", url.Values{
		"posts": []string{fmt.Sprintf("%d_%d", post.Owner, post.ID)},
	})
	var v struct {
		Response []struct {
			Reposts struct {
				Count int `json:"count"`
			} `json:"reposts"`
		} `json:"response"`
	}
	err := json.Unmarshal(body, &v)
	if err != nil {
		panic(err)
	}
	if len(v.Response) != 1 {
		panic("Post is hidden")
	}
	return v.Response[0].Reposts.Count
}

// TODO: parallelize
func doesHaveRepost(client *VKClient, userID UserID, post Post) bool {
	total := -1
	var offset uint = 0
	const count uint = 100
	countString, ownerIDString := fmt.Sprint(userID), fmt.Sprint(count)
	var v struct {
		Response struct {
			Count int `json:"count"`
			Items []struct {
				CopyHistory []struct {
					PostID  int    `json:"id"`
					OwnerID UserID `json:"owner_id"`
				} `json:"copy_history"`
			} `json:"items"`
		} `json:"response"`
	}
	for offset < uint(total) || total == -1 {
		body := client.apiRequest("wall.get", url.Values{
			"owner_id": []string{ownerIDString},
			"offset":   []string{fmt.Sprint(offset)},
			"count":    []string{countString},
		})
		err := json.Unmarshal(body, &v)
		if err != nil {
			panic(err)
		}
		if total == -1 {
			total = v.Response.Count
		}
		for _, item := range v.Response.Items {
			copyHistory := item.CopyHistory
			if len(copyHistory) != 0 && copyHistory[0].PostID == post.ID && copyHistory[0].OwnerID == post.Owner {
				return true
			}
		}
		offset += count
	}
	return false
}

func getUniqueIDs(client *VKClient, post Post, ownerID int, res *RepostSearchResult) <-chan UserID {
	var wg sync.WaitGroup
	userIDs := make(chan UserID)

	wg.Add(1)
	go func() {
		likers := client.getLikes(post)
		for userID := range likers {
			res.Likes++
			userIDs <- userID
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		var potentialUserIDs <-chan UserID
		if ownerID < 0 { // owner is group
			potentialUserIDs = client.getGroupMembers(-ownerID)
		} else { // owner is user
			potentialUserIDs = client.getFriends(UserID(ownerID))
		}
		for userID := range potentialUserIDs {
			userIDs <- userID
		}
		wg.Done()
	}()

	go func() {
		wg.Wait()
		close(userIDs)
	}()

	wasChecked := make(map[UserID]bool)
	toCheckQueue := make(chan UserID)
	go func() {
		for userID := range userIDs {
			if !wasChecked[userID] {
				toCheckQueue <- userID
				wasChecked[userID] = true
			}
		}
		close(toCheckQueue)
	}()
	return toCheckQueue
}

func getCheckedIDs(client *VKClient, post Post, ids <-chan UserID) <-chan UserID {
	resultQueue := make(chan UserID)
	var wg sync.WaitGroup
	const THREADS = 1000
	wg.Add(THREADS)
	for i := 1; i <= THREADS; i++ {
		go func() {
			for userID := range ids {
				if doesHaveRepost(client, userID, post) {
					resultQueue <- userID
					// TODO: consider if len(res.Reposters) == res.TotalReposts { break } // which is highly unlikely
				}
			}
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		close(resultQueue)
	}()
	return resultQueue
}

func getReposters(client *VKClient, postUrl string) RepostSearchResult {
	var ownerID, postID int
	fmt.Sscanf(postUrl, "https://vk.com/wall%d_%d", &ownerID, &postID)

	post := Post{
		Owner: UserID(ownerID),
		ID:    postID,
	}
	res := RepostSearchResult{
		TotalReposts: client.getPostRepostsCount(post),
	}

	uniqueIDs := getUniqueIDs(client, post, ownerID, &res)
	resultQueue := getCheckedIDs(client, post, uniqueIDs)

	res.Reposters = make([]UserID, 0)
	for userID := range resultQueue {
		res.Reposters = append(res.Reposters, userID)
	}

	return res
}
