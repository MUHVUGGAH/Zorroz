/*
Async (concurrent) example — Go equivalent of example/async_example.py.

Uses goroutines for concurrent page scraping, which is Go's native
equivalent of Python's asyncio.
*/
package main

import (
	"fmt"
	"log"
	"sync"

	camoufox "github.com/MUHVUGGAH/zorroz/camoufox/golanglib/camoufox"
	"github.com/playwright-community/playwright-go"
)

var urls = []string{
	"https://httpbin.org/headers",
	"https://httpbin.org/user-agent",
	"https://httpbin.org/ip",
}

type scrapeResult struct {
	URL  string
	Body string
}

func scrape(page playwright.Page, url string) scrapeResult {
	if _, err := page.Goto(url); err != nil {
		return scrapeResult{URL: url, Body: fmt.Sprintf("error: %v", err)}
	}
	body, err := page.Locator("body").InnerText()
	if err != nil {
		return scrapeResult{URL: url, Body: fmt.Sprintf("error: %v", err)}
	}
	if len(body) > 300 {
		body = body[:300]
	}
	return scrapeResult{URL: url, Body: body}
}

func main() {
	headless := true
	opts := camoufox.LaunchOptionsConfig{
		Headless: &headless,
	}

	ac, err := camoufox.NewAsyncCamoufox(opts)
	if err != nil {
		log.Fatal(err)
	}
	defer ac.Close()

	context, err := ac.Browser().NewContext()
	if err != nil {
		log.Fatal(err)
	}

	// Create pages
	pages := make([]playwright.Page, len(urls))
	for i := range urls {
		p, err := context.NewPage()
		if err != nil {
			log.Fatal(err)
		}
		pages[i] = p
	}

	// Scrape concurrently using goroutines
	results := make([]scrapeResult, len(urls))
	var wg sync.WaitGroup
	for i, url := range urls {
		wg.Add(1)
		go func(idx int, u string) {
			defer wg.Done()
			results[idx] = scrape(pages[idx], u)
		}(i, url)
	}
	wg.Wait()

	for _, r := range results {
		fmt.Printf("\n--- %s ---\n", r.URL)
		fmt.Println(r.Body)
	}
}
