package vkutils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	f "github.com/primetalk/goio/fun"
	i "github.com/primetalk/goio/io"
	s "github.com/primetalk/goio/stream"
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
	// getUserList_threads     = 10
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

func jsonUnmarshall[J any](body []byte) i.IO[J] {
	return i.Eval(func() (J, error) {
		var v J
		err := json.Unmarshal(body, &v)
		return v, errors.Wrap(err, string(body))
	})
}

// TODO: print request, response, timing
func (client *VKClient) apiRequestRaw(method string, params url.Values) (res []byte, err error) {
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
		defer resp.Body.Close()
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

func apiRequest(client *VKClient, method string, params url.Values) i.IO[[]byte] {
	return i.Eval(func() ([]byte, error) {
		return client.apiRequestRaw(method, params)
	})
}

// TODO: returns total and stream, make better interface (how?)
func getUserListImplImpl(client *VKClient, method string, params url.Values, countString string, offset uint, urlParams url.Values) i.IO[f.Pair[uint, s.Stream[UserID]]] {
	urlParams.Set("offset", fmt.Sprint(offset))
	urlParams.Set("count", countString)
	body := apiRequest(client, method, urlParams)
	userList := i.FlatMap(body, jsonUnmarshall[UserList])
	return i.Map(
		userList,
		func(v UserList) f.Pair[uint, s.Stream[UserID]] {
			return f.NewPair(
				v.Response.Count,
				s.FromSlice(v.Response.Items),
			)
		},
	)
}

// TODO: rewrite to loop?
func getUserListImpl(client *VKClient, method string, params url.Values, count uint, countString string, offset uint, total uint, urlParams url.Values) s.Stream[s.Stream[UserID]] {
	var stepResultIo i.IO[s.StepResult[s.Stream[UserID]]]
	newOffset := offset + count
	if offset == 0 {
		stepResultIo = i.Map(
			getUserListImplImpl(client, method, params, countString, newOffset, urlParams),
			func(totalAndFirstBatch f.Pair[uint, s.Stream[UserID]]) s.StepResult[s.Stream[UserID]] {
				total, firstBatch := totalAndFirstBatch.V1, totalAndFirstBatch.V2
				return s.NewStepResult(
					firstBatch,
					getUserListImpl(client, method, params, count, countString, newOffset, total, urlParams),
				)
			},
		)
	} else if offset >= total {
		stepResultIo = i.Lift(s.NewStepResultFinished[s.Stream[UserID]]())
	} else {
		stepResultIo = i.Map(
			getUserListImplImpl(client, method, params, countString, newOffset, urlParams),
			func(somethingAndFirstBatch f.Pair[uint, s.Stream[UserID]]) s.StepResult[s.Stream[UserID]] {
				return s.NewStepResult(
					somethingAndFirstBatch.V2,
					getUserListImpl(client, method, params, count, countString, newOffset, total, urlParams),
				)
			},
		)
	}
	return s.FromStepResult(stepResultIo)
}

// TODO: merge method and count in one structure
func (client *VKClient) getUserList(method string, params url.Values, count uint) s.Stream[UserID] {
	countString := fmt.Sprint(count)
	urlParams := make(url.Values)
	for k, v := range params {
		urlParams[k] = v
	}
	return s.Flatten(getUserListImpl(client, method, params, count, countString, 0, 42 /*TODO: remove*/, urlParams))
}

// TODO: group id is groupid, user id is userid
func (client *VKClient) getGroupMembers(groupID UserID) s.Stream[UserID] {
	return client.getUserList("groups.getMembers", url.Values{
		"group_id": []string{fmt.Sprint(-groupID)},
	}, groups_getMembers_limit)
}

func (client *VKClient) getFriends(userID UserID) s.Stream[UserID] {
	return client.getUserList("friends.get", url.Values{
		"user_id": []string{fmt.Sprint(userID)},
	}, friends_get_limit)
}

func (client *VKClient) getLikes(ownerId UserID, postId uint) s.Stream[UserID] {
	return client.getUserList("likes.getList", url.Values{
		"type":     []string{"post"},
		"owner_id": []string{fmt.Sprint(ownerId)},
		"item_id":  []string{fmt.Sprint(postId)},
		"skip_own": []string{"0"},
	}, likes_getList_limit)
}

type PostHiddenErr struct {
	*Post
}

func (p PostHiddenErr) Error() string {
	return fmt.Sprintf("Post %d_%d is hidden", p.Owner, p.ID)
}

func (client *VKClient) getPostTime(post Post) i.IO[uint] {
	body := apiRequest(client, "wall.getById", url.Values{
		"posts": []string{fmt.Sprintf("%d_%d", post.Owner, post.ID)},
	})
	type V struct {
		Response []struct {
			Date    uint `json:"date"`
			Reposts struct {
				Count int `json:"count"`
			} `json:"reposts"`
		} `json:"response"`
	}
	userList := i.FlatMap(body, jsonUnmarshall[V])
	return i.FlatMap(
		userList,
		func(v V) i.IO[uint] {
			if len(v.Response) != 1 {
				return i.Fail[uint](PostHiddenErr{&post})
			} else {
				return i.Lift(v.Response[0].Date)
			}
		},
	)
}
