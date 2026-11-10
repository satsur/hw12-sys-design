package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultPort = "8080"
	maxRPS      = 1000
	maxDuration = 300
)

type testRequest struct {
	RPS      int    `json:"rps"`
	Duration int    `json:"duration"`
	Word     string `json:"word"`
}

type testResult struct {
	mutex          sync.RWMutex   `json:"-"`
	ID             string         `json:"id"`
	Status         string         `json:"status"`
	RequestedRPS   int            `json:"requested_rps"`
	Duration       int            `json:"duration_seconds"`
	Word           string         `json:"word"`
	StartedAt      time.Time      `json:"started_at"`
	FinishedAt     *time.Time     `json:"finished_at,omitempty"`
	RequestsSent   int            `json:"requests_sent"`
	Successes      int            `json:"successes"`
	Errors         int            `json:"errors"`
	StatusCodes    map[string]int `json:"status_codes"`
	MinLatencyMS   float64        `json:"min_latency_ms"`
	AverageLatency float64        `json:"average_latency_ms"`
	P50Latency     float64        `json:"p50_latency_ms"`
	P95Latency     float64        `json:"p95_latency_ms"`
	ErrorMessage   string         `json:"error_message,omitempty"`
}

type testStore struct {
	sync.RWMutex
	tests  map[string]*testResult
	active bool
}

var tests = &testStore{tests: make(map[string]*testResult)}
var targetURL = strings.TrimRight(os.Getenv("LOADGEN_TARGET_URL"), "/")

func main() {
	if targetURL == "" {
		targetURL = "http://nginx-service:80/wordle"
	}

	http.HandleFunc("/health", health)
	http.HandleFunc("/start", start)
	http.HandleFunc("/tests", listTests)
	http.HandleFunc("/tests/", getTest)

	port := os.Getenv("LOADGEN_PORT")
	if port == "" {
		port = defaultPort
	}
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
}

func health(response http.ResponseWriter, _ *http.Request) {
	writeJSON(response, http.StatusOK, map[string]string{"status": "ok"})
}

func start(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeJSON(response, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}

	var input testRequest
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		writeJSON(response, http.StatusBadRequest, map[string]string{"error": "request body must be valid JSON"})
		return
	}
	if input.RPS < 1 || input.RPS > maxRPS || input.Duration < 1 || input.Duration > maxDuration {
		writeJSON(response, http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("rps must be 1-%d and duration must be 1-%d seconds", maxRPS, maxDuration),
		})
		return
	}
	if input.Word == "" {
		input.Word = "hello"
	}

	tests.Lock()
	defer tests.Unlock()
	if tests.active {
		writeJSON(response, http.StatusConflict, map[string]string{"error": "another load test is already running"})
		return
	}

	testID := "test-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	result := &testResult{
		ID:           testID,
		Status:       "running",
		RequestedRPS: input.RPS,
		Duration:     input.Duration,
		Word:         input.Word,
		StartedAt:    time.Now().UTC(),
		StatusCodes:  make(map[string]int),
	}
	tests.tests[testID] = result
	tests.active = true
	go runTest(result)

	writeJSON(response, http.StatusAccepted, result)
}

func listTests(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeJSON(response, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}

	tests.RLock()
	defer tests.RUnlock()
	results := make([]*testResult, 0, len(tests.tests))
	for _, result := range tests.tests {
		results = append(results, snapshotResult(result))
	}
	sort.Slice(results, func(i, j int) bool { return results[i].StartedAt.Before(results[j].StartedAt) })
	writeJSON(response, http.StatusOK, results)
}

func getTest(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeJSON(response, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}

	testID := strings.TrimPrefix(request.URL.Path, "/tests/")
	tests.RLock()
	result, found := tests.tests[testID]
	if found {
		result = snapshotResult(result)
	}
	tests.RUnlock()
	if !found {
		writeJSON(response, http.StatusNotFound, map[string]string{"error": "load test not found"})
		return
	}
	writeJSON(response, http.StatusOK, result)
}

func runTest(result *testResult) {
	var waitGroup sync.WaitGroup
	latencies := make([]float64, 0, result.RequestedRPS*result.Duration)
	client := &http.Client{Timeout: 10 * time.Second}
	interval := time.Second / time.Duration(result.RequestedRPS)
	ticker := time.NewTicker(interval)
	deadline := time.NewTimer(time.Duration(result.Duration) * time.Second)

	for {
		select {
		case <-ticker.C:
			waitGroup.Add(1)
			go func() {
				defer waitGroup.Done()
				requestURL, err := url.Parse(targetURL)
				if err != nil {
					result.mutex.Lock()
					result.Errors++
					result.mutex.Unlock()
					return
				}
				query := requestURL.Query()
				query.Set("word", result.Word)
				requestURL.RawQuery = query.Encode()

				started := time.Now()
				response, err := client.Get(requestURL.String())
				latency := float64(time.Since(started).Microseconds()) / 1000
				result.mutex.Lock()
				result.RequestsSent++
				latencies = append(latencies, latency)
				if err != nil {
					result.Errors++
				} else {
					result.StatusCodes[strconv.Itoa(response.StatusCode)]++
					if response.StatusCode >= 200 && response.StatusCode < 300 {
						result.Successes++
					} else {
						result.Errors++
					}
					response.Body.Close()
				}
				result.mutex.Unlock()
			}()
		case <-deadline.C:
			ticker.Stop()
			waitGroup.Wait()
			finishTest(result, latencies)
			return
		}
	}
}

func finishTest(result *testResult, latencies []float64) {
	tests.Lock()
	defer tests.Unlock()
	result.mutex.Lock()
	defer result.mutex.Unlock()
	if len(latencies) > 0 {
		sort.Float64s(latencies)
		result.MinLatencyMS = round(latencies[0])
		result.AverageLatency = round(average(latencies))
		result.P50Latency = round(percentile(latencies, 0.50))
		result.P95Latency = round(percentile(latencies, 0.95))
	}
	now := time.Now().UTC()
	result.FinishedAt = &now
	result.Status = "completed"
	tests.active = false
}

func average(values []float64) float64 {
	var total float64
	for _, value := range values {
		total += value
	}
	return total / float64(len(values))
}

func percentile(values []float64, fraction float64) float64 {
	index := int(math.Ceil(fraction*float64(len(values)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(values) {
		index = len(values) - 1
	}
	return values[index]
}

func round(value float64) float64 {
	return math.Round(value*100) / 100
}

func cloneStatusCodes(values map[string]int) map[string]int {
	copy := make(map[string]int, len(values))
	for key, value := range values {
		copy[key] = value
	}
	return copy
}

func snapshotResult(result *testResult) *testResult {
	result.mutex.RLock()
	defer result.mutex.RUnlock()
	return &testResult{
		ID:             result.ID,
		Status:         result.Status,
		RequestedRPS:   result.RequestedRPS,
		Duration:       result.Duration,
		Word:           result.Word,
		StartedAt:      result.StartedAt,
		FinishedAt:     result.FinishedAt,
		RequestsSent:   result.RequestsSent,
		Successes:      result.Successes,
		Errors:         result.Errors,
		StatusCodes:    cloneStatusCodes(result.StatusCodes),
		MinLatencyMS:   result.MinLatencyMS,
		AverageLatency: result.AverageLatency,
		P50Latency:     result.P50Latency,
		P95Latency:     result.P95Latency,
		ErrorMessage:   result.ErrorMessage,
	}
}

func writeJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(value)
}
