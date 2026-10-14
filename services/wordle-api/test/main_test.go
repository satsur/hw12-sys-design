package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func WordleGET(t *testing.T) {
	request, _ := http.NewRequest(http.MethodGet, "/wordle?", nil)
	response := httptest.NewRecorder()

	t.

}
