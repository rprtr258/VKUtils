package vkutils

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	f "github.com/rprtr258/vk-utils/flow/fun"
	i "github.com/rprtr258/vk-utils/flow/io"
	s "github.com/rprtr258/vk-utils/flow/stream"
)

// UserID is id of some user
type UserID int

// UserList is a list of users from VK api
type UserList struct {
	Response struct {
		Count uint     `json:"count"`
		Items []UserID `json:"items"`
	} `json:"response"`
}

// Post is VK wall post
type Post struct {
	Owner UserID `json:"owner_id"`
	ID    uint   `json:"id"`
	Date  uint   `json:"date"`
	Text  string `json:"text"`
}

// VKClient is a client to VK api
type VKClient struct {
	AccessToken string
	Client      http.Client
}

// vk api constants
const (
	// getUserListThreads     = 10
	groupsGetMembersLimit = 1000
	getFriendsLimit       = 5000
	getLikesListLimit     = 1000
)

// application constants
const (
	apiVersion        = "5.131"
	apiRequestRetries = 100
	waitTimeToRetry   = time.Millisecond * 500
)

// VkError is vk api error
type VkError struct {
	Code    uint   `json:"error_code"`
	Message string `json:"error_msg"`
}

// vk api error codes
const (
	tooManyRequests = 6
	accessDenied    = 15
	userBanned      = 18
	userHidden      = 30
)

func (err *VkError) Error() string {
	return fmt.Sprintf("Error(%d) %s", err.Code, err.Message)
}

func jsonUnmarshall[J any](body []byte) i.Result[J] {
	return i.Eval(func() (J, error) {
		var v J
		err := json.Unmarshal(body, &v)
		if err != nil {
			var j J
			return j, fmt.Errorf("error while parsing '%s': %w", string(body), err)
		}
		return v, nil
	})
}

// TODO: print request, response, timing
func (client *VKClient) apiRequestRaw(method string, params url.Values) (body []byte, err error) {
	url := fmt.Sprintf("https://api.vk.com/method/%s", method)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	reqParams := req.URL.Query()
	reqParams.Add("v", apiVersion)
	reqParams.Add("access_token", client.AccessToken)
	for k, v := range params {
		reqParams.Add(k, v[0])
	}
	req.URL.RawQuery = reqParams.Encode()
	timeLimitTries := 0
	var resp *http.Response
	for timeLimitTries < apiRequestRetries {
		resp, err = client.Client.Do(req)
		if err != nil {
			// if user hid their wall
			// TODO: fix, not working
			return
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Println(err)
			}
		}()
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return
		}
		// log.Println("GOT: ", string(body), " ON ", method, " ", params)
		var v struct {
			Err VkError `json:"error"`
		}
		err = json.Unmarshal(body, &v)
		if err != nil {
			return
		}
		if v.Err.Code != 0 {
			// TODO: define behavior on error (retry or throw error) in loop
			if v.Err.Code == tooManyRequests {
				time.Sleep(waitTimeToRetry)
				continue
			} else if v.Err.Code == accessDenied || v.Err.Code == userBanned || v.Err.Code == userHidden {
				return
			} else {
				err = fmt.Errorf("%s(%v) = Error(%d) %s", method, params, v.Err.Code, v.Err.Message)
			}
			return
		}
		return
	}
	err = fmt.Errorf("%s(%v) = Timeout", method, params)
	return
}

func apiRequest(client *VKClient, method string, params url.Values) i.Result[[]byte] {
	return i.Eval(func() ([]byte, error) {
		return client.apiRequestRaw(method, params)
	})
}

type userListImpl struct {
	client    *VKClient
	method    string
	params    url.Values
	count     uint
	offset    uint
	total     f.Option[uint]
	urlParams url.Values

	curPage s.Stream[UserID]
}

func (xs *userListImpl) Next() f.Option[UserID] {
	if x := xs.curPage.Next(); x.IsSome() {
		return x
	}

	// log.Println("GET USER LIST", xs.offset)
	if /*xs.offset == 0*/ xs.total.IsNone() || xs.offset < xs.total.Unwrap() {
		xs.urlParams.Set("offset", fmt.Sprint(xs.offset))
		body := apiRequest(xs.client, xs.method, xs.urlParams)
		userList := i.FlatMap(body, jsonUnmarshall[UserList])
		onePageOfUsers := i.Map(
			userList,
			func(v UserList) f.Pair[uint, s.Stream[UserID]] {
				return f.NewPair(
					v.Response.Count,
					s.FromSlice(v.Response.Items),
				)
			},
		)
		return i.Fold(
			onePageOfUsers,
			func(totalAndFirstBatch f.Pair[uint, s.Stream[UserID]]) f.Option[UserID] {
				xs.offset += xs.count
				xs.total, xs.curPage = f.Some(totalAndFirstBatch.Left), totalAndFirstBatch.Right
				return xs.curPage.Next()
			},
			func(err error) f.Option[UserID] {
				log.Println("ERROR WHILE GETTING PAGE: ", err)
				return f.None[UserID]()
			},
		)
	}
	return f.None[UserID]()
}

// TODO: merge method and count in one structure
func (client *VKClient) getUserList(method string, params url.Values, count uint) s.Stream[UserID] {
	countString := fmt.Sprint(count)
	urlParams := make(url.Values)
	for k, v := range params {
		urlParams[k] = v
	}
	urlParams.Set("count", countString)
	return &userListImpl{
		client:    client,
		method:    method,
		params:    params,
		count:     count,
		offset:    0,
		total:     f.None[uint](),
		urlParams: urlParams,
		curPage:   s.NewStreamEmpty[UserID](),
	}
}

// TODO: group id is groupid, user id is userid
func (client *VKClient) getGroupMembers(groupID UserID) s.Stream[UserID] {
	return client.getUserList("groups.getMembers", url.Values{
		"group_id": []string{fmt.Sprint(-groupID)},
	}, groupsGetMembersLimit)
}

func (client *VKClient) getFriends(userID UserID) s.Stream[UserID] {
	return client.getUserList("friends.get", url.Values{
		"user_id": []string{fmt.Sprint(userID)},
	}, getFriendsLimit)
}

func (client *VKClient) getLikes(ownerID UserID, postID uint) s.Stream[UserID] {
	return client.getUserList("likes.getList", url.Values{
		"type":     []string{"post"},
		"owner_id": []string{fmt.Sprint(ownerID)},
		"item_id":  []string{fmt.Sprint(postID)},
		"skip_own": []string{"0"},
	}, getLikesListLimit)
}

type postHiddenError struct {
	*Post
}

func (p postHiddenError) Error() string {
	return fmt.Sprintf("Post %d_%d is hidden", p.Owner, p.ID)
}

func (client *VKClient) getPostTime(post Post) i.Result[uint] {
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
		func(v V) i.Result[uint] {
			if len(v.Response) != 1 {
				return i.Fail[uint](postHiddenError{&post})
			}
			return i.Success(v.Response[0].Date)
		},
	)
}
