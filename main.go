package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/go-redis/redis"
)

func main() {
	done := make(chan string)

	go sendData(done)
	go rcvData()

	ctr := 0
	for range done {
		ctr++
	}
	fmt.Printf("%v requests completed\n", ctr)
}

func rcvData() {
	client := newRedisClient()
	for {
		time.Sleep(time.Duration(1 * time.Second))
		scores, _ := getScores(client)
		fmt.Printf("%s\n", scores)
	}
}

func sendData(done chan string) {
	client := newRedisClient()

	requests := 20000
	// Send requests
	for i := 0; i < requests; i++ {
		// Introduce some random latency to make things interesting
		latency := time.Duration(rand.Intn(5)) * time.Millisecond
		time.Sleep(latency)
		go addScore(client, done)
	}

}

func addScore(client *redis.Client, done chan string) error {
	score := float64(rand.Intn(1000))
	user := "user-" + strconv.Itoa(rand.Intn(1000))
	//fmt.Printf("%s\n", user)
	value := redis.Z{
		Score:  score,
		Member: user,
	}
	state, err := client.ZAdd("leaderboard", value).Result()
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	switch state {
	case 1:
		status := fmt.Sprintf("User %s added to the leaderboard.\n", user)
		done <- status
	case 0:
		status := fmt.Sprintf("User %s updated on the leaderboard.\n", user)
		done <- status
	}
	return nil
}

func getScores(client *redis.Client) ([]string, error) {
	scores, err := client.ZRevRange("leaderboard", 0, 9).Result()
	if err != nil {
		return nil, err
	}
	return scores, nil
}

// newRedisClient returns a redis client
func newRedisClient() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	return client
}
