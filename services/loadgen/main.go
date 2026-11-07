package loadgen

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync/atomic"
	"time"
)

const baseUrl = "/wordle"
const maxRPS = 100       // requests per second
var defaultRPS = 0       // requests per second
var defaultDuration = 10 // seconds

func sendRequest(word string) (*http.Response, error) {
	u, err := url.Parse(baseUrl)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("word", word)
	u.RawQuery = q.Encode()

	return http.Get(u.String())
}

// newRate in requests per second
func updateRate(newRate int) {
	if newRate >= 0 {
		defaultRPS = newRate
	}
}

func updateDuration(newDuration int) {
	if newDuration >= 0 {
		defaultDuration = newDuration
	}
}

// Request accepts two query params
// 'rps' = rate per second of load test
// 'duration' = duration in seconds of the test
func start(w http.ResponseWriter, req *http.Request) {
	word := "hello"

	rpsParam := req.URL.Query().Get("rps")
	durationParam := req.URL.Query().Get("duration")

	rps := defaultRPS
	if rpsParam != "" {
		intRPS, err := strconv.Atoi(rpsParam)
		if err == nil {
			rps = intRPS
		}
	}

	duration := defaultDuration
	if durationParam != "" {
		intDuration, err := strconv.Atoi(durationParam)
		if err == nil {
			duration = intDuration
		}
	}

	ticker := time.NewTicker()
	timeout := time.After(time.Duration(duration) * time.Second)
	var successCount int64

	for {
		select {
		case <-timeout:
			fmt.Printf("Test finished. %d requests sent in %.2f seconds", successCount, duration.Seconds())
			return
		case <-ticker.C:
			go func() {
				resp, err := sendRequest(word)
				if err != nil {
					fmt.Printf("Encountered an error: %s", err.Error())
					return
				}
				defer resp.Body.Close() // Prevents memory and/or file descriptor leaks

				if resp.StatusCode == http.StatusOK {
					atomic.AddInt64(&successCount, 1)
				}
			}()
		}
	}
}

func main() {
	port := "8080"
	http.HandleFunc("/start", start)
	log.Fatal(http.ListenAndServe(":"+port, nil))

}
