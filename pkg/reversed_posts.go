package vkutils

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// ReversePostsResult is a list of posts
type ReversePostsResult []Post

// TODO: replace with get-from-first also
func getPostsCount(client *VKClient, userID UserID) (postsCount uint, err error) {
	ownerIDString := fmt.Sprint(userID)
	var v struct {
		Response struct {
			Count uint `json:"count"`
		} `json:"response"`
	}
	body, err := client.apiRequestRaw("wall.get", url.Values{
		"owner_id": []string{ownerIDString},
		"offset":   []string{"0"},
		"count":    []string{"1"},
	})
	if err != nil {
		return
	}
	err = json.Unmarshal(body, &v)
	if err != nil {
		return
	}
	postsCount = v.Response.Count
	return
}

// GetReversedPosts gets reversed posts from group
func GetReversedPosts(client *VKClient, groupURL string) (res ReversePostsResult, err error) {
	var groupName string
	fmt.Sscanf(groupURL, "https://vk.com/%s", &groupName)

	params := make(url.Values)
	params.Set("group_id", groupName)
	body, err := client.apiRequestRaw("groups.getById", params)
	if err != nil {
		return
	}
	var v struct {
		Response []struct {
			ID int `json:"id"`
		} `json:"response"`
	}
	err = json.Unmarshal(body, &v)
	if err != nil {
		return
	}
	groupID := UserID(-v.Response[0].ID)
	postsCount, err := getPostsCount(client, groupID)
	if err != nil {
		return
	}
	const maxPostsCount = 100
	var offset uint
	if postsCount < maxPostsCount {
		offset = 0
	} else {
		offset = postsCount - maxPostsCount
	}
	body, err = client.apiRequestRaw("wall.get", url.Values{
		"owner_id": []string{fmt.Sprint(groupID)},
		"offset":   []string{fmt.Sprint(offset)},
		"count":    []string{fmt.Sprint(maxPostsCount)},
	})
	if err != nil {
		return
	}
	var w struct {
		Response struct {
			Items []struct {
				ID      uint   `json:"id"`
				OwnerID UserID `json:"owner_id"`
				Text    string `json:"text"`
				Date    uint   `json:"date"`
			} `json:"items"`
		} `json:"response"`
	}
	err = json.Unmarshal(body, &w)
	if err != nil {
		panic(err)
	}
	res = make(ReversePostsResult, 0, len(w.Response.Items))
	for i := len(w.Response.Items) - 1; i >= 0; i-- {
		res = append(res, Post{
			// Link: fmt.Sprintf("https://vk.com/wall%d_%d", s.OwnerId, s.Id),
			Owner: w.Response.Items[i].OwnerID,
			ID:    w.Response.Items[i].ID,
			Date:  w.Response.Items[i].Date,
			// Text:  w.Response.Items[i].Text, // TODO: come back
		})
	}
	return
}
