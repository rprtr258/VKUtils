package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func handler(w http.ResponseWriter, r *http.Request) {
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
		panic(fmt.Sprintf("%s was not found in env vars", "VK_ACCESS_TOKEN"))
	}
	log.Println("Server started successfully on http://localhost:8000")
	http.HandleFunc("/reposts", handler) // each request calls handler
	log.Fatal(http.ListenAndServe("localhost:8000", nil))
}
