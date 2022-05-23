package vkutils

import (
	"encoding/json"
	"fmt"
	"net/url"
)

type ReversePostsResult []Post

// TODO: replace with get-from-first also
func getPostsCount(client *VKClient, userID UserID) (postsCount uint, err error) {
	ownerIDString := fmt.Sprint(userID)
	var v struct {
		Response struct {
			Count uint `json:"count"`
		} `json:"response"`
	}
	body, err := client.apiRequest("wall.get", url.Values{
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

func GetReversedPosts(client *VKClient, groupUrl string) (res ReversePostsResult, err error) {
	var groupName string
	fmt.Sscanf(groupUrl, "https://vk.com/%s", &groupName)

	params := make(url.Values)
	params.Set("group_id", groupName)
	body, err := client.apiRequest("groups.getById", params)
	if err != nil {
		return
	}
	var v struct {
		Response []struct {
			Id int `json:"id"`
		} `json:"response"`
	}
	err = json.Unmarshal(body, &v)
	if err != nil {
		return
	}
	groupId := UserID(-v.Response[0].Id)
	postsCount, err := getPostsCount(client, groupId)
	if err != nil {
		return
	}
	const MAX_POSTS_COUNT = 100
	var offset uint
	if postsCount < MAX_POSTS_COUNT {
		offset = 0
	} else {
		offset = postsCount - MAX_POSTS_COUNT
	}
	body, err = client.apiRequest("wall.get", url.Values{
		"owner_id": []string{fmt.Sprint(groupId)},
		"offset":   []string{fmt.Sprint(offset)},
		"count":    []string{fmt.Sprint(MAX_POSTS_COUNT)},
	})
	if err != nil {
		return
	}
	var w struct {
		Response struct {
			Items []struct {
				Id      uint   `json:"id"`
				OwnerId UserID `json:"owner_id"`
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
			Owner: w.Response.Items[i].OwnerId,
			ID:    w.Response.Items[i].Id,
			Date:  w.Response.Items[i].Date,
			Text:  w.Response.Items[i].Text,
		})
	}
	return
}
