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

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func main() {
	http.HandleFunc("/wordle", handleRequest)
	log.Fatal(http.ListenAndServe(":5000", nil))
}
