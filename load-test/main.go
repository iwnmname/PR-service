package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/fatih/color"
	"golang.org/x/time/rate"
)

const (
	baseURL         = "http://localhost:8080"
	targetRPS       = 5
	testDuration    = 60 * time.Second
	sliResponseTime = 300 * time.Millisecond
	sliSuccessRate  = 99.9
)

type Result struct {
	StatusCode   int
	ResponseTime time.Duration
	Error        error
}

type Stats struct {
	mu              sync.Mutex
	results         []Result
	totalRequests   int
	successRequests int
	errorRequests   int
}

func (s *Stats) Add(r Result) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.results = append(s.results, r)
	s.totalRequests++

	if r.Error != nil {
		s.errorRequests++
	} else if r.StatusCode >= 200 && r.StatusCode < 400 {
		s.successRequests++
	}
}

func (s *Stats) Print() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.results) == 0 {
		fmt.Println("No results to display")
		return
	}

	var responseTimes []float64
	statusCodes := make(map[int]int)

	for _, r := range s.results {
		if r.Error == nil {
			responseTimes = append(responseTimes, float64(r.ResponseTime.Milliseconds()))
			statusCodes[r.StatusCode]++
		}
	}

	sort.Float64s(responseTimes)

	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	fmt.Println("\n" + bold("═══════════════════════════════════════════"))
	fmt.Println(bold("        LOAD TEST RESULTS"))
	fmt.Println(bold("═══════════════════════════════════════════"))

	fmt.Printf("\n%s\n", cyan("Overall Statistics:"))
	fmt.Printf("  Total Requests:      %d\n", s.totalRequests)
	fmt.Printf("  Successful:          %d (%.2f%%)\n", s.successRequests, float64(s.successRequests)/float64(s.totalRequests)*100)
	fmt.Printf("  Errors:              %d (%.2f%%)\n", s.errorRequests, float64(s.errorRequests)/float64(s.totalRequests)*100)

	if len(responseTimes) > 0 {
		fmt.Printf("\n%s\n", cyan("Response Times:"))
		minTime := responseTimes[0]
		maxTime := responseTimes[len(responseTimes)-1]
		avg := average(responseTimes)
		p50 := percentile(responseTimes, 50)
		p95 := percentile(responseTimes, 95)
		p99 := percentile(responseTimes, 99)

		fmt.Printf("  Min:                 %.0fms\n", minTime)
		fmt.Printf("  Max:                 %.0fms\n", maxTime)
		fmt.Printf("  Avg:                 %.0fms\n", avg)
		fmt.Printf("  p50 (median):        %.0fms\n", p50)
		fmt.Printf("  p95:                 %.0fms\n", p95)
		fmt.Printf("  p99:                 %.0fms", p99)

		if p99 <= float64(sliResponseTime.Milliseconds()) {
			fmt.Printf(" %s\n", green("✅ < 300ms"))
		} else {
			fmt.Printf(" %s\n", red("❌ > 300ms"))
		}
	}

	fmt.Printf("\n%s\n", cyan("Status Codes:"))
	for code := 200; code < 600; code++ {
		if count, ok := statusCodes[code]; ok {
			percent := float64(count) / float64(s.totalRequests) * 100
			status := ""
			if code >= 200 && code < 300 {
				status = green(fmt.Sprintf("%d", code))
			} else if code >= 400 && code < 500 {
				status = yellow(fmt.Sprintf("%d", code))
			} else {
				status = red(fmt.Sprintf("%d", code))
			}
			fmt.Printf("  %s:              %d (%.2f%%)\n", status, count, percent)
		}
	}

	successRate := float64(s.successRequests) / float64(s.totalRequests) * 100
	fmt.Printf("\n%s\n", cyan("SLI Requirements:"))
	fmt.Printf("  Success Rate:        %.2f%%", successRate)
	if successRate >= sliSuccessRate {
		fmt.Printf(" %s\n", green("✅ >= 99.9%"))
	} else {
		fmt.Printf(" %s\n", red("❌ < 99.9%"))
	}

	p99 := percentile(responseTimes, 99)
	fmt.Printf("  p99 Response Time:   %.0fms", p99)
	if p99 <= float64(sliResponseTime.Milliseconds()) {
		fmt.Printf(" %s\n", green("✅ < 300ms"))
	} else {
		fmt.Printf(" %s\n", red("❌ > 300ms"))
	}

	allPassed := successRate >= sliSuccessRate && p99 <= float64(sliResponseTime.Milliseconds())
	fmt.Println("\n" + bold("═══════════════════════════════════════════"))
	if allPassed {
		fmt.Printf("  %s\n", green(bold("✅ ALL SLI REQUIREMENTS MET!")))
	} else {
		fmt.Printf("  %s\n", red(bold("❌ SOME SLI REQUIREMENTS FAILED")))
	}
	fmt.Println(bold("═══════════════════════════════════════════") + "\n")
}

