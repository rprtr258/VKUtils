package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

type UserID int

type UserList struct {
	Response struct {
		Count uint     `json:"count"`
		Items []UserID `json:"items"`
	} `json:"response"`
}

type VKClient struct {
	AccessToken string
	Client      http.Client
}

func (client *VKClient) apiRequest(method string, params url.Values) []byte {
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
	return body
}
