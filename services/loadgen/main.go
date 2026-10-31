package loadgen

import (
	"fmt"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"
)

const baseUrl = "/wordle"
const maxRPS = 100    // requests per second
var targetRPS = 0     // requests per second
var loadDuration = 10 // seconds

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
		targetRPS = newRate
	}
}

func updateDuration(newDuration int) {
	if newDuration >= 0 {
		loadDuration = newDuration
	}
}

func main() {
	word := "hello"

	duration := time.Duration(loadDuration) * time.Second
	ticker := time.NewTicker(time.Second / time.Duration(targetRPS))
	timeout := time.After(duration)
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
