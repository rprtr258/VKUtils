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
		jsonUnmarshall[V],
	)
	return r.Map(v, func(v V) uint {
		return v.Response.Count
	})
}

func parseGroupURL(groupURL string) r.Result[string] {
	var groupName string
	if _, err := fmt.Sscanf(groupURL, "https://vk.com/%s", &groupName); err != nil {
		return r.Err[string](err)
	}
	return r.Success(groupName)
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
func GetReversedPosts(client *VKClient, groupURL string) r.Result[s.Stream[Post]] {
	groupNameR := parseGroupURL(groupURL)
	vR := r.FlatMap2(
		groupNameR,
		func(groupName string) r.Result[[]byte] {
			return r.FromGoResult(client.apiRequestRaw("groups.getById", MakeUrlValues("group_id", groupName)))
		},
		jsonUnmarshall[V],
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
				jsonUnmarshall[W],
				func(w W) r.Result[s.Stream[Post]] {
					return r.Success(s.FromSlice(s.ReverseSlice(w.Response.Items)))
				},
			)
		},
	)
}
