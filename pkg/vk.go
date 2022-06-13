package vkutils

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	f "github.com/rprtr258/goflow/fun"
	r "github.com/rprtr258/goflow/result"
	s "github.com/rprtr258/goflow/stream"
)

// vk api constants
const (
	// getUserListThreads     = 10
	groupsGetMembersLimit = 1000
	getFriendsLimit       = 5000
	getLikesListLimit     = 1000
)

// vk api error codes
const (
	tooManyRequests        = 6
	accessDenied           = 15
	userWasDeletedOrBanned = 18
	profileIsPrivate       = 30
)

// application constants
const (
	apiVersion        = "5.131"
	apiRequestRetries = 100
	waitTimeToRetry   = time.Millisecond * 500
)

// UserID is id of some user if positive or group if negative.
type UserID int

type User struct {
	ID         UserID `json:"id"`
	FirstName  string `json:"first_name"`
	SecondName string `json:"last_name"`
}

// UserList is a list of users from VK api.
type UserList struct {
	Response struct {
		Count uint   `json:"count"`
		Items []User `json:"items"`
	} `json:"response"`
}

// PostID is pair of post author ID and post index.
type PostID struct {
	OwnerID UserID `json:"owner_id"`
	ID      uint   `json:"id"`
}

// Post is post on some user or group wall.
// TODO: separate api structs and lib structs(?)
type Post struct {
	Owner       UserID   `json:"owner_id"`
	ID          uint     `json:"id"`
	Date        uint     `json:"date"`
	Text        string   `json:"text"`
	CopyHistory []PostID `json:"copy_history"`
}

type wallPostsResponse struct {
	Count uint   `json:"count"`
	Items []Post `json:"items"`
}

// WallPosts is a list of posts from user or group wall.
type WallPosts struct {
	Response wallPostsResponse `json:"response"`
}

type WallGetByIDResponse struct {
	Response []struct {
		Date uint `json:"date"`
	} `json:"response"`
}

type GroupsGetByIDResponse struct {
	Response []struct {
		ID int `json:"id"`
	} `json:"response"`
}

// VKClient is a client to VK api.
type VKClient struct {
	accessToken string
	client      http.Client
}

type postHiddenError struct {
	ownerID UserID
	postID  uint
}

func (p postHiddenError) Error() string {
	return fmt.Sprintf("Post %d_%d is hidden", p.ownerID, p.postID)
}

// VkError is vk api error.
type VkError struct {
	Code    uint   `json:"error_code"`
	Message string `json:"error_msg"`
}

func (err VkError) Error() string {
	return fmt.Sprintf("Error(%d) %s", err.Code, err.Message)
}

type VkErrorResponse struct {
	Err VkError `json:"error"`
}

type ApiCallError struct {
	vkError VkError
	method  string
	params  url.Values
}

func (err ApiCallError) Error() string {
	return fmt.Sprintf("error while %s(%v): %v", err.method, err.params, err.vkError)
}

// NewVKClient creates new VKClient.
func NewVKClient(accessToken string) VKClient {
	return VKClient{
		accessToken: accessToken,
		client:      *http.DefaultClient,
	}
}

func jsonUnmarshal[J any](body []byte) r.Result[J] {
	return r.Eval(func() (J, error) {
		var j J
		err := json.Unmarshal(body, &j)
		if err != nil {
			return j, fmt.Errorf("error while parsing '%s': %w", string(body), err)
		}
		return j, nil
	})
}

// TODO: print request, response, timing
func (client *VKClient) apiRequest(method string, params url.Values, params2 ...string) r.Result[[]byte] {
	for i := 0; i < len(params2); i += 2 {
		params.Set(params2[i], params2[i+1])
	}
	url := fmt.Sprintf("https://api.vk.com/method/%s", method)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return r.Err[[]byte](err)
	}
	reqParams := req.URL.Query()
	reqParams.Add("v", apiVersion)
	reqParams.Add("access_token", client.accessToken)
	for k, v := range params {
		reqParams.Add(k, v[0])
	}
	req.URL.RawQuery = reqParams.Encode()
	timeLimitTries := 0
	var resp *http.Response
	for timeLimitTries < apiRequestRetries {
		resp, err = client.client.Do(req)
		if err != nil {
			// TODO: fix, not working
			return r.Err[[]byte](err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Println(err)
			}
		}()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return r.Err[[]byte](err)
		}
		// log.Println("ON ", method, " ", params, " GOT:\n", string(body))
		// move out parsing response
		errr := jsonUnmarshal[VkErrorResponse](body)
		if errr.IsErr() {
			return r.Err[[]byte](err)
		}
		v := errr.Unwrap()
		if v.Err.Code != 0 {
			// TODO: define behavior on error (retry or throw error) in loop
			switch {
			case v.Err.Code == tooManyRequests:
				log.Printf("Too many requests on %s(%v)\n", method, params)
				time.Sleep(waitTimeToRetry)
				continue
			default:
				return r.Err[[]byte](ApiCallError{
					vkError: v.Err,
					method:  method,
					params:  params,
				})
			}
		}
		return r.Success(body)
	}
	return r.Err[[]byte](fmt.Errorf("%s(%v) = Timeout", method, params))
}

