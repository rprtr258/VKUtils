package vkutils

import (
	"fmt"
	"net/url"

	r "github.com/rprtr258/goflow/result"
	"github.com/rprtr258/goflow/slice"
	s "github.com/rprtr258/goflow/stream"
)

func max0XminusY(x uint, y uint) uint {
	if x < y {
		return 0
	}
	return x - y
}

// TODO: remove
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
					client.apiRequest("wall.get", url.Values{
						"owner_id": []string{fmt.Sprint(groupID)},
						"offset":   []string{"0"},
						"count":    []string{"1"},
					}),
					jsonUnmarshal[V],
				), func(v V) uint {
					return v.Response.Count
				}),
				func(postsCount uint) r.Result[[]byte] {
					return client.apiRequest("wall.get", MakeUrlValues(
						"owner_id", fmt.Sprint(groupID),
						"offset", fmt.Sprint(max0XminusY(postsCount, uint(wallGetPageSize))),
						"count", fmt.Sprint(wallGetPageSize),
					))
				},
				jsonUnmarshal[WallPosts],
				func(w WallPosts) r.Result[s.Stream[Post]] {
					slice.ReverseInplace(w.Response.Items)
					return r.Success(s.FromSlice(w.Response.Items))
				},
			)
		},
	)
}
