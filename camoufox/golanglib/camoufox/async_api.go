package camoufox

import (
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
)

// AsyncCamoufox wraps Playwright to launch a browser with Camoufox settings.
// In Go, "async" is native via goroutines. This struct provides a channel-based
// lifecycle that mirrors the Python AsyncCamoufox context manager.
type AsyncCamoufox struct {
	pw      *playwright.Playwright
	browser playwright.Browser
	context playwright.BrowserContext
	options LaunchOptionsConfig

	// VirtualDisplay holds the Xvfb display if headless == "virtual"
	virtualDisplay *VirtualDisplay
}

// NewAsyncCamoufox creates and starts an AsyncCamoufox instance.
// This is the Go equivalent of `async with AsyncCamoufox(...) as browser:`.
func NewAsyncCamoufox(opts LaunchOptionsConfig) (*AsyncCamoufox, error) {
	ac := &AsyncCamoufox{options: opts}

	pw, err := playwright.Run()
	if err != nil {
		return nil, err
	}
	ac.pw = pw

	result, err := AsyncNewBrowser(pw, opts)
	if err != nil {
		_ = pw.Stop()
		return nil, err
	}

	ac.browser = result.Browser
	ac.context = result.Context
	return ac, nil
}

// Browser returns the underlying Browser (nil if persistent context).
func (ac *AsyncCamoufox) Browser() playwright.Browser {
	return ac.browser
}

// Context returns the underlying BrowserContext (non-nil if persistent context).
func (ac *AsyncCamoufox) Context() playwright.BrowserContext {
	return ac.context
}

// Close shuts down the browser and Playwright runtime.
func (ac *AsyncCamoufox) Close() error {
	var closeErr error
	if ac.context != nil {
		closeErr = ac.context.Close()
	} else if ac.browser != nil {
		closeErr = ac.browser.Close()
	}
	if ac.pw != nil {
		if err := ac.pw.Stop(); closeErr == nil {
			closeErr = err
		}
	}
	if ac.virtualDisplay != nil {
		ac.virtualDisplay.Kill()
	}
	return closeErr
}

// AsyncNewBrowserOptions extends LaunchOptionsConfig with virtual display support.
type AsyncNewBrowserOptions struct {
	LaunchOptionsConfig
	PersistentContext bool
	Debug             *bool
}

// AsyncNewBrowser launches a new browser instance for Camoufox.
// Mirrors the Python AsyncNewBrowser function.
func AsyncNewBrowser(pw *playwright.Playwright, opts LaunchOptionsConfig) (*BrowserInstance, error) {
	launchResult, err := LaunchOptions(opts)
	if err != nil {
		return nil, err
	}

	pwOpts := launchResult.ToPlaywrightLaunchOptions()

	browser, err := pw.Firefox.Launch(pwOpts)
	if err != nil {
		return nil, err
	}

	return &BrowserInstance{Browser: browser}, nil
}

// AsyncNewBrowserPersistent launches a persistent browser context.
func AsyncNewBrowserPersistent(pw *playwright.Playwright, userDataDir string, opts LaunchOptionsConfig) (*BrowserInstance, error) {
	launchResult, err := LaunchOptions(opts)
	if err != nil {
		return nil, err
	}

	persistOpts := playwright.BrowserTypeLaunchPersistentContextOptions{
		ExecutablePath: playwright.String(launchResult.ExecutablePath),
		Headless:       playwright.Bool(launchResult.Headless),
		Args:           launchResult.Args,
	}
	if len(launchResult.FirefoxUserPrefs) > 0 {
		persistOpts.FirefoxUserPrefs = launchResult.FirefoxUserPrefs
	}
	if len(launchResult.Env) > 0 {
		persistOpts.Env = launchResult.Env
	}

	context, err := pw.Firefox.LaunchPersistentContext(userDataDir, persistOpts)
	if err != nil {
		return nil, err
	}

	return &BrowserInstance{Context: context}, nil
}

// AsyncNewContextOptions holds options for creating a new context with fingerprinting.
type AsyncNewContextOptions struct {
	Preset      map[string]any
	OS          *string
	FFVersion   *string
	WebRTCIP    *string
	Proxy       *playwright.Proxy
	Geolocation *playwright.Geolocation
	// Extra Playwright context options to merge.
	ContextKwargs playwright.BrowserNewContextOptions
}

