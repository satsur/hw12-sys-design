package main

import (
	"encoding/json"
	"log"
	"net/http"
)

const answer = "SLATE"

type ResponseBody struct {
	Results []string
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		word := r.URL.Query().Get("word")

		result, _ := ProcessWord(word, answer)
		resp := ResponseBody{
			Results: result,
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
		return
	}
}

func main() {
	http.HandleFunc("/wordle", handleRequest)
	log.Fatal(http.ListenAndServe(":5000", nil))
}
