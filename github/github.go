package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"net/http"
	"os"

	"golang.org/x/sync/errgroup"
)

type Client struct {
	Token string
}

func NewClient(token string) *Client {
	return &Client{
		Token: token,
	}
}

// GetUser call github api to get user info
// https://docs.github.com/en/rest/users/users?apiVersion=2022-11-28
func (c *Client) GetUser(ctx context.Context, userName string) (*User, error) {
	var url string
	if userName == "" {
		url = "https://api.github.com/user"
	} else {
		url = fmt.Sprintf("https://api.github.com/users/%s", userName)
	}
	body, err := c.httpQuery(ctx, url)
	if err != nil {
		return nil, err
	}
	var user User
	err = json.Unmarshal(body, &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *Client) httpQuery(ctx context.Context, query string) ([]byte, error) {
	req, _ := http.NewRequest("GET", query, nil)
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", fmt.Sprintf("token %s", c.Token))
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, err
	}
	if resp.Status == "403 Forbidden" {
		println("403, maybe you have hit the ratelimit, read this: https://docs.github.com/en/rest/overview/resources-in-the-rest-api#rate-limiting") //
		os.Exit(1)
	}
	body, err := io.ReadAll(resp.Body)
	return body, err
}

func (c *Client) GetAllFollowings(ctx context.Context, user *User) chan *FollowingUser {
	g, ctx := errgroup.WithContext(ctx)
	ch := make(chan *FollowingUser)

	for i := 1; (i-1)*30 < user.Following; i++ {
		page := i
		g.Go(func() error {
			users, err := c.getFollowingsByPage(ctx, user.Login, page)
			if err != nil {
				return err
			}
			for _, u := range users {
				ch <- u
			}
			return nil
		})
	}
	// close channel once all fetch goroutines complete
	go func() {
		if err := g.Wait(); err != nil {
			fmt.Printf("getAllFollowing error: %v\n", err)
		}
		close(ch)
	}()

	return ch
}

func (c *Client) getFollowingsByPage(ctx context.Context, username string, page int) ([]*FollowingUser, error) {
	url := fmt.Sprintf("https://api.github.com/users/%s/following?page=%d", username, page)
	body, err := c.httpQuery(ctx, url)
	if err != nil {
		return nil, err
	}
	var users []*FollowingUser
	if err := json.Unmarshal(body, &users); err != nil {
		return nil, err
	}
	return users, nil
}
