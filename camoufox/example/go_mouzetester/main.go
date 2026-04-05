/*
Go replica of mouzeteseter.py — locator-based mouse humanization example.

Demonstrates Camoufox's humanize feature which intercepts mouse movements
and replaces them with natural, human-like curves.
*/
package main

import (
	"fmt"
	"log"

	camoufox "github.com/MUHVUGGAH/zorroz/camoufox/golanglib/camoufox"
	"github.com/playwright-community/playwright-go"
)

func main() {
	// Enable humanize with default settings (pass true).
	// You can also pass a float64 for max duration in seconds, e.g. 2.0
	headless := false
	opts := camoufox.LaunchOptionsConfig{
		Headless: &headless,
		Humanize: true,
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

	if _, err := page.Goto("https://camoufox.com/tests/buttonclick"); err != nil {
		log.Fatal(err)
	}

	// Define the locator once. It will re-find the button every time you use it.
	button := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Click me!"})
	// Alternatively: button := page.Locator("button.button")

	for i := 0; i < 10; i++ {
		// 1. Playwright automatically waits for the button to be 'actionable'
		// 2. Camoufox intercepts the click and moves the mouse in a human curve
		if err := button.Click(); err != nil {
			log.Printf("click %d failed: %v", i+1, err)
			continue
		}
		fmt.Printf("Clicked %d times\n", i+1)

		// Small random delay to look more human between clicks
		page.WaitForTimeout(500)
	}
}
