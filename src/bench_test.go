package main

import (
	"fmt"
	"os"
	"testing"
)

func areEqualRepostSearchResult(a, b RepostSearchResult) bool {
	if a.Likes != b.Likes {
		return false
	}
	if a.TotalReposts != b.TotalReposts {
		return false
	}
	if len(a.Reposters) != len(b.Reposters) {
		return false
	}
	for i := range a.Reposters {
		if a.Reposters[i] != b.Reposters[i] {
			return false
		}
	}
	return true
}

// TODO: fix/remove benchmark
// func BenchmarkSample(b *testing.B) {
// 	for i := 0; i < b.N; i++ {
// 		for _, postUrl := range []string{
// 			"https://vk.com/wall-149859311_975",
// 		} {
// 			client := VKClient{
// 				AccessToken: os.Getenv("VK_ACCESS_TOKEN"),
// 			}
// 			res := getReposters(&client, postUrl)
// 			expected := RepostSearchResult{
// 				Likes:        594,
// 				TotalReposts: 102,
// 				Reposters:    []UserID{478324650},
// 			}
// 			if !areEqualRepostSearchResult(res, expected) {
// 				panic(fmt.Sprintf("Got %v, Expected %v", res, expected))
// 			}
// 		}
// 	}
// }
