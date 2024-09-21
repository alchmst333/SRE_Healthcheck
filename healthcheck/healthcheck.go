package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

// Configuration struct to hold endpoint details
type Configuration struct {
	Name    string            `yaml:"name"`
	Url     string            `yaml:"url"`
	Method  string            `yaml:"method,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty"`
}

// Availability struct to track UP and DOWN counts and latency metrics
type Availability struct {
	SuccessCount int
	FailureCount int
	TotalLatency time.Duration
	MinLatency   time.Duration
	MaxLatency   time.Duration
}

// Function to extract domain from URL
func extractDomain(rawUrl string) string {
	parsedUrl, err := url.Parse(rawUrl)
	if err != nil {
		log.Printf("Invalid URL '%s': %v", rawUrl, err)
		return "invalid_domain"
	}
	return parsedUrl.Host
}

// Function to get file data from a given file path
func GetFileDataFromFlag(filePath string) []byte {
	// Read the file data
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Failed to read file '%s': %v", filePath, err)
	}
	return data
}

// Function to parse YAML contents into a slice of Configuration
func parser(data []byte) []Configuration {
	var requests []Configuration

	// Unmarshal YAML data into the requests slice
	err := yaml.Unmarshal(data, &requests)
	if err != nil {
		log.Fatalf("Error parsing YAML: %v", err)
	}

	return requests
}

// Function to check endpoint health with latency metrics
func checkEndpointHealth(req Configuration, avail *Availability, latencyThreshold time.Duration) {
	// Set default method to GET if not specified
	method := req.Method
	if method == "" {
		method = "GET"
	}

	// Create HTTP request
	httpReq, err := http.NewRequest(method, req.Url, nil)
	if err != nil {
		log.Printf("Error creating request for %s: %v", req.Url, err)
		avail.FailureCount++
		return
	}

	// Add headers if any
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Initialize HTTP client with timeout
	client := &http.Client{
		Timeout: 1 * time.Second, // Adjust as needed
	}

	// Measure latency
	startTime := time.Now()
	resp, err := client.Do(httpReq)
	latency := time.Since(startTime)

	if err != nil {
		log.Printf("DOWN: %s (%s) - Error: %v", req.Name, req.Url, err)
		log.Println("Error occurred, check your connection or the target URL.")
		avail.FailureCount++
		return
	}
	defer resp.Body.Close()

	// Determine UP or DOWN
	if resp.StatusCode >= 200 && resp.StatusCode < 300 && latency < latencyThreshold {
		log.Printf("UP: %s (%s) - Status: %d, Latency: %v", req.Name, req.Url, resp.StatusCode, latency)
		avail.SuccessCount++
		avail.TotalLatency += latency

		// Update MinLatency
		if avail.MinLatency == 0 || latency < avail.MinLatency {
			avail.MinLatency = latency
		}
		// Update MaxLatency
		if latency > avail.MaxLatency {
			avail.MaxLatency = latency
		}
	} else {
		log.Printf("DOWN: %s (%s) - Status: %d, Latency: %v", req.Name, req.Url, resp.StatusCode, latency)
		avail.FailureCount++
	}
}

// Function to log availability percentages and detailed metrics per URL
func logAvailability(requests []Configuration, availability map[string]*Availability) {
	// Iterate over each request (each endpoint)
	for _, req := range requests {
		stats := availability[req.Url] // Keyed by full URL

		total := stats.SuccessCount + stats.FailureCount
		if total == 0 {
			fmt.Printf("%s (%s) has no availability data yet.\n", req.Name, req.Url)
			continue
		}

		percentage := (float64(stats.SuccessCount) / float64(total)) * 100
		percentage = float64(int(percentage + 0.5)) // Round to nearest whole number

		// Print the availability percentage and detailed metrics per URL
		fmt.Printf("%s (%s) has %d%% availability percentage\n", req.Name, req.Url, int(percentage))
		fmt.Printf("   Total Checks: %d\n", total)
		fmt.Printf("   Successful Checks: %d\n", stats.SuccessCount)
		fmt.Printf("   Failed Checks: %d\n", stats.FailureCount)
		if stats.SuccessCount > 0 {
			fmt.Printf("   Average Latency: %v\n", time.Duration(int64(stats.TotalLatency)/int64(stats.SuccessCount)))
		} else {
			fmt.Printf("   Average Latency: N/A\n")
		}
		if stats.MinLatency > 0 {
			fmt.Printf("   Minimum Latency: %v\n", stats.MinLatency)
		}
		if stats.MaxLatency > 0 {
			fmt.Printf("   Maximum Latency: %v\n", stats.MaxLatency)
		}
	}
	fmt.Println()
}

// Logger function to set up logging to a file
func logger(logFilePath string) (*os.File, error) {
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file '%s': %v", logFilePath, err)
	}

	log.SetOutput(file)
	log.SetFlags(log.LstdFlags | log.Lshortfile) // Includes date, time, and file info
	return file, nil
}

func main() {
	// Define all command-line flags at the beginning
	configFilePath := flag.String("file", "./sample.yml", "Path to the YAML configuration file")
	logFilePath := flag.String("log", "./healthcheck.log", "Path to the log file")
	checkInterval := flag.Duration("interval", 15*time.Second, "Health check interval (e.g., 15s, 1m)")
	latencyThreshold := flag.Duration("latency", 500*time.Millisecond, "Latency threshold for UP status (e.g., 500ms, 1s)")
	flag.Parse()

	// Validate that the config file path is provided
	if *configFilePath == "" {
		fmt.Println("Error: Configuration file path must be provided using the --file flag.")
		flag.Usage()
		os.Exit(1)
	}

	// Initialize logger
	logFile, err := logger(*logFilePath)
	if err != nil {
		fmt.Printf("Error initializing logger: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()

	// Retrieve and parse the YAML configuration
	yamlData := GetFileDataFromFlag(*configFilePath)
	requests := parser(yamlData)

	// Log the domains and URLs being monitored
	log.Println("Domains and URLs being monitored:")
	for _, req := range requests {
		domain := extractDomain(req.Url)
		log.Printf("- Domain: %s, URL: %s", domain, req.Url)
	}
	log.Println()

	// Initialize availability tracking per URL
	availability := make(map[string]*Availability)
	for _, req := range requests {
		if _, exists := availability[req.Url]; !exists {
			availability[req.Url] = &Availability{}
		}
	}

	// Handle graceful termination
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Create a ticker to run the checks at the specified interval
	ticker := time.NewTicker(*checkInterval)
	defer ticker.Stop()

	// Initial health check before entering the loop
	log.Println("Starting initial health check...")
	var wg sync.WaitGroup
	wg.Add(len(requests))
	for _, req := range requests {
		go func(r Configuration) {
			defer wg.Done()
			checkEndpointHealth(r, availability[r.Url], *latencyThreshold)
		}(req)
	}
	wg.Wait()
	logAvailability(requests, availability)

	// Infinite loop to keep checking the endpoints at the specified interval
	for {
		select {
		case <-ticker.C:
			log.Println("Starting new health check cycle...")
			var wg sync.WaitGroup
			wg.Add(len(requests))
			for _, req := range requests {
				go func(r Configuration) {
					defer wg.Done()
					checkEndpointHealth(r, availability[r.Url], *latencyThreshold)
				}(req)
			}
			wg.Wait() // Wait for all health checks to complete
			logAvailability(requests, availability) // Log after all checks
		case sig := <-sigs:
			log.Printf("Received signal %s. Exiting program.", sig)
			os.Exit(0)
		}
	}
	
}
