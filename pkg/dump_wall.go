package vkutils

import (
	"fmt"
	"log"
	"net/url"

	f "github.com/rprtr258/goflow/fun"
	r "github.com/rprtr258/goflow/result"
	s "github.com/rprtr258/goflow/stream"
)

type postsPager struct {
	client *VKClient
	offset uint
	total  f.Option[uint]
	params url.Values
}

func (pager *postsPager) NextPage() r.Result[f.Option[[]Post]] {
	if pager.total.IsSome() && pager.offset >= pager.total.Unwrap() {
		return r.Success(f.None[[]Post]())
	}
	wallPosts := r.TryRecover(
		pager.client.getWallPosts(pager.params, "offset", fmt.Sprint(pager.offset)),
		func(err error) r.Result[WallPosts] {
			if errMsg, ok := err.(ApiCallError); ok {
				switch errMsg.vkError.Code {
				case accessDenied, userWasDeletedOrBanned, profileIsPrivate:
					return r.Success(WallPosts{Response: wallPostsResponse{
						Count: 0,
						Items: []Post{},
					}})
				}
			}
			log.Printf("Error getting posts: %T(%[1]v)\n", err)
			return r.Err[WallPosts](err)
		},
	)
	return r.Map(
		wallPosts,
		func(page WallPosts) f.Option[[]Post] {
			pager.offset += uint(wallGetPageSize)
			pager.total = f.Some(page.Response.Count)
			return f.Some(page.Response.Items)
		},
	)
}

// GetPosts gets posts stream from group.
func GetPosts(client *VKClient, groupName string) r.Result[s.Stream[Post]] {
	return r.Map(
		client.getGroupID(groupName),
		func(groupID UserID) s.Stream[Post] {
			return getPaged[Post](&postsPager{
				client: client,
				offset: 0,
				total:  f.None[uint](),
				params: MakeUrlValues(map[string]any{
					"owner_id": groupID,
					"count":    wallGetPageSize,
				}),
			})
		},
	)
}
