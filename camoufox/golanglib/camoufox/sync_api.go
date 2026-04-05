package camoufox

import (
	"encoding/json"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

// Camoufox is the Go equivalent of the Python sync context manager.
// Go uses explicit construction and Close() instead of with-statements.
type Camoufox struct {
	Playwright *playwright.Playwright
	Browser    playwright.Browser
	Context    playwright.BrowserContext
}

// BrowserInstance represents either a launched browser or a persistent context.
type BrowserInstance struct {
	Browser playwright.Browser
	Context playwright.BrowserContext
}

// PersistentContextOptions wraps the user data dir required by Playwright Go.
type PersistentContextOptions struct {
	UserDataDir string
	Options     playwright.BrowserTypeLaunchPersistentContextOptions
}

// NewBrowserOptions holds the already-built Playwright launch options.
// The Python launch_options() port can feed these later.
// TODO: Port utils.py:launch_options into Go and populate these options directly.
type NewBrowserOptions struct {
	LaunchOptions            *playwright.BrowserTypeLaunchOptions
	PersistentContextOptions *PersistentContextOptions
}

// ContextFingerprint carries the per-context init script and context options.
type ContextFingerprint struct {
	InitScript     string
	ContextOptions playwright.BrowserNewContextOptions
}

// GenerateContextFingerprintOptions mirrors the Python generator inputs.
type GenerateContextFingerprintOptions struct {
	Preset    map[string]any
	OS        *string
	FFVersion *string
	WebRTCIP  *string
}

// GenerateContextFingerprintFunc is a hook for the future fingerprints.go port.
// TODO: Replace this hook with a real Go port of fingerprints.py when that file is replicated.
type GenerateContextFingerprintFunc func(options GenerateContextFingerprintOptions) (*ContextFingerprint, error)

// NewContextOptions mirrors the supported inputs from sync_api.py.
type NewContextOptions struct {
	Preset              map[string]any
	OS                  *string
	FFVersion           *string
	WebRTCIP            *string
	Proxy               *playwright.Proxy
	Geolocation         *playwright.Geolocation
	ContextOptions      playwright.BrowserNewContextOptions
	GenerateFingerprint GenerateContextFingerprintFunc
}

type proxyGeo struct {
	IP       *string
	Timezone *string
}

type proxyGeoResponse struct {
	Query    string `json:"query"`
	Timezone string `json:"timezone"`
}

// NewCamoufox starts Playwright and launches a browser or persistent context.
func NewCamoufox(options NewBrowserOptions) (*Camoufox, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, err
	}

	instance, err := NewBrowser(pw, options)
	if err != nil {
		_ = pw.Stop()
		return nil, err
	}

	return &Camoufox{
		Playwright: pw,
		Browser:    instance.Browser,
		Context:    instance.Context,
	}, nil
}

// Close closes the active browser/context and then stops Playwright.
func (c *Camoufox) Close() error {
	var closeErr error

	if c.Context != nil {
		closeErr = c.Context.Close()
	} else if c.Browser != nil {
		closeErr = c.Browser.Close()
	}

	if c.Playwright != nil {
		if err := c.Playwright.Stop(); closeErr == nil {
			closeErr = err
		}
	}

	return closeErr
}

// NewBrowser launches either a Browser or a persistent BrowserContext.
func NewBrowser(pw *playwright.Playwright, options NewBrowserOptions) (*BrowserInstance, error) {
	if options.PersistentContextOptions != nil {
		context, err := pw.Firefox.LaunchPersistentContext(
			options.PersistentContextOptions.UserDataDir,
			options.PersistentContextOptions.Options,
		)
		if err != nil {
			return nil, err
		}
		return &BrowserInstance{Context: context}, nil
	}

	launchOptions := playwright.BrowserTypeLaunchOptions{}
	if options.LaunchOptions != nil {
		launchOptions = *options.LaunchOptions
	}

	browser, err := pw.Firefox.Launch(launchOptions)
	if err != nil {
		return nil, err
	}

	return &BrowserInstance{Browser: browser}, nil
}

