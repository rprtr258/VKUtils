package vkutils

import (
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
					client.apiRequest("wall.get", MakeUrlValues(map[string]any{
						"owner_id": groupID,
						"offset":   "0",
						"count":    "1",
					})),
					jsonUnmarshal[V],
				), func(v V) uint {
					return v.Response.Count
				}),
				func(postsCount uint) r.Result[[]byte] {
					return client.apiRequest("wall.get", MakeUrlValues(map[string]any{
						"owner_id": groupID,
						"offset":   max0XminusY(postsCount, uint(wallGetPageSize)),
						"count":    wallGetPageSize,
					}))
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
