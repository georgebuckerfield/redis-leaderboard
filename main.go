package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis"
)

const (
	// The number of records to retrieve from the leaderboard
	topn = 5000
)

var totalLatency time.Duration

func main() {
	done := make(chan bool)
	stop := make(chan bool)

	start := time.Now()

	requests, _ := strconv.Atoi(os.Getenv("TOTALREQUESTS"))
	fmt.Printf("Preparing to send %v requests\n", requests)

	go sendData(requests, done)
	go rcvData(stop)

	for {
		_, y := <-done
		if !y {
			// The done channel has been closed by the sendData routine,
			// so send a stop signal to the rcvData routine
			stop <- true
			_, y := <-stop
			if !y {
				// The stop channel has been closed by the rcvData routine,
				// so we can now exit
				elapsed := time.Since(start)
				fmt.Printf("\n%v requests completed in %v\n", requests, elapsed)
				fmt.Printf("%v of artificial latency\n", totalLatency)
				return
			}
		}
	}
}

// Simulate leaderboard results being viewed
func rcvData(stop chan bool) {
	client := newRedisClient()
	for {
		select {
		case <-stop:
			// Recieved a stop signal, close the channel and exit
			close(stop)
			return
		default:
			// No stop signal received, get the top scores
			time.Sleep(time.Duration(1 * time.Second))
			start := time.Now()
			_, err := getScores(client)
			elapsed := time.Since(start)
			if err != nil {
				return
			}
			count, err := getCount(client)
			if err != nil {
				return
			}
			fmt.Printf("Got top %v scores in %v\n", topn, elapsed)
			fmt.Printf("%v scores in the leaderboard\n", count)
		}
	}
}

// Launch goroutines to insert data into the leaderboard
func sendData(requests int, done chan bool) {
	client := newRedisClient()
	// Channel for individual goroutines to signal they're finished on
	tasks := make(chan string)
	users, _ := strconv.Atoi(os.Getenv("MAXUSERS"))
	fmt.Printf("Simulating max %v users\n", users)
	// Send requests
	for i := 0; i < requests; i++ {
		// Introduce some random latency to make things interesting
		latency := time.Duration(rand.Intn(5)) * time.Millisecond
		totalLatency = totalLatency + latency
		user := "user-" + strconv.Itoa(rand.Intn(users))
		time.Sleep(latency)
		go addScore(client, user, tasks)
	}
	ctr := 0
	for range tasks {
		// Listen for completed addScore tasks and keep a count
		ctr++
		if ctr == requests {
			//fmt.Printf("\n%v addScore tasks completed with %v of artificial latency\n", requests, totalLatency)
			close(done)
		}
	}

}

// Use this if we want to reset the score for the member each time
func setScore(client *redis.Client, user string, done chan string) error {
	score := float64(rand.Intn(1000))

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

// Use this if we want the score to be increased each time
func addScore(client *redis.Client, user string, done chan string) error {
	score := float64(rand.Intn(1000))

	_, err := client.ZIncrBy("leaderboard", score, user).Result()
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	done <- "Score added"

	return nil
}

func getScores(client *redis.Client) ([]string, error) {
	scores, err := client.ZRevRange("leaderboard", 0, topn-1).Result()
	if err != nil {
		return nil, err
	}
	return scores, nil
}

func getCount(client *redis.Client) (int64, error) {
	count, err := client.ZCount("leaderboard", "-inf", "+inf").Result()
	if err != nil {
		return 0, err
	}
	return count, nil
}

// newRedisClient returns a redis client
func newRedisClient() *redis.Client {
	host := os.Getenv("REDISHOST")
	if host == "" {
		host = "localhost"
	}
	address := fmt.Sprintf(host + ":6379")
	client := redis.NewClient(&redis.Options{
		Addr:     address,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	return client
}