// AsyncNewContext creates a new browser context with a unique fingerprint identity.
// Each context gets its own fingerprint preset (navigator, screen, WebGL, fonts, etc.)
// with unique seeds for audio, canvas, and font spacing noise.
func AsyncNewContext(browser playwright.Browser, opts AsyncNewContextOptions) (playwright.BrowserContext, error) {
	webrtcIP := opts.WebRTCIP

	// Auto-derive WebRTC IP and timezone from proxy's exit IP
	if opts.Proxy != nil && (webrtcIP == nil || opts.ContextKwargs.TimezoneId == nil) {
		geo := asyncResolveProxyGeo(*opts.Proxy)
		if webrtcIP == nil {
			webrtcIP = geo.IP
		}
		if opts.ContextKwargs.TimezoneId == nil && geo.Timezone != nil {
			opts.ContextKwargs.TimezoneId = geo.Timezone
		}
	}

	fp, err := GenerateContextFingerprint(GenerateContextFingerprintOptions{
		Preset:    opts.Preset,
		OS:        opts.OS,
		FFVersion: opts.FFVersion,
		WebRTCIP:  webrtcIP,
	})
	if err != nil {
		return nil, err
	}

	// Merge generated context options with user overrides (user wins)
	merged := fp.ContextOptions
	mergeContextOptions(&merged, opts.ContextKwargs)

	if opts.Proxy != nil {
		merged.Proxy = opts.Proxy
	}
	if opts.Geolocation != nil {
		merged.Geolocation = opts.Geolocation
		if !containsString(merged.Permissions, "geolocation") {
			merged.Permissions = append(merged.Permissions, "geolocation")
		}
	}

	context, err := browser.NewContext(merged)
	if err != nil {
		return nil, err
	}

	if fp.InitScript != "" {
		if err := context.AddInitScript(playwright.Script{
			Content: playwright.String(fp.InitScript),
		}); err != nil {
			_ = context.Close()
			return nil, err
		}
	}

	return context, nil
}

// asyncResolveProxyGeo resolves proxy exit IP and timezone concurrently.
func asyncResolveProxyGeo(proxy playwright.Proxy) proxyGeo {
	type result struct {
		geo proxyGeo
	}
	ch := make(chan result, 1)
	go func() {
		ch <- result{geo: resolveProxyGeo(proxy)}
	}()
	r := <-ch
	return r.geo
}

// AsyncResolveProxyGeo is a concurrent version that resolves proxy geo in a goroutine.
// Returns IP and timezone via channels.
func AsyncResolveProxyGeo(proxy playwright.Proxy) <-chan proxyGeo {
	ch := make(chan proxyGeo, 1)
	go func() {
		ch <- resolveProxyGeo(proxy)
	}()
	return ch
}

// ParallelNewContexts creates multiple browser contexts concurrently, each with unique fingerprints.
// This showcases Go's goroutine advantage over Python's async approach.
func ParallelNewContexts(browser playwright.Browser, count int, opts AsyncNewContextOptions) ([]playwright.BrowserContext, error) {
	type contextResult struct {
		ctx playwright.BrowserContext
		err error
		idx int
	}

	results := make([]contextResult, count)
	var wg sync.WaitGroup

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ctx, err := AsyncNewContext(browser, opts)
			results[idx] = contextResult{ctx: ctx, err: err, idx: idx}
		}(i)
	}
	wg.Wait()

	contexts := make([]playwright.BrowserContext, 0, count)
	for _, r := range results {
		if r.err != nil {
			// Close any successfully created contexts on error
			for _, c := range contexts {
				_ = c.Close()
			}
			return nil, r.err
		}
		contexts = append(contexts, r.ctx)
	}
	return contexts, nil
}

// proxyURLWithCredsAsync is the same as proxyURLWithCreds but exported for async use.
func proxyURLWithCredsAsync(proxy playwright.Proxy) string {
	return proxyURLWithCreds(proxy)
}

// asyncResolveProxyGeoHTTP performs the actual HTTP lookup through the proxy.
func asyncResolveProxyGeoHTTP(proxyURL string) proxyGeo {
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
