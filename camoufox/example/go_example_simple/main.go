/*
Simple example — Go equivalent of example/example_simple.py.
*/
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	camoufox "github.com/MUHVUGGAH/zorroz/camoufox/golanglib/camoufox"
	"github.com/playwright-community/playwright-go"
)

const acceptEncoding = "identity"

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

	page, err := cf.Browser.NewPage(playwright.BrowserNewPageOptions{
		ExtraHttpHeaders: map[string]string{"accept-encoding": acceptEncoding},
	})
	if err != nil {
		log.Fatal(err)
	}

	if _, err := page.Goto("https://abrahamjuliot.github.io/creepjs/"); err != nil {
		log.Fatal(err)
	}

	fmt.Print("Press Enter to close...")
	bufio.NewReader(os.Stdin).ReadLine()
}
