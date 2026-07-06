package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

var (
	targetURL = flag.String(
		"target-url",
		"https://hosted-routinator.rarc.net/json",
		"The URL to fetch ROA data from")
	listenAddr = flag.String(
		"listen_addr",
		"127.0.0.1",
		"Address upon which to listen for connections.")
	port = flag.Int(
		"port",
		8080,
		"The port to listen on")
	timeout = flag.Duration(
		"timeout",
		30*time.Second,
		"Timeout for fetching data from target URL")
)

func handleROA(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	log.Printf("Received request from %s", r.RemoteAddr)

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Printf("Method not allowed: %s", r.Method)
		return
	}

	client := http.Client{
		Timeout: *timeout,
	}

	resp, err := client.Get(*targetURL)
	if err != nil {
		log.Printf("Error fetching from target %s: %v", *targetURL, err)
		http.Error(w, fmt.Sprintf("Failed to fetch ROA data: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Copy relevant headers
	if contentType := resp.Header.Get("Content-Type"); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}

	w.WriteHeader(resp.StatusCode)

	bytesCopied, err := io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("Error copying response body: %v", err)
		return
	}

	log.Printf("Successfully responded to %s: status %d, %d bytes, took %v", r.RemoteAddr, resp.StatusCode, bytesCopied, time.Since(start))
}

func main() {
	flag.Parse()

	http.HandleFunc("/", handleROA)

	addr := fmt.Sprintf("%s:%d", *listenAddr, *port)
	log.Printf("Starting ROA proxy on %s, proxying to %s", addr, *targetURL)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
