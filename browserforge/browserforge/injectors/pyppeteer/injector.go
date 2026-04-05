package pyppeteer

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"browserforge/fingerprints"
	"browserforge/injectors"
)

// PyppeteerBrowser is an interface for Pyppeteer/Puppeteer-like browser implementations.
type PyppeteerBrowser interface {
	NewPage() (Page, error)
	Version() (string, error)
}

// Page is an interface for Pyppeteer/Puppeteer-like page implementations.
type Page interface {
	SetUserAgent(userAgent string) error
	SetExtraHTTPHeaders(headers map[string]string) error
	EvaluateOnNewDocument(script string) error
	Target() Target
	Client() CDPSession
}

// Target is an interface for page targets.
type Target interface {
	CreateCDPSession() (CDPSession, error)
}

// CDPSession is an interface for Chrome DevTools Protocol sessions.
type CDPSession interface {
	Send(method string, params map[string]interface{}) error
}

// NewPage injects a Pyppeteer browser object with a Fingerprint.
func NewPage(
	browser PyppeteerBrowser,
	fp *fingerprints.Fingerprint,
	generator *fingerprints.FingerprintGenerator,
	genOpts *fingerprints.GenerateOptions,
	utilsJSPath string,
) (Page, error) {
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

	// Create a new page
	page, err := browser.NewPage()
	if err != nil {
		return nil, err
	}

	err = page.SetUserAgent(fp.Navigator.UserAgent)
	if err != nil {
		return nil, err
	}

	// Pyppeteer does not support firefox, so we can ignore checks
	cdpSess, err := page.Target().CreateCDPSession()
	if err != nil {
		return nil, err
	}

	// Determine if mobile
	isMobile := strings.Contains(strings.ToLower(fp.Navigator.UserAgent), "phone") ||
		strings.Contains(strings.ToLower(fp.Navigator.UserAgent), "android") ||
		strings.Contains(strings.ToLower(fp.Navigator.UserAgent), "mobile")

	// Determine screen orientation
	orientation := map[string]interface{}{
		"angle": 90,
		"type":  "landscapePrimary",
	}
	if fp.Screen.Height > fp.Screen.Width {
		orientation = map[string]interface{}{
			"angle": 0,
			"type":  "portraitPrimary",
		}
	}

	err = cdpSess.Send("Page.setDeviceMetricsOverride", map[string]interface{}{
		"screenHeight":      fp.Screen.Height,
		"screenWidth":       fp.Screen.Width,
		"width":             fp.Screen.Width,
		"height":            fp.Screen.Height,
		"mobile":            isMobile,
		"screenOrientation": orientation,
		"deviceScaleFactor": fp.Screen.DevicePixelRatio,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set device metrics: %w", err)
	}

	err = page.SetExtraHTTPHeaders(injectors.OnlyInjectableHeaders(fp.Headers, "chrome"))
	if err != nil {
		return nil, err
	}

	// Only set to dark mode if the Chrome version >= 76
	version, err := browser.Version()
	if err == nil {
		re := regexp.MustCompile(`.*?/(\d+)[\d\.]+?`)
		matches := re.FindStringSubmatch(version)
		if len(matches) > 1 {
			majorVersion, _ := strconv.Atoi(matches[1])
			if majorVersion >= 76 {
				page.Client().Send("Emulation.setEmulatedMedia", map[string]interface{}{
					"features": []map[string]interface{}{
						{"name": "prefers-color-scheme", "value": "dark"},
					},
				})
			}
		}
	}

	// Inject function
	err = page.EvaluateOnNewDocument(function)
	if err != nil {
		return nil, err
	}

	return page, nil
}