// Close closes the underlying browser or persistent context.
func (b *BrowserInstance) Close() error {
	if b == nil {
		return nil
	}
	if b.Context != nil {
		return b.Context.Close()
	}
	if b.Browser != nil {
		return b.Browser.Close()
	}
	return nil
}

// NewContext creates a new browser context and injects the generated init script.
func NewContext(browser playwright.Browser, options NewContextOptions) (playwright.BrowserContext, error) {
	webRTCIP := options.WebRTCIP

	if options.Proxy != nil && (webRTCIP == nil || options.ContextOptions.TimezoneId == nil) {
		geo := resolveProxyGeo(*options.Proxy)
		if webRTCIP == nil {
			webRTCIP = geo.IP
		}
		if options.ContextOptions.TimezoneId == nil && geo.Timezone != nil {
			options.ContextOptions.TimezoneId = geo.Timezone
		}
	}

	var fingerprint *ContextFingerprint
	var err error
	generateFingerprint := options.GenerateFingerprint
	if generateFingerprint == nil {
		generateFingerprint = GenerateContextFingerprint
	}
	if generateFingerprint != nil {
		fingerprint, err = generateFingerprint(GenerateContextFingerprintOptions{
			Preset:    options.Preset,
			OS:        options.OS,
			FFVersion: options.FFVersion,
			WebRTCIP:  webRTCIP,
		})
		if err != nil {
			return nil, err
		}
	}
	merged := playwright.BrowserNewContextOptions{}
	if fingerprint != nil {
		merged = fingerprint.ContextOptions
	}
	mergeContextOptions(&merged, options.ContextOptions)

	if options.Proxy != nil {
		merged.Proxy = options.Proxy
	}
	if options.Geolocation != nil {
		merged.Geolocation = options.Geolocation
		if !containsString(merged.Permissions, "geolocation") {
			merged.Permissions = append(merged.Permissions, "geolocation")
		}
	}

	context, err := browser.NewContext(merged)
	if err != nil {
		return nil, err
	}

	if fingerprint != nil && fingerprint.InitScript != "" {
		if err := context.AddInitScript(playwright.Script{
			Content: playwright.String(fingerprint.InitScript),
		}); err != nil {
			_ = context.Close()
			return nil, err
		}
	}

	return context, nil
}

func proxyURLWithCreds(proxy playwright.Proxy) string {
	server := strings.TrimSpace(proxy.Server)
	if server == "" {
		return ""
	}

	parsed, err := url.Parse(server)
	if err != nil {
		return server
	}

	if proxy.Username != nil && proxy.Password != nil {
		parsed.User = url.UserPassword(*proxy.Username, *proxy.Password)
		return parsed.String()
	}

	return server
}

func resolveProxyGeo(proxy playwright.Proxy) proxyGeo {
	proxyURL := proxyURLWithCreds(proxy)
	if proxyURL == "" {
		return proxyGeo{}
	}

	parsedProxyURL, err := url.Parse(proxyURL)
	if err != nil {
		return proxyGeo{}
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(parsedProxyURL),
		},
	}

	resp, err := client.Get("http://ip-api.com/json?fields=query,timezone")
	if err != nil {
		return proxyGeo{}
	}
	defer resp.Body.Close()

	var payload proxyGeoResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return proxyGeo{}
	}

	return proxyGeo{
		IP:       stringPtrIfNotEmpty(payload.Query),
		Timezone: stringPtrIfNotEmpty(payload.Timezone),
	}
}

func mergeContextOptions(target *playwright.BrowserNewContextOptions, source playwright.BrowserNewContextOptions) {
	targetValue := reflect.ValueOf(target).Elem()
	sourceValue := reflect.ValueOf(source)

	for i := 0; i < targetValue.NumField(); i++ {
		field := sourceValue.Field(i)
		if shouldCopyField(field) {
			targetValue.Field(i).Set(field)
		}
	}
}

func shouldCopyField(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface:
		return !value.IsNil()
	default:
		return !value.IsZero()
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func stringPtrIfNotEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
