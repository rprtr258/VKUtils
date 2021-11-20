package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

func postRepostsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println(*r)
	client := VKClient{
		AccessToken: os.Getenv("VK_ACCESS_TOKEN"),
	}
	start := time.Now()
	response := getReposters(&client, r.FormValue("postUrl"))
	log.Printf("Time elapsed %v", time.Since(start))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	if _, presented := os.LookupEnv("VK_ACCESS_TOKEN"); !presented {
		panic("VK_ACCESS_TOKEN was not found in env vars")
	}
	log.Println("Server started successfully on http://localhost:8000")
	http.HandleFunc("/reposts", postRepostsHandler)
	log.Fatal(http.ListenAndServe("localhost:8000", nil))
}
