package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"sync"

	"github.com/brettski/go-termtables"
	"github.com/chenminhua/gitfofo/github"
	"github.com/chenminhua/gitfofo/syncset"
	"golang.org/x/sync/errgroup"
)

// config holds command-line and environment settings.
type config struct {
	EntryUser   string
	Threshold   int
	Concurrency int
	Token       string
}

func main() {
	ctx := context.Background()

	cfg := loadConfig()
	gc := github.NewClient(cfg.Token)
	viewer := fetchViewer(ctx, gc)
	entry := determineEntryUser(ctx, gc, viewer, cfg.EntryUser)

	viewerFollowingSet, followsCh := fetchFirstDegreeFollowings(ctx, gc, viewer, entry)
	secondDegreeFollowings := fetchSecondDegreeFollowings(ctx, gc, followsCh, cfg.Concurrency)
	recUsers := recommandUsers(ctx, gc, viewerFollowingSet, secondDegreeFollowings, cfg.Threshold)

	printUserTable(recUsers)
}

// loadConfig parses flags and environment variables.
func loadConfig() *config {
	thr := flag.Int("threshold", 5, "shared follower threshold")
	entry := flag.String("entry", "", "entry username (default: yourself)")
	con := flag.Int("concurrency", 200, "parallel fetch count")
	flag.Parse()

	token := os.Getenv("git_token")
	if token == "" {
		log.Fatal("Please set 'git_token' environment variable")
	}

	return &config{
		EntryUser:   *entry,
		Threshold:   *thr,
		Concurrency: *con,
		Token:       token,
	}
}

// fetchViewer fetches and displays the current user.
func fetchViewer(ctx context.Context, client *github.Client) *github.User {
	fmt.Println("----------- get your info ------------")
	user, err := client.GetUser(ctx, "")
	if err != nil {
		log.Fatalf("get your info failed: %v", err)
	}
	printUserTable([]*github.User{user})
	return user
}

// determineEntryUser fetches the 'entry' user or returns the viewer.
func determineEntryUser(ctx context.Context, client *github.Client, viewer *github.User, entryName string) *github.User {
	if entryName == "" {
		return viewer
	}
	fmt.Printf("------------- get entry user %s info -----------------\n", entryName)
	entry, err := client.GetUser(ctx, entryName)
	if err != nil {
		log.Fatalf("get entry user %s failed: %v", entryName, err)
	}
	printUserTable([]*github.User{entry})
	return entry
}

// fetchFirstDegreeFollowings streams followings of the entry user and returns viewer's
// following set. The channel will be closed when all followings are collected and sent
// to the channel.
func fetchFirstDegreeFollowings(
	ctx context.Context,
	client *github.Client,
	viewer, entry *github.User,
) (*syncset.Set[string], chan *github.FollowingUser) {
	isSame := viewer == entry
	ch := make(chan *github.FollowingUser)
	vfs := syncset.New[string]()

	go func() {
		for u := range client.GetAllFollowings(ctx, viewer) {
			vfs.Add(u.Login)
			// viewer == entry, just use the following users here for downstream to collect
			// the second-degree followings.
			if isSame {
				ch <- u
			}
		}
		if !isSame {
			for u := range client.GetAllFollowings(ctx, entry) {
				ch <- u
			}
		}
		close(ch)
	}()
	return vfs, ch
}

// fetchSecondDegreeFollowings reads a stream of first-degree followers from ch
// and counts how many times each second-degree following appears. It launches
// 'concurrency' goroutines that, for each user in ch:
//  1. Fetch the user, who is followed by the entry user.
//  2. Skip users with more than 300 followings to limit workload.
//  3. Fetch all followings of that user, which is a second-degree following of
//     the entry user.
//
// Every second-degree following of the entry user will be counted and recorded
// in a map before being returned.
func fetchSecondDegreeFollowings(
	ctx context.Context,
	client *github.Client,
	ch chan *github.FollowingUser,
	concurrency int,
) map[string]int {
	eg, _ := errgroup.WithContext(ctx)
	mu := sync.Mutex{}
	counter := make(map[string]int)

	for i := 0; i < concurrency; i++ {
		eg.Go(func() error {
			for u := range ch {
				fu, err := client.GetUser(ctx, u.Login)
				if err != nil {
					return fmt.Errorf("failed to get user %s: %w", u.Login, err)
				}
				if fu.Following > 300 {
					continue
				}
				for u2 := range client.GetAllFollowings(ctx, fu) {
					mu.Lock()
					counter[u2.Login]++
					mu.Unlock()
				}
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		log.Fatalf("worker error: %v", err)
	}
	return counter
}

// recommandUsers fetches detailed user info for candidates.
func recommandUsers(
	ctx context.Context,
	ghClient *github.Client,
	viewerFollowings *syncset.Set[string],
	secondDegreeFollowings map[string]int,
	threshold int,
) []*github.User {
	var wg sync.WaitGroup
	var mu sync.Mutex
	recs := make([]*github.User, 0, len(secondDegreeFollowings))

	for username, count := range secondDegreeFollowings {
		if count <= threshold || viewerFollowings.Contains(username) {
			continue
		}
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			ru, err := ghClient.GetUser(ctx, u)
			if err != nil {
				log.Printf("get user %s failed: %v", u, err)
				return
			}
			mu.Lock()
			recs = append(recs, ru)
			mu.Unlock()
		}(username)
	}
	wg.Wait()

	// Sort recommendations by follower count descending
	sort.Slice(recs, func(i, j int) bool {
		return recs[i].Followers > recs[j].Followers
	})

	return recs
}

// printUserTable renders users as a terminal table.
func printUserTable(users []*github.User) {
	if len(users) == 0 {
		return
	}
	ut := termtables.CreateTable()
	ut.AddHeaders("name", "url", "bio", "location", "followers", "following", "repos")
	for _, u := range users {
		ut.AddRow(
			u.Login,
			u.HTMLURL,
			stringLimitLen(u.Bio, 30),
			u.Location,
			u.Followers,
			u.Following,
			u.PublicRepos,
		)
	}
	fmt.Println(ut.Render())
}

// stringLimitLen truncates a string to a maximum length.
func stringLimitLen(str string, length int) string {
	if len(str) <= length {
		return str
	}
	return str[:length]
}
