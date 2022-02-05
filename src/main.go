package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type Handler func(http.ResponseWriter, *http.Request)
type Any interface{}

const (
	SERVER_HOST = "localhost"
	SERVER_PORT = 8000
)

func handlerMiddleware(handler func(*VKClient, *http.Request) Any) Handler {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println(*r)
		client := VKClient{
			AccessToken: os.Getenv("VK_ACCESS_TOKEN"),
		}
		start := time.Now()
		response := handler(&client, r)
		log.Printf("Time elapsed %v", time.Since(start))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func main() {
	if _, presented := os.LookupEnv("VK_ACCESS_TOKEN"); !presented {
		panic("VK_ACCESS_TOKEN was not found in env vars")
	}
	// TODO: fix
	http.HandleFunc("/reposts", handlerMiddleware(
		func(client *VKClient, r *http.Request) Any {
			return getRepostersByPostUrl(client, r.FormValue("postUrl"))
		},
	))

	// TODO: sqlite like query, filter by date range, reversed flag, in text, etc.
	// TODO: search in different groups, profiles
	// https://vk.com/app3876642
	// https://vk.com/wall-2158488_651604
	http.HandleFunc("/rev_posts", handlerMiddleware(
		func(client *VKClient, r *http.Request) Any {
			res, err := getReversedPosts(client, r.FormValue("groupUrl"))
			if err != nil {
				return err
			}
			return res
		},
	))

	http.HandleFunc("/intersection", handlerMiddleware(
		func(client *VKClient, r *http.Request) Any {
			return getIntersection(client, json.NewDecoder(r.Body))
		},
	))

	serverAddress := fmt.Sprintf("%s:%d", SERVER_HOST, SERVER_PORT)
	log.Printf("Starting server on http://%s\n", serverAddress)
	log.Fatal(http.ListenAndServe(serverAddress, nil))
}
