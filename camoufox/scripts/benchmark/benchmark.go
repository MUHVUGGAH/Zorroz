// benchmark measures browser memory usage across different modes and URLs.
//
// Usage (from the scripts/benchmark/ directory):
//
//	go run . --mode headless --browser camoufox
//	go run . --mode headful --browser firefox
//	go run . --mode headless --browser camoufox-ubo
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	camoufox "github.com/MUHVUGGAH/zorroz/camoufox/golanglib/camoufox"
	"github.com/playwright-community/playwright-go"
)

var benchmarkURLs = []string{
	"about:blank",
	"https://google.com",
	"https://yahoo.com",
}

// getProcessMemoryMB returns total RSS in MB for all processes with the given name.
func getProcessMemoryMB(name string) float64 {
	out, err := exec.Command("ps", "-C", name, "-o", "rss=").Output()
	if err != nil {
		return 0
	}
	var totalKB int64
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		kb, err := strconv.ParseInt(line, 10, 64)
		if err == nil {
			totalKB += kb
		}
	}
	return float64(totalKB) / 1024.0
}

// getAverageMemory samples memory over duration seconds and returns the average.
func getAverageMemory(processName string, durationSec int) float64 {
	var total float64
	for i := 0; i < durationSec; i++ {
		total += getProcessMemoryMB(processName)
		time.Sleep(1 * time.Second)
	}
	if durationSec == 0 {
		return 0
	}
	return total / float64(durationSec)
}

type result struct {
	URL      string
	MemoryMB float64
}

func runBenchmark(mode, browserName string) ([]result, error) {
	headless := mode == "headless"

	// Virtual display env for headful mode on Linux
	var envVars map[string]string
	if !headless {
		virt := camoufox.NewVirtualDisplay(false)
		display, err := virt.Get()
		if err != nil {
			return nil, fmt.Errorf("virtual display: %w", err)
		}
		envVars = map[string]string{"DISPLAY": display}
	}

	var browser playwright.Browser
	var pw *playwright.Playwright
	var cfox *camoufox.Camoufox

	switch browserName {
	case "camoufox-ubo":
		// Use the full Camoufox API (includes uBlock Origin)
		var err error
		launchOpts := playwright.BrowserTypeLaunchOptions{
			Headless: playwright.Bool(headless),
		}
		if envVars != nil {
			launchOpts.Env = envVars
		}
		opts := camoufox.NewBrowserOptions{
			LaunchOptions: &launchOpts,
		}
		cfox, err = camoufox.NewCamoufox(opts)
		if err != nil {
			return nil, fmt.Errorf("launching camoufox-ubo: %w", err)
		}
		browser = cfox.Browser

	case "camoufox":
		// Launch camoufox binary via raw Playwright (no addons)
		var err error
		pw, err = playwright.Run()
		if err != nil {
			return nil, fmt.Errorf("starting playwright: %w", err)
		}
		launchPath, err := camoufox.LaunchPath("")
		if err != nil {
			return nil, fmt.Errorf("getting launch path: %w", err)
		}
		launchOpts := playwright.BrowserTypeLaunchOptions{
			Headless:       playwright.Bool(headless),
			ExecutablePath: playwright.String(launchPath),
		}
		if envVars != nil {
			launchOpts.Env = envVars
		}
		browser, err = pw.Firefox.Launch(launchOpts)
		if err != nil {
			return nil, fmt.Errorf("launching camoufox: %w", err)
		}

	case "firefox":
		// Launch stock Firefox via Playwright
		var err error
		pw, err = playwright.Run()
		if err != nil {
			return nil, fmt.Errorf("starting playwright: %w", err)
		}
		launchOpts := playwright.BrowserTypeLaunchOptions{
			Headless: playwright.Bool(headless),
		}
		if envVars != nil {
			launchOpts.Env = envVars
		}
		browser, err = pw.Firefox.Launch(launchOpts)
		if err != nil {
			return nil, fmt.Errorf("launching firefox: %w", err)
		}

	default:
		return nil, fmt.Errorf("unknown browser: %s", browserName)
	}

	// Determine process name for memory sampling
	processName := "firefox"
	if strings.HasPrefix(browserName, "camoufox") {
		processName = "camoufox-bin"
	}

	var results []result
	for _, url := range benchmarkURLs {
		page, err := browser.NewPage()
		if err != nil {
			return nil, fmt.Errorf("new page: %w", err)
		}
		if _, err := page.Goto(url); err != nil && url != "about:blank" {
			fmt.Fprintf(os.Stderr, "Warning: goto %s: %v\n", url, err)
		}
		time.Sleep(5 * time.Second) // Allow page to fully load

		mem := getAverageMemory(processName, 10)
		results = append(results, result{URL: url, MemoryMB: mem})

		page.Close()
	}

	browser.Close()
	if cfox != nil {
		cfox.Close()
	}
	if pw != nil {
		pw.Stop()
	}

	return results, nil
}

func printTable(browserName string, results []result) {
	fmt.Printf("\n=== MEMORY RESULTS FOR %s ===\n", strings.ToUpper(browserName))

	// Calculate column widths
	urlWidth := 3 // "URL"
	for _, r := range results {
		if len(r.URL) > urlWidth {
			urlWidth = len(r.URL)
		}
	}
	memWidth := 18 // "Memory Usage (MB)"

	// Print header
	sep := "+" + strings.Repeat("-", urlWidth+2) + "+" + strings.Repeat("-", memWidth+2) + "+"
	fmt.Println(sep)
	fmt.Printf("| %-*s | %-*s |\n", urlWidth, "URL", memWidth, "Memory Usage (MB)")
	fmt.Println(sep)

	// Print rows
	for _, r := range results {
		fmt.Printf("| %-*s | %*.2f |\n", urlWidth, r.URL, memWidth, r.MemoryMB)
	}
	fmt.Println(sep)
}

func main() {
	mode := flag.String("mode", "", "Mode to run the browser in (headless or headful)")
	browserName := flag.String("browser", "", "Browser to benchmark (firefox, camoufox, camoufox-ubo)")
	flag.Parse()

	if *mode != "headless" && *mode != "headful" {
		fmt.Fprintln(os.Stderr, "Error: --mode must be 'headless' or 'headful'")
		flag.Usage()
		os.Exit(1)
	}
	if *browserName != "firefox" && *browserName != "camoufox" && *browserName != "camoufox-ubo" {
		fmt.Fprintln(os.Stderr, "Error: --browser must be 'firefox', 'camoufox', or 'camoufox-ubo'")
		flag.Usage()
		os.Exit(1)
	}

	results, err := runBenchmark(*mode, *browserName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Benchmark error: %v\n", err)
		os.Exit(1)
	}

	printTable(*browserName, results)
}
