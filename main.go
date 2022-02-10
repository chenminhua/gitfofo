package main

import (
	"github.com/chenminhua/gitfofo/types"
	"sync"
	"sync/atomic"
	"time"
)

var followCount int32
var fofomap = NewRWMutexMap()
var followingMap = NewRWMutexMap() // viewer的following
var fofochan = make(chan *types.FollowingUser)

func getFofo(username string, page int) {
	for _, u := range getFollowing(username, page) {
		fofomap.Inc(u.Login)
	}
}

func initGetFofoWorker() {
	// 10个并发线程拉follower数据
	for i := 0; i < 10; i++ {
		go func() {
			for {
				u, more := <-fofochan
				if more {
					fu := getUser(u.Login)
					atomic.AddInt32(&followCount, 1)
					for i := 1; (i-1)*30 < fu.Following; i++ {
						go func(page int) {
							getFofo(fu.Login, page)
						}(i)
					}
				} else {
					break
				}

			}
		}()
	}
}

func getViewerAndEntryFollowing() {
	for i := 1; (i-1)*30 < config.Viewer.Following; i++ {
		go func(page int) {
			for _, u := range getFollowing(config.Viewer.Login, page) {
				followingMap.Inc(u.Login)
				if config.IsViewerEqualsToEntry() {
					fofochan <- u
				}
			}
		}(i)
	}

	if !config.IsViewerEqualsToEntry() {
		for i := 1; (i-1)*30 < config.EntryUser.Following; i++ {
			go func(page int) {
				for _, u := range getFollowing(config.EntryUser.Login, page) {
					fofochan <- u
				}
			}(i)
		}
	}
}

func main() {
	LoadConfig()
	initGetFofoWorker()
	getViewerAndEntryFollowing()
	for {
		// 当全部收集完成后，打印数据。
		if followCount == int32(config.EntryUser.Following) {
			time.Sleep(2 * time.Second)
			fmap := followingMap.Data()
			var wg sync.WaitGroup
			var recommendFofo []*types.User

			// 拿所有满足条件的fofo的详细信息。条件为：超过ShareFollowerThreshold个共同关注，我还没有关注
			for k, v := range fofomap.Data() {
				if v > config.ShareFollowerThreshold {
					if _, ok := fmap[k]; !ok {
						wg.Add(1)
						go func(username string) {
							defer wg.Done()
							recommendFofo = append(recommendFofo, getUser(username))
						}(k)
					}
				}
			}
			wg.Wait()
			PrintUserTable(recommendFofo)
			return
		}
	}
}
