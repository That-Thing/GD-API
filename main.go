package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"that-thing/gundeals-api/handlers" // Import handlers

	"github.com/gocolly/colly/v2"
)

func main() {
	// Command-line flags
	host := flag.String("host", "localhost", "Server host address")
	port := flag.String("port", "8080", "Server port")
	flag.Parse()

	// Collector
	setupCollector()

	// Register routes
	handlers.SetupDealsRoutes()
	handlers.SetupCouponRoutes()

	// Handle /
	// TODO - Write a better handler for /
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Gun Tuah. Sale on that thang."))
	})

	addr := fmt.Sprintf("%s:%s", *host, *port)
	log.Printf("Server starting on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// === Global Collector Setup ===

func setupCollector() {
	collector := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"),
		colly.AllowURLRevisit(),
		colly.MaxDepth(1),
		colly.Async(true),
	)

	collector.Limit(&colly.LimitRule{
		DomainGlob: "*",
	})

	collector.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
		r.Headers.Set("Referer", "https://www.google.com/")
		r.Headers.Set("Accept-Language", "en-US,en;q=0.9")
		r.Headers.Set("Sec-Ch-Ua", "\"Chromium\";v=\"122\", \"Not(A:Brand\";v=\"24\", \"Google Chrome\";v=\"122\"")
		r.Headers.Set("Sec-Ch-Ua-Mobile", "?0")
		r.Headers.Set("Sec-Ch-Ua-Platform", "\"Windows\"")
		r.Headers.Set("Sec-Fetch-Dest", "document")
		r.Headers.Set("Sec-Fetch-Mode", "navigate")
		r.Headers.Set("Sec-Fetch-Site", "none")
		r.Headers.Set("Sec-Fetch-User", "?1")
		r.Headers.Set("Upgrade-Insecure-Requests", "1")
		r.Headers.Set("Cache-Control", "max-age=0")
		r.Headers.Set("Connection", "keep-alive")
	})

	// Share the collector with the handlers package
	handlers.SharedCollector = collector
}
