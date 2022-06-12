package vkutils

import (
	"fmt"
	"net/url"

	r "github.com/rprtr258/goflow/result"
	s "github.com/rprtr258/goflow/stream"
)

const maxPostsCount = 100

// TODO: replace with get-from-first also.
func getPostsCount(client *VKClient, userID UserID) r.Result[uint] {
	ownerIDString := fmt.Sprint(userID)
	type V struct {
		Response struct {
			Count uint `json:"count"`
		} `json:"response"`
	}
	v := r.FlatMap(
		r.FromGoResult(client.apiRequestRaw("wall.get", url.Values{
			"owner_id": []string{ownerIDString},
			"offset":   []string{"0"},
			"count":    []string{"1"},
		})),
		jsonUnmarshal[V],
	)
	return r.Map(v, func(v V) uint {
		return v.Response.Count
	})
}

type V struct {
	Response []struct {
		ID int `json:"id"`
	} `json:"response"`
}

type W struct {
	Response struct {
		Items []Post `json:"items"`
	} `json:"response"`
}

func max0XminusY(x uint, y uint) uint {
	if x < y {
		return 0
	}
	return x - y
}

// GetReversedPosts gets reversed posts from group.
// TODO: fix to really get all posts
func GetReversedPosts(client *VKClient, groupName string) r.Result[s.Stream[Post]] {
	// TODO: move out getting group id by name
	vR := r.FlatMap(
		r.FromGoResult(client.apiRequestRaw("groups.getById", MakeUrlValues("group_id", groupName))),
		jsonUnmarshal[V],
	)
	groupIDR := r.Map(vR, func(v V) UserID { return UserID(-v.Response[0].ID) })
	return r.FlatMap(
		groupIDR,
		func(groupID UserID) r.Result[s.Stream[Post]] {
			return r.FlatMap3(
				getPostsCount(client, groupID),
				func(postsCount uint) r.Result[[]byte] {
					return r.FromGoResult(client.apiRequestRaw("wall.get", MakeUrlValues(
						"owner_id", fmt.Sprint(groupID),
						"offset", fmt.Sprint(max0XminusY(postsCount, maxPostsCount)),
						"count", fmt.Sprint(maxPostsCount),
					)))
				},
				jsonUnmarshal[W],
				func(w W) r.Result[s.Stream[Post]] {
					return r.Success(s.FromSlice(s.ReverseSlice(w.Response.Items)))
				},
			)
		},
	)
}
