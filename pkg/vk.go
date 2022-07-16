package vkutils

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	f "github.com/rprtr258/go-flow/fun"
	r "github.com/rprtr258/go-flow/result"
	s "github.com/rprtr258/go-flow/stream"
)

// vk api constants
const (
	groupsGetMembersPageSize  = PageSize(1000)
	getFriendsPageSize        = PageSize(5000)
	wallGetCommentsPageSize   = PageSize(100)
	getLikesPageSize          = PageSize(1000)
	usersGetFollowersPageSize = PageSize(1000)
	apiUrl                    = "https://api.vk.com/method/"
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

type GetCommentsResponse struct {
	Response struct {
		Count uint `json:"count"`
		Items []struct {
			ID       uint   `json:"id"`
			AuthorID UserID `json:"from_id"`
			Thread   struct {
				Count uint `json:"count"`
			} `json:"thread"`
		} `json:"items"`
		Profiles []struct {
			ID        UserID `json:"id"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
		} `json:"profiles"`
		Groups []struct {
			// unsigned because api authors decided it would be funny
			ID   uint   `json:"id"`
			Name string `json:"name"`
		} `json:"groups"`
	} `json:"response"`
}

// VKClient is a client to VK api.
type VKClient struct {
	accessToken string
	client      http.Client
	logAPICalls bool
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

func (client *VKClient) apiRequest(method string, params url.Values, params2 ...string) r.Result[[]byte] {
	for i := 0; i < len(params2); i += 2 {
		params.Set(params2[i], params2[i+1])
	}
	methodUrl := fmt.Sprintf("%s%s", apiUrl, method)
	req, err := http.NewRequest(http.MethodGet, methodUrl, nil)
	if err != nil {
		return r.Err[[]byte](err)
	}

	reqParams := make(url.Values)
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
		client.mylog("VK API REQUEST", map[string]any{
			"method":   method,
			"params":   params,
			"response": string(body),
		})
		// move out parsing response
		errr := jsonUnmarshal[VkErrorResponse](body)
		if errr.IsErr() {
			return r.Err[[]byte](err)
		}
		v := errr.Unwrap()
		if v.Err.Code != 0 {
			switch {
			case v.Err.Code == tooManyRequests:
				log.Printf("TOO MANY REQUESTS: method=%s params=%+v\n", method, params)
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
	}), groupsGetMembersPageSize)
}

func (client *VKClient) getFriends(userID UserID) s.Stream[User] {
	return client.getUserList("friends.get", MakeUrlValues(map[string]any{
		"user_id": userID,
		"fields":  "first_name,last_name",
	}), getFriendsPageSize)
}

func (client *VKClient) getLikes(postID PostID) s.Stream[User] {
	return client.getUserList("likes.getList", MakeUrlValues(map[string]any{
		"type":     "post",
		"owner_id": postID.OwnerID,
		"item_id":  postID.ID,
		"skip_own": "0",
		"extended": "1",
	}), getLikesPageSize)
}

func (client *VKClient) getFollowers(userID UserID) s.Stream[User] {
	return client.getUserList("users.getFollowers", MakeUrlValues(map[string]any{
		"user_id": userID,
		"fields":  "first_name,last_name",
	}), usersGetFollowersPageSize)
}

func (client *VKClient) GetComments(postID PostID) s.Stream[User] {
	groupNames := map[UserID]string{}
	userNames := map[UserID]f.Pair[string, string]{}
	res := f.NewSet[UserID]()
	total := f.None[uint]()
	totalSeen := uint(0)
	offset := uint(0)
	params := MakeUrlValues(map[string]any{
		"owner_id": postID.OwnerID,
		"post_id":  postID.ID,
		"extended": 1,
		"fields":   "first_name,last_name,name",
		"count":    wallGetCommentsPageSize,
	})
	commentsThreadsToCheck := make([]f.Pair[uint, uint], 0)
	for total.IsNone() || offset < total.Unwrap() {
		kk := client.apiRequest("wall.getComments", params, "offset", fmt.Sprint(offset))
		k := r.FlatMap(kk, jsonUnmarshal[GetCommentsResponse])
		if k.IsErr() {
			log.Println(k.UnwrapErr())
			break
		}
		k0 := k.Unwrap()
		total = f.Some(k0.Response.Count)
		totalSeen += uint(len(k0.Response.Items))
		offset += uint(wallGetCommentsPageSize)
		for _, item := range k0.Response.Items {
			res.Add(item.AuthorID)
			if item.Thread.Count > 0 {
				commentsThreadsToCheck = append(commentsThreadsToCheck, f.NewPair(item.ID, item.Thread.Count))
			}
		}
		for _, profile := range k0.Response.Profiles {
			userNames[profile.ID] = f.NewPair(profile.FirstName, profile.LastName)
		}
		for _, group := range k0.Response.Groups {
			groupNames[-UserID(group.ID)] = group.Name
		}
	}
	for _, commentIDAndThreadSize := range commentsThreadsToCheck {
		for offset := uint(0); offset < commentIDAndThreadSize.Right; offset++ {
			kk := client.apiRequest("wall.getComments", params, "comment_id", fmt.Sprint(commentIDAndThreadSize.Left), "offset", fmt.Sprint(offset))
			k := r.FlatMap(kk, jsonUnmarshal[GetCommentsResponse])
			if k.IsErr() {
				log.Println(k.UnwrapErr())
				break
			}
			k0 := k.Unwrap()
			for _, item := range k0.Response.Items {
				res.Add(item.AuthorID)
			}
			for _, profile := range k0.Response.Profiles {
				userNames[profile.ID] = f.NewPair(profile.FirstName, profile.LastName)
			}
			for _, group := range k0.Response.Groups {
				groupNames[-UserID(group.ID)] = group.Name
			}
		}
	}
	return s.Map(
		s.FromSet(res),
		func(userID UserID) User {
			if userID < 0 {
				return User{
					ID:         userID,
					FirstName:  groupNames[userID],
					SecondName: "_GROUP",
				}
			}
			userName := userNames[userID]
			return User{
				ID:         userID,
				FirstName:  userName.Left,
				SecondName: userName.Right,
			}
		},
	)
}

func (client *VKClient) getWallPosts(params url.Values, params2 ...string) r.Result[WallPosts] {
	body := client.apiRequest("wall.get", params, params2...)
	return r.FlatMap(body, jsonUnmarshal[WallPosts])
}

// getPostTime returns post creation time as Unix uint.
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

func (client *VKClient) mylog(message string, data map[string]any) {
	if !client.logAPICalls {
		return
	}
	log.Print(message, ": ")
	for k, v := range data {
		log.Printf("%s=%+v ", k, v)
	}
}
