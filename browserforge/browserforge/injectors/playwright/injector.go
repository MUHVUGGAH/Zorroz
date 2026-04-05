package playwright

import (
	"browserforge/fingerprints"
	"browserforge/injectors"
)

// PlaywrightBrowser is an interface for Playwright browser implementations.
type PlaywrightBrowser interface {
	NewContext(options BrowserNewContextOptions) (BrowserContext, error)
	BrowserTypeName() string
}

// BrowserContext is an interface for Playwright browser context implementations.
type BrowserContext interface {
	SetExtraHTTPHeaders(headers map[string]string) error
	AddInitScript(script string) error
	OnPage(handler func(Page))
}

// Page is an interface for Playwright page implementations.
type Page interface {
	EmulateMedia(colorScheme string) error
}

// BrowserNewContextOptions holds options for creating a new browser context.
type BrowserNewContextOptions struct {
	UserAgent         string
	ColorScheme       string
	Viewport          Viewport
	ExtraHTTPHeaders  map[string]string
	DeviceScaleFactor float64
}

// Viewport represents browser viewport dimensions.
type Viewport struct {
	Width  int
	Height int
}

// NewContext injects a Playwright context with a Fingerprint.
// This is the Go equivalent of both AsyncNewContext and NewContext from the Python version,
// since Go does not distinguish between sync and async at the function signature level.
func NewContext(
	browser PlaywrightBrowser,
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
		UserAgent:         fp.Navigator.UserAgent,
		ColorScheme:       "dark",
		Viewport:          viewport,
		ExtraHTTPHeaders:  extraHTTPHeaders,
		DeviceScaleFactor: fp.Screen.DevicePixelRatio,
	}
}
