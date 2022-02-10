package main

import (
	"flag"
	"fmt"
	"github.com/chenminhua/gitfofo/types"
	"os"
)

type Config struct {
	Token                  string
	EntryUserName          string
	Viewer                 *types.User
	EntryUser              *types.User
	ShareFollowerThreshold int // 共同关注数阈值
}

func (c *Config) IsViewerEqualsToEntry() bool {
	return c.Viewer.Login == c.EntryUserName
}

var config = &Config{}

func LoadConfig() {
	shareFollowerThreshold := flag.Int("threshold", 5, "threshold of shared follower count")
	token := flag.String("token", "", "github personal access token")
	entryUserName := flag.String("entry", "", "gitfofo entry user, default is *you*")
	flag.Parse()
	config.ShareFollowerThreshold = *shareFollowerThreshold
	config.Token = loadToken(*token)
	config.EntryUserName = *entryUserName
	println("----------- get your info ------------")
	viewer := getUser("")
	if viewer == nil {
		println("-------------get your info failed---------------------")
		os.Exit(1)
	}
	config.Viewer = viewer

	if config.EntryUserName == "" {
		config.EntryUserName = viewer.Login
		config.EntryUser = viewer
	} else {
		fmt.Printf("-------------get entry user %s info---------------------\n", *entryUserName)
		entryUser := getUser(*entryUserName)
		config.EntryUser = entryUser
		if entryUser == nil {
			fmt.Printf("-------------get entry user %s failed---------------------\n", *entryUserName)
			os.Exit(1)
		}
	}
	PrintUserTable([]*types.User{config.Viewer, config.EntryUser})

}

func loadToken(token string) string {
	if token == "" {
		token = os.Getenv("git_token")
	}
	if token == "" {
		println("no valid token!!")
		os.Exit(1)
	}
	return token
}
