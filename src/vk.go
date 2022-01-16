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

func (client *VKClient) apiRequest(method string, params url.Values) (res []byte, err error) {
	const (
		TOO_MANY_REQUESTS = 6
		ACCESS_DENIED     = 15
		USER_BANNED       = 18
		USER_HIDDEN       = 30
	)
	url := fmt.Sprintf("https://api.vk.com/method/%s", method)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	req_params := req.URL.Query()
	req_params.Add("v", "5.131")
	req_params.Add("access_token", client.AccessToken)
	for k, v := range params {
		req_params.Add(k, v[0])
	}
	req.URL.RawQuery = req_params.Encode()
	timeLimitTries := 0
	TIME_LIMIT_TRIES_LIMIT := 100
	var resp *http.Response
	var body []byte
	for timeLimitTries < TIME_LIMIT_TRIES_LIMIT {
		log.Printf("%s(%v)", method, params)
		resp, err = client.Client.Do(req)
		if err != nil {
			// if user hid their wall
			return
		}
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		var v struct {
			Error struct {
				Code    int    `json:"error_code"`
				Message string `json:"error_msg"`
			} `json:"error"`
		}
		err = json.Unmarshal(body, &v)
		if err != nil {
			return
		}
		if v.Error.Code != 0 {
			// TODO: define behavior on error (retry or throw error) in loop
			if v.Error.Code == TOO_MANY_REQUESTS {
				time.Sleep(time.Millisecond * 500)
				continue
			} else if v.Error.Code == ACCESS_DENIED || v.Error.Code == USER_BANNED || v.Error.Code == USER_HIDDEN {
				err = fmt.Errorf("Error(%d) %s", v.Error.Code, v.Error.Message)
				return
			}
			err = fmt.Errorf("%s(%v) = Error(%d) %s", method, params, v.Error.Code, v.Error.Message)
			return
		}
		res = body
		return
	}
	err = fmt.Errorf("%s(%v) = Timeout", method, params)
	return
}
