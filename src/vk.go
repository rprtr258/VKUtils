package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type UserID int

type UserList struct {
	Response struct {
		Count uint     `json:"count"`
		Items []UserID `json:"items"`
	} `json:"response"`
}

type Post struct {
	Owner UserID `json:"owner_id"`
	ID    uint   `json:"id"`
	Date  uint   `json:"date"`
	Text  string `json:"text"`
}

type VKClient struct {
	AccessToken string
	Client      http.Client
}

// vk api constants
const (
	getUserList_threads     = 10
	groups_getMembers_limit = 1000
	friends_get_limit       = 5000
	likes_getList_limit     = 1000
)

// application constants
const (
	api_version         = "5.131"
	API_REQUEST_RETRIES = 100
	WAIT_TIME_TO_RETRY  = time.Millisecond * 500
)

// vk api error codes
type VkError struct {
	Code    uint   `json:"error_code"`
	Message string `json:"error_msg"`
}

const (
	TOO_MANY_REQUESTS = 6
	ACCESS_DENIED     = 15
	USER_BANNED       = 18
	USER_HIDDEN       = 30
)

func (err *VkError) Error() string {
	return fmt.Sprintf("Error(%d) %s", err.Code, err.Message)
}

func (client *VKClient) apiRequest(method string, params url.Values) (res []byte, err error) {
	url := fmt.Sprintf("https://api.vk.com/method/%s", method)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	req_params := req.URL.Query()
	req_params.Add("v", api_version)
	req_params.Add("access_token", client.AccessToken)
	for k, v := range params {
		req_params.Add(k, v[0])
	}
	req.URL.RawQuery = req_params.Encode()
	timeLimitTries := 0
	var resp *http.Response
	var body []byte
	for timeLimitTries < API_REQUEST_RETRIES {
		resp, err = client.Client.Do(req)
		if err != nil {
			// if user hid their wall
			return
		}
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		var v struct {
			Err VkError `json:"error"`
		}
		err = json.Unmarshal(body, &v)
		if err != nil {
			return
		}
		if v.Err.Code != 0 {
			// TODO: define behavior on error (retry or throw error) in loop
			if v.Err.Code == TOO_MANY_REQUESTS {
				time.Sleep(WAIT_TIME_TO_RETRY)
				continue
			} else if v.Err.Code == ACCESS_DENIED || v.Err.Code == USER_BANNED || v.Err.Code == USER_HIDDEN {
				return
			} else {
				err = fmt.Errorf("%s(%v) = Error(%d) %s", method, params, v.Err.Code, v.Err.Message)
			}
			return
		}
		res = body
		return
	}
	err = fmt.Errorf("%s(%v) = Timeout", method, params)
	return
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
		wg.Add(getUserList_threads)
		STEP := count * getUserList_threads
		for i := uint(0); i < getUserList_threads; i++ {
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

func (client *VKClient) getGroupMembers(groupID UserID) (<-chan UserID, <-chan error) {
	return client.getUserList("groups.getMembers", url.Values{
		"group_id": []string{fmt.Sprint(-groupID)},
	}, groups_getMembers_limit)
}

func (client *VKClient) getFriends(userID UserID) (<-chan UserID, <-chan error) {
	return client.getUserList("friends.get", url.Values{
		"user_id": []string{fmt.Sprint(userID)},
	}, friends_get_limit)
}

func (client *VKClient) getLikes(ownerId UserID, postId uint) (<-chan UserID, <-chan error) {
	return client.getUserList("likes.getList", url.Values{
		"type":     []string{"post"},
		"owner_id": []string{fmt.Sprint(ownerId)},
		"item_id":  []string{fmt.Sprint(postId)},
		"skip_own": []string{"0"},
	}, likes_getList_limit)
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
