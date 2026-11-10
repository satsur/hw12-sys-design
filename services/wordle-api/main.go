package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var answer string
var requestSlots = make(chan struct{}, 4)
var processingDelay = 50 * time.Millisecond

type ResponseBody struct {
	Results []string
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		word := r.URL.Query().Get("word")

		if word == "" {
			http.Error(w, `"word" parameter not supplied.`, http.StatusBadRequest)
			return
		}

		result, err := ProcessWord(word, answer)
		resp := ResponseBody{
			Results: result,
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		requestSlots <- struct{}{}
		defer func() { <-requestSlots }()
		time.Sleep(processingDelay)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func init() {
	fmt.Println("Starting Wordle...")

	// load answer into memory on startu
	target := os.Getenv("WORDLE_ANSWER")
	if target == "" {
		fmt.Println("Could not get saved Wordle answer, falling back to default...")
		target = "WORLD" // fallback
	}
	answer = strings.ToUpper(target)

	fmt.Println("Today's Wordle Answer: " + answer)
}

func main() {
	configureCapacity()
	port := os.Getenv("WORDLE_PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Println("Now listening on port " + port)
	http.HandleFunc("/wordle", handleRequest)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func configureCapacity() {
	maxInFlight := environmentInt("WORDLE_MAX_IN_FLIGHT", 4)
	delayMS := environmentInt("WORDLE_DELAY_MS", 50)
	requestSlots = make(chan struct{}, maxInFlight)
	processingDelay = time.Duration(delayMS) * time.Millisecond
	fmt.Printf("Capacity configured: %d concurrent requests, %d ms processing delay\n", maxInFlight, delayMS)
}

func environmentInt(name string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(name))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}
