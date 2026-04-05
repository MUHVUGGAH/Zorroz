/*
Quick example to test camoufox Go port.

This is the Go equivalent of example/example.py.

Prerequisites:
  - Ensure camoufox binary is fetched (e.g. via the Python CLI: python -m camoufox fetch)
  - Set BROWSERFORGE_DATA_DIR if BrowserForge data files are not auto-detected
*/
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	camoufox "github.com/MUHVUGGAH/zorroz/camoufox/golanglib/camoufox"
)

func main() {
	headless := false
	opts := camoufox.LaunchOptionsConfig{
		Headless: &headless,
	}

	launchResult, err := camoufox.LaunchOptions(opts)
	if err != nil {
		log.Fatal(err)
	}

	cf, err := camoufox.NewCamoufox(camoufox.NewBrowserOptions{
		LaunchOptions: launchResult.ToPlaywrightLaunchOptionsPtr(),
	})
	if err != nil {
		log.Fatal(err)
	}
	defer cf.Close()

	page, err := cf.Browser.NewPage()
	if err != nil {
		log.Fatal(err)
	}

	// Visit a fingerprint test page
	if _, err := page.Goto("https://abrahamjuliot.github.io/creepjs/"); err != nil {
		log.Fatal(err)
	}
	page.WaitForLoadState()

	title, err := page.Title()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Page title: %s\n", title)

	// Grab the trust score CreepJS assigns (using Locator API)
	gradeLocator := page.Locator("#creep-results .grade")
	count, _ := gradeLocator.Count()
	if count > 0 {
		text, err := gradeLocator.InnerText()
		if err == nil {
			fmt.Printf("CreepJS trust grade: %s\n", text)
		}
	} else {
		fmt.Println("Score element not found — page may still be loading.")
	}

	// Print the spoofed user-agent the browser reported
	ua, err := page.Evaluate("navigator.userAgent")
	if err == nil {
		fmt.Printf("User-Agent: %v\n", ua)
	}

	fmt.Print("\nPress Enter to close the browser...")
	bufio.NewReader(os.Stdin).ReadLine()
}
