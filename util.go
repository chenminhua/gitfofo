package main

import (
	"fmt"
	"sync"

	termtables "github.com/brettski/go-termtables"
	"github.com/chenminhua/gitfofo/types"
)

type RWMutexMap struct {
	data map[string]int
	mu   sync.RWMutex
}

func (m *RWMutexMap) Get(key string) (value int, ok bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, ok = m.data[key]
	return value, ok
}

func (m *RWMutexMap) Inc(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if value, ok := m.data[key]; ok {
		m.data[key] = value + 1
	} else {
		m.data[key] = 1
	}
}

func (m *RWMutexMap) Data() map[string]int {
	return m.data
}

func NewRWMutexMap() RWMutexMap {
	m := RWMutexMap{}
	m.data = make(map[string]int)
	return m
}

func StringLimitLen(str string, length int) string {
	if len(str) <= length {
		return str
	}
	return str[:length]
}

func PrintUserTable(users []*types.User) {
	if len(users) == 0 {
		return
	}
	ut := termtables.CreateTable()
	ut.AddHeaders("name", "url", "bio", "location", "follower", "following", "repos")
	for _, user := range users {
		ut.AddRow(user.Login, user.HTMLURL, StringLimitLen(user.Bio, 30), user.Location, user.Followers, user.Following, user.PublicRepos)
	}
	fmt.Println(ut.Render())
}
