package main

import (
	"encoding/json"
	"fmt"
	"net/url"
)

type UserSet map[UserID]bool

func getFollowers(client *VKClient, userId UserID) <-chan UserID {
	return client.getUserList("users.getFollowers", url.Values{
		"user_id": []string{fmt.Sprint(userId)},
	}, 1000)
}

func contains(userId UserID, m UserSet) bool {
	present, contains := m[userId]
	return present && contains
}

func intersectChans(chans []<-chan UserID) UserSet {
	res := make(UserSet)
	chansCount := len(chans)
	if chansCount == 0 {
		return res
	}
	for userId := range chans[0] {
		res[userId] = true
	}
	for i := 1; i < chansCount; i++ {
		for userId := range chans[i] {
			if !contains(userId, res) {
				res[userId] = false
			}
		}
	}
	return res
}

func getIntersection(client *VKClient, data *json.Decoder) []UserID {
	type GroupSet struct {
		GroupId UserID `json:"group_id"`
	}

	type PostSet struct {
		OwnerId UserID `json:"owner_id"`
		PostId  uint   `json:"post_id"`
	}

	type ProfileSet struct {
		UserId UserID `json:"user_id"`
	}

	type UserSets struct {
		GroupMembers []GroupSet   `json:"group_members"`
		Friends      []ProfileSet `json:"friends"`
		Followers    []ProfileSet `json:"followers"`
		Likers       []PostSet    `json:"likers"`
		// Sharers      PostSet    `json:"sharers"` // TODO: implement
		// TODO: user provided
	}

	var v struct {
		Include UserSets `json:"include"`
		Exclude UserSets `json:"exclude"`
	}
	data.Decode(&v)

	// TODO: rewrite to channels
	// TODO: parallelize
	toExclude := make(UserSet)
	for _, groupSet := range v.Exclude.GroupMembers {
		for userId := range client.getGroupMembers(groupSet.GroupId) {
			toExclude[userId] = true
		}
	}
	for _, profileSet := range v.Exclude.Friends {
		for userId := range client.getFriends(profileSet.UserId) {
			toExclude[userId] = true
		}
	}
	for _, profileSet := range v.Exclude.Followers {
		for userId := range getFollowers(client, profileSet.UserId) {
			toExclude[userId] = true
		}
	}
	for _, postSet := range v.Exclude.Likers {
		for userId := range client.getLikes(postSet.OwnerId, postSet.PostId) {
			toExclude[userId] = true
		}
	}

	groupMembersChans := make([]<-chan UserID, 0)
	for _, groupSet := range v.Include.GroupMembers {
		groupMembersChans = append(groupMembersChans, client.getGroupMembers(groupSet.GroupId))
	}
	groupMembersIntersection := intersectChans(groupMembersChans)

	friendsChans := make([]<-chan UserID, 0)
	for _, profileSet := range v.Include.Friends {
		friendsChans = append(friendsChans, client.getFriends(profileSet.UserId))
	}
	friendsIntersection := intersectChans(friendsChans)

	followersChans := make([]<-chan UserID, 0)
	for _, profileSet := range v.Include.Followers {
		followersChans = append(followersChans, getFollowers(client, profileSet.UserId))
	}
	followersIntersection := intersectChans(followersChans)

	likersChans := make([]<-chan UserID, 0)
	for _, postSet := range v.Include.Likers {
		likersChans = append(likersChans, client.getLikes(postSet.OwnerId, postSet.PostId))
	}
	likersIntersection := intersectChans(likersChans)

	res := make([]UserID, 0)
	// TODO: differentiate between empty map and empty intersection
	// TODO: find most-intersected user ids?
	// TODO: iterate over something different, maybe change algo
	for userId := range groupMembersIntersection {
		if !contains(userId, toExclude) &&
			// TODO: look about algo higher
			// (len(groupMembersIntersection) == 0 || contains(userId, groupMembersIntersection)) &&
			(len(friendsIntersection) == 0 || contains(userId, friendsIntersection)) &&
			(len(followersIntersection) == 0 || contains(userId, followersIntersection)) &&
			(len(likersIntersection) == 0 || contains(userId, likersIntersection)) {
			res = append(res, userId)
		}
	}
	return res
}
