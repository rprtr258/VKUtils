package vkutils

// type UserSet f.Set[UserID]

// func getFollowers(client *VKClient, userId UserID) (<-chan UserID, <-chan error) {
// 	return client.getUserList("users.getFollowers", url.Values{
// 		"user_id": []string{fmt.Sprint(userId)},
// 	}, 1000)
// }

// func contains(userId UserID, m UserSet) bool {
// 	present, contains := m[userId]
// 	return present && contains
// }

// func intersectChans(chans []<-chan UserID) UserSet {
// 	res := make(UserSet)
// 	chansCount := len(chans)
// 	if chansCount == 0 {
// 		return res
// 	}
// 	for userID := range chans[0] {
// 		res[userID] = true
// 	}
// 	for i := 1; i < chansCount; i++ {
// 		for userID := range chans[i] {
// 			if !contains(userID, res) {
// 				res[userID] = false
// 			}
// 		}
// 	}
// 	return res
// }

// func GetIntersection(client *VKClient, data *json.Decoder) []UserID {
// 	type GroupSet struct {
// 		GroupId UserID `json:"group_id"`
// 	}

// 	type PostSet struct {
// 		OwnerId UserID `json:"owner_id"`
// 		PostId  uint   `json:"post_id"`
// 	}

// 	type ProfileSet struct {
// 		UserId UserID `json:"user_id"`
// 	}

// 	type UserSets struct {
// 		GroupMembers []GroupSet   `json:"group_members"`
// 		Friends      []ProfileSet `json:"friends"`
// 		Followers    []ProfileSet `json:"followers"`
// 		Likers       []PostSet    `json:"likers"`
// 		Sharers      []PostSet    `json:"sharers"` // TODO: check inexactly
// 		// TODO: user provided
// 	}

// 	var v struct {
// 		Include UserSets `json:"include"`
// 		Exclude UserSets `json:"exclude"`
// 	}
// 	data.Decode(&v)

// 	// TODO: rewrite to channels
// 	// TODO: parallelize
//	toExclude := make(UserSet)
//	inserters := make(
//		[]func() (<-chan UserID, <-chan error),
//		len(v.Exclude.GroupMembers)+
//			len(v.Include.Friends)+
//			len(v.Include.Followers)+
//			len(v.Exclude.Likers)+
//			len(v.Exclude.Likers),
// 	)
// 	for _, groupSet := range v.Exclude.GroupMembers {
// 		inserters = append(inserters, func() (<-chan UserID, <-chan error) { return client.getGroupMembers(groupSet.GroupId) })
// 	}
// 	for _, profileSet := range v.Exclude.Friends {
// 		inserters = append(inserters, func() (<-chan UserID, <-chan error) { return client.getFriends(profileSet.UserId) })
// 	}
// 	for _, profileSet := range v.Exclude.Followers {
// 		inserters = append(inserters, func() (<-chan UserID, <-chan error) { return getFollowers(client, profileSet.UserId) })
// 	}
// 	for _, postSet := range v.Exclude.Likers {
// 		inserters = append(inserters, func() (<-chan UserID, <-chan error) { return client.getLikes(postSet.OwnerId, postSet.PostId) })
// 	}
// 	for _, postSet := range v.Exclude.Sharers {
// 		inserters = append(inserters, func() (<-chan UserID, <-chan error) { return getSharers(client, postSet.OwnerId, postSet.PostId) })
// 	}
// 	errors := make(chan error)
// 	for _, f := range inserters {
// 		userIDs, errorsInserting := f()
// 		drainErrorChan(errors, errorsInserting)
// 		for userID := range userIDs {
// 			toExclude[userID] = true
// 		}
// 	}

// 	groupMembersChans := make([]<-chan UserID, 0)
// 	for _, groupSet := range v.Include.GroupMembers {
// 		groupMembers, errorsGroupMembers := client.getGroupMembers(groupSet.GroupId)
// 		drainErrorChan(errors, errorsGroupMembers)
// 		groupMembersChans = append(groupMembersChans, groupMembers)
// 	}
// 	groupMembersIntersection := intersectChans(groupMembersChans)
// 	friendsChans := make([]<-chan UserID, 0)
// 	for _, profileSet := range v.Include.Friends {
// 		friendsChan, errorsFriends := client.getFriends(profileSet.UserId)
// 		drainErrorChan(errors, errorsFriends)
// 		friendsChans = append(friendsChans, friendsChan)
// 	}
// 	friendsIntersection := intersectChans(friendsChans)
// 	// followersChans := make([]<-chan UserID, 0)
// 	// for _, profileSet := range v.Include.Followers {
// 	// 	followersChans = append(followersChans, getFollowers(client, profileSet.UserId))
// 	// }
// 	// followersIntersection := intersectChans(followersChans)
// 	// likersChans := make([]<-chan UserID, 0)
// 	// for _, postSet := range v.Include.Likers {
// 	// 	likersChans = append(likersChans, client.getLikes(postSet.OwnerId, postSet.PostId))
// 	// }
// 	// likersIntersection := intersectChans(likersChans)
// 	// sharersChans := make([]<-chan UserID, 0)
// 	// for _, postSet := range v.Include.Sharers {
// 	// 	sharersChans = append(sharersChans, getSharers(client, postSet.OwnerId, postSet.PostId))
// 	// }
// 	// sharersIntersection := intersectChans(sharersChans)

// 	res := make([]UserID, 0)
// 	// TODO: differentiate between empty map and empty intersection
// 	// TODO: find most-intersected user ids?
// 	// TODO: iterate over something different, maybe change algo
// 	for userId := range groupMembersIntersection {
// 		if !contains(userId, toExclude) &&
// 			// TODO: look about algo higher
// 			// (len(groupMembersIntersection) == 0 || contains(userId, groupMembersIntersection)) &&
// 			(len(friendsIntersection) == 0 || contains(userId, friendsIntersection)) { // &&
// 			// (len(followersIntersection) == 0 || contains(userId, followersIntersection)) &&
// 			// (len(likersIntersection) == 0 || contains(userId, likersIntersection) &&
// 			// (len(sharersIntersection) == 0 || contains(userId, sharersIntersection))) {
// 			res = append(res, userId)
// 		}
// 	}
// 	return res
// }
