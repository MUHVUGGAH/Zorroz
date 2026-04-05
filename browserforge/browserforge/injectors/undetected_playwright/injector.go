// Package undetected_playwright is a 1:1 copy of the playwright injector,
// using separate interfaces for undetected-playwright typing purposes.
package undetected_playwright

import (
	"browserforge/fingerprints"
	"browserforge/injectors"
)

// UndetectedPlaywrightBrowser is an interface for Undetected-Playwright browser implementations.
type UndetectedPlaywrightBrowser interface {
	NewContext(options BrowserNewContextOptions) (BrowserContext, error)
	BrowserTypeName() string
}

// BrowserContext is an interface for Undetected-Playwright browser context implementations.
type BrowserContext interface {
	SetExtraHTTPHeaders(headers map[string]string) error
	AddInitScript(script string) error
	OnPage(handler func(Page))
}

// Page is an interface for Undetected-Playwright page implementations.
type Page interface {
	EmulateMedia(colorScheme string) error
}

// BrowserNewContextOptions holds options for creating a new browser context.
type BrowserNewContextOptions struct {
	UserAgent        string
	ColorScheme      string
	Viewport         Viewport
	ExtraHTTPHeaders map[string]string
}

// Viewport represents browser viewport dimensions.
type Viewport struct {
	Width  int
	Height int
}

// NewContext injects an Undetected-Playwright context with a Fingerprint.
// This is the Go equivalent of both AsyncNewContext and NewContext from the Python version.
func NewContext(
	browser UndetectedPlaywrightBrowser,
	fp *fingerprints.Fingerprint,
	generator *fingerprints.FingerprintGenerator,
	genOpts *fingerprints.GenerateOptions,
	utilsJSPath string,
) (BrowserContext, error) {
	var err error
	fp, err = injectors.GenerateFingerprint(fp, generator, genOpts)
	if err != nil {
		return nil, err
	}

	utilsJS, err := injectors.UtilsJS(utilsJSPath)
	if err != nil {
		return nil, err
	}

	function, err := injectors.InjectFunction(fp, utilsJS)
	if err != nil {
		return nil, err
	}

	opts := ContextOptions(fp, nil)
	context, err := browser.NewContext(opts)
	if err != nil {
		return nil, err
	}

	// Set headers
	err = context.SetExtraHTTPHeaders(
		injectors.OnlyInjectableHeaders(fp.Headers, browser.BrowserTypeName()),
	)
	if err != nil {
		return nil, err
	}

	// Dark mode
	context.OnPage(func(page Page) {
		page.EmulateMedia("dark")
	})

	// Inject function
	err = context.AddInitScript(function)
	if err != nil {
		return nil, err
	}

	return context, nil
}

// AsyncNewContext is an alias for NewContext in Go, since Go handles concurrency
// differently from Python's async/await pattern.
var AsyncNewContext = NewContext

// ContextOptions builds options for a new browser context from a fingerprint.
func ContextOptions(fp *fingerprints.Fingerprint, extraOptions map[string]interface{}) BrowserNewContextOptions {
	viewport := Viewport{
		Width:  fp.Screen.Width,
		Height: fp.Screen.Height,
	}

	extraHTTPHeaders := map[string]string{
		"accept-language": fp.Headers["Accept-Language"],
	}

	return BrowserNewContextOptions{
		UserAgent:        fp.Navigator.UserAgent,
		ColorScheme:      "dark",
		Viewport:         viewport,
		ExtraHTTPHeaders: extraHTTPHeaders,
	}
}