type pagedImpl[A any] struct {
	Pager[A]
}

func (xs *pagedImpl[A]) Next() f.Option[[]A] {
	pageResult := xs.NextPage()
	if pageResult.IsErr() {
		log.Printf("ERROR WHILE GETTING PAGE in %T(%[1]v): %v\n", xs.Pager, pageResult.UnwrapErr())
		return f.None[[]A]()
	}
	return pageResult.Unwrap()
}

func getPaged[A any](pager Pager[A]) s.Stream[A] {
	return s.Paged[A](&pagedImpl[A]{pager})
}

type Pager[A any] interface {
	NextPage() r.Result[f.Option[[]A]]
}

type userListPager struct {
	client    *VKClient
	method    string
	urlParams url.Values
	offset    uint
	total     f.Option[uint]
	pageSize  PageSize
}

func (pager *userListPager) NextPage() r.Result[f.Option[[]User]] {
	if pager.total.IsSome() && pager.offset >= pager.total.Unwrap() {
		return r.Success(f.None[[]User]())
	}
	// log.Println("GET USER LIST", xs.offset, xs.total)
	userList := r.FlatMap(
		pager.client.apiRequest(pager.method, pager.urlParams, "offset", fmt.Sprint(pager.offset)),
		jsonUnmarshal[UserList],
	)
	return r.Map(
		userList,
		func(ul UserList) f.Option[[]User] {
			pager.offset += uint(pager.pageSize)
			pager.total = f.Some(ul.Response.Count)
			return f.Some(ul.Response.Items)
		},
	)
}

func (client *VKClient) getUserList(method string, params url.Values, pageSize PageSize) s.Stream[User] {
	params.Set("count", fmt.Sprint(pageSize))
	return getPaged[User](&userListPager{
		offset:    0,
		total:     f.None[uint](),
		client:    client,
		method:    method,
		urlParams: params,
		pageSize:  pageSize,
	})
}

func (client *VKClient) getGroupMembers(groupID UserID) s.Stream[User] {
	return client.getUserList("groups.getMembers", MakeUrlValues(map[string]any{
		"group_id": -groupID,
		"fields":   "first_name,last_name",
	}), groupsGetMembersLimit)
}

func (client *VKClient) getFriends(userID UserID) s.Stream[User] {
	return client.getUserList("friends.get", MakeUrlValues(map[string]any{
		"user_id": userID,
		"fields":  "first_name,last_name",
	}), getFriendsLimit)
}

func (client *VKClient) getLikes(postID PostID) s.Stream[User] {
	return client.getUserList("likes.getList", MakeUrlValues(map[string]any{
		"type":     "post",
		"owner_id": postID.OwnerID,
		"item_id":  postID.ID,
		"skip_own": "0",
		"extended": "1",
	}), getLikesListLimit)
}

func (client *VKClient) getFollowers(userId UserID) s.Stream[User] {
	return client.getUserList("users.getFollowers", MakeUrlValues(map[string]any{
		"user_id": userId,
		"fields":  "first_name,last_name",
	}), 1000)
}

func (client *VKClient) getWallPosts(params url.Values, params2 ...string) r.Result[WallPosts] {
	body := client.apiRequest("wall.get", params, params2...)
	return r.FlatMap(body, jsonUnmarshal[WallPosts])
}

// TODO: return time.Time
func (client *VKClient) getPostTime(postID PostID) r.Result[uint] {
	body := client.apiRequest("wall.getById", MakeUrlValues(map[string]any{
		"posts": fmt.Sprintf("%d_%d", postID.OwnerID, postID.ID),
	}))
	userList := r.FlatMap(body, jsonUnmarshal[WallGetByIDResponse])
	return r.FlatMap(
		userList,
		func(v WallGetByIDResponse) r.Result[uint] {
			if len(v.Response) != 1 {
				return r.Err[uint](postHiddenError{postID.OwnerID, postID.ID})
			}
			return r.Success(v.Response[0].Date)
		},
	)
}

func (client *VKClient) getGroupID(groupName string) r.Result[UserID] {
	vR := r.FlatMap(
		client.apiRequest("groups.getById", MakeUrlValues(map[string]any{
			"group_id": groupName,
		})),
		jsonUnmarshal[GroupsGetByIDResponse],
	)
	return r.Map(vR, func(v GroupsGetByIDResponse) UserID { return UserID(-v.Response[0].ID) })
}

func MakeUrlValues(kvs map[string]any) url.Values {
	res := make(url.Values)
	for k, v := range kvs {
		res.Set(k, fmt.Sprint(v))
	}
	return res
}
