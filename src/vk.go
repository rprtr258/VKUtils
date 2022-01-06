package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

type UserID int

type UserList struct {
	Response struct {
		Count uint     `json:"count"`
		Items []UserID `json:"items"`
	} `json:"response"`
}

type Post struct {
	Owner UserID `json:"owner_id"`
	ID    uint   `json:"id"`
	Date  uint   `json:"date"`
	Text  string `json:"text"`
}

type VKClient struct {
	AccessToken string
	Client      http.Client
}

func (client *VKClient) apiRequest(method string, params url.Values) []byte {
	const TOO_MANY_REQUESTS = 6
	url := fmt.Sprintf("https://api.vk.com/method/%s", method)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	req_params := req.URL.Query()
	req_params.Add("v", "5.131")
	req_params.Add("access_token", client.AccessToken)
	for k, v := range params {
		req_params.Add(k, v[0])
	}
	req.URL.RawQuery = req_params.Encode()
	resp, err := client.Client.Do(req)
	if err != nil {
		// if user hid their wall
		log.Fatal(err)
		return []byte{}
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	var v struct {
		Error struct {
			Code    int    `json:"error_code"`
			Message string `json:"error_msg"`
		} `json:"error"`
	}
	json.Unmarshal(body, &v)
	if v.Error.Code != 0 {
		log.Printf("[ERROR] While doing request %s %v: (%d) %s", method, params, v.Error.Code, v.Error.Message)
		// TODO: define behavior on error (retry or throw error) in loop
		if v.Error.Code == TOO_MANY_REQUESTS {
			time.Sleep(time.Millisecond * 500)
			return client.apiRequest(method, params)
		}
		return []byte{}
	}
	return body
}
