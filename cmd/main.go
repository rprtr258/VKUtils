package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

type VKClient struct {
	AccessToken string
	Client      http.Client
}

type UserID int

type Post struct {
	Owner UserID
	ID    int
}

type UserList struct {
	Response struct {
		Count int      `json:"count"`
		Items []UserID `json:"items"`
	} `json:"response"`
}

type RepostSearchResult struct {
	Likes        int      `json:"likes"`
	TotalReposts int      `json:"totalReposts"`
	Reposters    []UserID `json:"reposters"`
}

func (client *VKClient) apiRequest(method string, params url.Values) []byte {
	url := fmt.Sprintf("https://api.vk.com/method/%s", method)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	req_params := req.URL.Query()
	req_params.Add("v", "5.131")
	req_params.Add("access_token", client.AccessToken)
	for k, v := range params {
		req_params.Add(k, v[0])
	}
	req.URL.RawQuery = req_params.Encode()
	resp, err := client.Client.Do(req)
	if err != nil {
		// if user hid their wall
		log.Fatal(err)
		return []byte{}
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	return body
}

func (client *VKClient) getUserList(method string, params url.Values, count uint) []UserID {
	res := make([]UserID, 0)
	var v UserList
	var offset uint = 0
	total := -1
	countString := fmt.Sprint(count)
	for len(res) < total || total == -1 {
		params.Set("offset", fmt.Sprint(offset))
		params.Set("count", countString)
		body := client.apiRequest(method, params)
		err := json.Unmarshal(body, &v)
		if err != nil {
			panic(err)
		}
		if total == -1 {
			total = v.Response.Count
		}
		res = append(res, v.Response.Items...)
		offset += count
	}
	return res
}

func (client *VKClient) getGroupMembers(groupID int) []UserID {
	return client.getUserList("groups.getMembers", url.Values{
		"group_id": []string{fmt.Sprint(groupID)},
	}, 1000)
}

func (client *VKClient) getFriends(userID UserID) []UserID {
	return client.getUserList("friends.get", url.Values{
		"user_id": []string{fmt.Sprint(userID)},
	}, 5000)
}

func (client *VKClient) getLikes(post Post) []UserID {
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

	wasChecked := make(map[UserID]bool)
	toCheckQueue := make(chan UserID, 1000)
	resultQueue := make(chan UserID, 100)

	go func() {
		likers := client.getLikes(post)
		res.Likes = len(likers)
		for _, userID := range likers {
			if !wasChecked[userID] {
				toCheckQueue <- userID
				wasChecked[userID] = true
			}
		}

		var usersToSearch []UserID
		if ownerID < 0 {
			// owner is group
			usersToSearch = client.getGroupMembers(-ownerID)
		} else {
			// owner is user
			usersToSearch = client.getFriends(UserID(ownerID))
		}
		for _, userID := range usersToSearch {
			if !wasChecked[userID] {
				toCheckQueue <- userID
				wasChecked[userID] = true
			}
		}
		close(toCheckQueue)
	}()

	done := make(chan bool)
	doneAll := make(chan bool)
	const THREADS = 1000
	for i := 1; i <= THREADS; i++ {
		go func() {
			for userID := range toCheckQueue {
				if doesHaveRepost(client, userID, post) {
					resultQueue <- userID
					// TODO: consider if len(res.Reposters) == res.TotalReposts { break } // which is highly unlikely
				}
			}
			done <- true
		}()
	}
	go func() {
		for i := 1; i <= THREADS; i++ {
			<-done
		}
		close(resultQueue)
		doneAll <- true
	}()

	res.Reposters = make([]UserID, 0)
	<-doneAll
	for userID := range resultQueue {
		res.Reposters = append(res.Reposters, userID)
	}

	return res
}

func handler(w http.ResponseWriter, r *http.Request) {
	log.Println(*r)
	client := VKClient{
		AccessToken: os.Getenv("VK_ACCESS_TOKEN"),
	}
	start := time.Now()
	response := getReposters(&client, r.FormValue("postUrl"))
	log.Printf("Time elapsed %v", time.Since(start))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	if _, presented := os.LookupEnv("VK_ACCESS_TOKEN"); !presented {
		panic(fmt.Sprintf("%s was not found in env vars", "VK_ACCESS_TOKEN"))
	}
	log.Println("Server started successfully on http://localhost:8000")
	http.HandleFunc("/reposts", handler) // each request calls handler
	log.Fatal(http.ListenAndServe("localhost:8000", nil))
}