func average(data []float64) float64 {
	sum := 0.0
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data))
}

func percentile(data []float64, p float64) float64 {
	if len(data) == 0 {
		return 0
	}
	index := (p / 100.0) * float64(len(data)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return data[lower]
	}

	return data[lower]*(float64(upper)-index) + data[upper]*(index-float64(lower))
}

func makeRequest(method, path string, body interface{}) Result {
	start := time.Now()

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return Result{Error: err}
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, baseURL+path, reqBody)
	if err != nil {
		return Result{Error: err}
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return Result{Error: err, ResponseTime: time.Since(start)}
	}
	defer resp.Body.Close()

	io.ReadAll(resp.Body)

	return Result{
		StatusCode:   resp.StatusCode,
		ResponseTime: time.Since(start),
	}
}

func setupTestData() error {
	fmt.Println("Setting up test data...")

	for i := 1; i <= 10; i++ {
		members := make([]map[string]interface{}, 0)
		for j := 1; j <= 20; j++ {
			userID := fmt.Sprintf("load-u%d-%d", i, j)
			members = append(members, map[string]interface{}{
				"user_id":   userID,
				"username":  fmt.Sprintf("User%d-%d", i, j),
				"is_active": true,
			})
		}

		payload := map[string]interface{}{
			"team_name": fmt.Sprintf("load-team-%d", i),
			"members":   members,
		}

		result := makeRequest("POST", "/team/add", payload)
		if result.Error != nil || (result.StatusCode != 201 && result.StatusCode != 400) {
			return fmt.Errorf("failed to create team: %v", result.Error)
		}
	}

	fmt.Println("Test data ready!")
	return nil
}

func randomEndpoint() (string, string, interface{}) {
	endpoints := []struct {
		method string
		path   string
		body   interface{}
	}{
		{"GET", "/statistics", nil},
		{"GET", fmt.Sprintf("/team/get?team_name=load-team-%d", rand.Intn(10)+1), nil},
		{"GET", fmt.Sprintf("/users/getReview?user_id=load-u%d-%d", rand.Intn(10)+1, rand.Intn(20)+1), nil},
		{
			"POST",
			"/pullRequest/create",
			map[string]interface{}{
				"pull_request_id":   fmt.Sprintf("load-pr-%d", rand.Intn(100000)),
				"pull_request_name": "Load test PR",
				"author_id":         fmt.Sprintf("load-u%d-%d", rand.Intn(10)+1, rand.Intn(20)+1),
			},
		},
	}

	selected := endpoints[rand.Intn(len(endpoints))]
	return selected.method, selected.path, selected.body
}

func runLoadTest(ctx context.Context, stats *Stats) {
	limiter := rate.NewLimiter(rate.Limit(targetRPS), 1)

	fmt.Printf("Starting load test: %d RPS for %v\n", targetRPS, testDuration)
	fmt.Println("Press Ctrl+C to stop early...")

	requestCount := 0

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := limiter.Wait(ctx); err != nil {
				return
			}

			method, path, body := randomEndpoint()
			result := makeRequest(method, path, body)
			stats.Add(result)

			requestCount++
			if requestCount%50 == 0 {
				fmt.Printf("Sent %d requests...\n", requestCount)
			}
		}
	}
}

func main() {
	fmt.Println("╔════════════════════════════════════════════════╗")
	fmt.Println("║   PR Reviewer Service - Load Testing Tool      ║")
	fmt.Println("╚════════════════════════════════════════════════╝")
	fmt.Println()

	if err := setupTestData(); err != nil {
		fmt.Printf("Error setting up test data: %v\n", err)
		fmt.Println("Make sure the service is running on http://localhost:8080")
		return
	}

	stats := &Stats{}

	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()

	startTime := time.Now()
	runLoadTest(ctx, stats)
	duration := time.Since(startTime)

	fmt.Printf("\nTest completed in %v\n", duration)

	stats.Print()
}