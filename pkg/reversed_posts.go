package vkutils

import (
	"fmt"
	"net/url"

	r "github.com/rprtr258/goflow/result"
	s "github.com/rprtr258/goflow/stream"
)

const maxPostsCount = 100

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

type V struct {
	Response struct {
		Count uint `json:"count"`
	} `json:"response"`
}

// GetReversedPosts gets reversed posts from group.
// TODO: replace with get-from-first also.
// TODO: fix to really get all posts
func GetReversedPosts(client *VKClient, groupName string) r.Result[s.Stream[Post]] {
	return r.FlatMap(
		client.getGroupID(groupName),
		func(groupID UserID) r.Result[s.Stream[Post]] {
			return r.FlatMap3(
				r.Map(r.FlatMap(
					r.FromGoResult(client.apiRequestRaw("wall.get", url.Values{
						"owner_id": []string{fmt.Sprint(groupID)},
						"offset":   []string{"0"},
						"count":    []string{"1"},
					})),
					jsonUnmarshal[V],
				), func(v V) uint {
					return v.Response.Count
				}),
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
