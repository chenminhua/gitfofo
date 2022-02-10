package main

import (
	"encoding/json"
	"fmt"
	"github.com/chenminhua/gitfofo/types"
	"io/ioutil"
	"net/http"
	"os"
)

// userName为""，则拿用户自己信息
func getUser(userName string) *types.User {
	var url string
	if userName == "" {
		url = "https://api.github.com/user"
	} else {
		url = fmt.Sprintf("https://api.github.com/users/%s", userName)
	}
	body, err := httpQuery(url)
	var user types.User
	err = json.Unmarshal(body, &user)
	if err != nil {
		fmt.Println(err)
	}
	return &user
}

func httpQuery(query string) ([]byte, error) {
	req, _ := http.NewRequest("GET", query, nil)
	req.Header.Set("Authorization", fmt.Sprintf("token %s", config.Token))
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, err
	}
	if resp.Status == "403 Forbidden" {
		println("403, maybe you have hit the ratelimit, read this: https://docs.github.com/en/rest/overview/resources-in-the-rest-api#rate-limiting") //
		os.Exit(1)
	}
	body, err := ioutil.ReadAll(resp.Body)
	return body, err
}

func getFollowing(username string, page int) []*types.FollowingUser {
	url := fmt.Sprintf("https://api.github.com/users/%s/following?page=%d", username, page)
	body, err := httpQuery(url)
	var users []*types.FollowingUser
	err = json.Unmarshal(body, &users)
	if err != nil {
		fmt.Printf("get user %s error: %s\n", username, err)
	}
	return users
}
