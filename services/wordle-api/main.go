package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"strings"
	"time"
)

var answer string

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

		simulateLatency()
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func simulateLatency() {
	ms := 10 + rand.IntN(90)
	time.Sleep(time.Duration(ms) * time.Millisecond)
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
	port := os.Getenv("WORDLE_PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Println("Now listening on port " + port)
	http.HandleFunc("/wordle", handleRequest)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
