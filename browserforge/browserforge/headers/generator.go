package headers

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/MUHVUGGAH/zorroz/browserforge/browserforge"
)

// Constants
var (
	SupportedBrowsers         = []string{"chrome", "firefox", "safari", "edge"}
	SupportedOperatingSystems = []string{"windows", "macos", "linux", "android", "ios"}
	SupportedDevices          = []string{"desktop", "mobile"}
	SupportedHTTPVersions     = []string{"1", "2"}
	MissingValueDatasetToken  = "*MISSING_VALUE*"
)

var http1SecFetchAttributes = map[string]string{
	"Sec-Fetch-Mode": "same-site",
	"Sec-Fetch-Dest": "navigate",
	"Sec-Fetch-Site": "?1",
	"Sec-Fetch-User": "document",
}

var http2SecFetchAttributes = map[string]string{
	"sec-fetch-mode": "same-site",
	"sec-fetch-dest": "navigate",
	"sec-fetch-site": "?1",
	"sec-fetch-user": "document",
}

// Browser represents a browser specification with name, min/max version, and HTTP version.
type Browser struct {
	Name        string
	MinVersion  int // 0 means no minimum
	MaxVersion  int // 0 means no maximum
	HTTPVersion string
}

// NewBrowser creates a Browser with the given name and HTTP version.
func NewBrowser(name string, httpVersion string) Browser {
	return Browser{Name: name, HTTPVersion: httpVersion}
}

// Validate checks that Browser constraints are valid.
func (b Browser) Validate() error {
	if b.MinVersion > 0 && b.MaxVersion > 0 && b.MinVersion > b.MaxVersion {
		return fmt.Errorf(
			"browser min version constraint (%d) cannot exceed max version (%d)",
			b.MinVersion, b.MaxVersion,
		)
	}
	return nil
}

// HttpBrowserObject represents an HTTP browser object with name, version, complete string,
// and HTTP version.
type HttpBrowserObject struct {
	Name           string
	Version        []int
	CompleteString string
	HTTPVersion    string
}

// IsHTTP2 returns true if this browser object uses HTTP/2.
func (h *HttpBrowserObject) IsHTTP2() bool {
	return h.HTTPVersion == "2"
}

// HeaderGeneratorConfig provides the file paths needed to initialize a HeaderGenerator.
type HeaderGeneratorConfig struct {
	InputNetworkPath      string
	HeaderNetworkPath     string
	HeadersOrderPath      string
	BrowserHelperFilePath string
}

// HeaderGeneratorOptions holds the default options for header generation.
type HeaderGeneratorOptions struct {
	Browsers    []Browser
	OS          []string
	Devices     []string
	Locales     []string
	HTTPVersion string
	Strict      bool
}

// GenerateOptions holds per-call options that override defaults.
type GenerateOptions struct {
	Browsers                []Browser
	OS                      []string
	Devices                 []string
	Locales                 []string
	HTTPVersion             string
	UserAgent               []string
	Strict                  *bool
	RequestDependentHeaders map[string]string
}

// HeaderGenerator generates HTTP headers based on a set of constraints.
type HeaderGenerator struct {
	options        HeaderGeneratorOptions
	uniqueBrowsers []HttpBrowserObject
	headersOrder   map[string][]string

	InputGeneratorNetwork  *browserforge.BayesianNetwork
	HeaderGeneratorNetwork *browserforge.BayesianNetwork
}

var relaxationOrder = []string{"locales", "devices", "operatingSystems", "browsers"}

// NewHeaderGenerator initializes the HeaderGenerator with the given config and options.
func NewHeaderGenerator(
	config HeaderGeneratorConfig,
	browser []Browser,
	os []string,
	device []string,
	locale []string,
	httpVersion int,
	strict bool,
) (*HeaderGenerator, error) {
	inputNet, err := browserforge.NewBayesianNetwork(config.InputNetworkPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load input network: %w", err)
	}
	headerNet, err := browserforge.NewBayesianNetwork(config.HeaderNetworkPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load header network: %w", err)
	}

	httpVer := strconv.Itoa(httpVersion)
	if httpVersion == 0 {
		httpVer = "2"
	}

	if browser == nil {
		browser = make([]Browser, len(SupportedBrowsers))
		for i, name := range SupportedBrowsers {
			browser[i] = NewBrowser(name, httpVer)
		}
	} else {
		browser = prepareBrowsersConfig(browser, httpVer)
	}

	if os == nil {
		os = SupportedOperatingSystems
	}
	if device == nil {
		device = SupportedDevices
	}
	if locale == nil {
		locale = []string{"en-US"}
	}

	hg := &HeaderGenerator{
		options: HeaderGeneratorOptions{
			Browsers:    browser,
			OS:          os,
			Devices:     device,
			Locales:     locale,
			HTTPVersion: httpVer,
			Strict:      strict,
		},
		InputGeneratorNetwork:  inputNet,
		HeaderGeneratorNetwork: headerNet,
	}

	hg.uniqueBrowsers, err = hg.loadUniqueBrowsers(config.BrowserHelperFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load unique browsers: %w", err)
	}

	hg.headersOrder, err = loadHeadersOrder(config.HeadersOrderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load headers order: %w", err)
	}

	return hg, nil
}

// Generate generates headers using the default options and their possible overrides.
func (hg *HeaderGenerator) Generate(opts *GenerateOptions) (map[string]string, error) {
	if opts == nil {
		opts = &GenerateOptions{}
	}

	options := hg.mergeOptions(opts)
	generated, err := hg.getHeaders(options)
	if err != nil {
		return nil, err
	}

	httpVer := options.HTTPVersion
	if httpVer == "" {
		httpVer = hg.options.HTTPVersion
	}
	if httpVer == "2" {
		return PascalizeHeaders(generated), nil
	}
	return generated, nil
}

type mergedOptions struct {
	Browsers                []Browser
	OS                      []string
	Devices                 []string
	Locales                 []string
	HTTPVersion             string
	Strict                  bool
	UserAgent               []string
	RequestDependentHeaders map[string]string
}

func (hg *HeaderGenerator) mergeOptions(opts *GenerateOptions) mergedOptions {
	m := mergedOptions{
		Browsers:    hg.options.Browsers,
		OS:          hg.options.OS,
		Devices:     hg.options.Devices,
		Locales:     hg.options.Locales,
		HTTPVersion: hg.options.HTTPVersion,
		Strict:      hg.options.Strict,
	}

	if opts.Browsers != nil {
		m.Browsers = opts.Browsers
	}
	if opts.OS != nil {
		m.OS = opts.OS
	}
	if opts.Devices != nil {
		m.Devices = opts.Devices
	}
	if opts.Locales != nil {
		m.Locales = opts.Locales
	}
	if opts.HTTPVersion != "" {
		m.HTTPVersion = opts.HTTPVersion
	}
	if opts.Strict != nil {
		m.Strict = *opts.Strict
	}
	if opts.UserAgent != nil {
		m.UserAgent = opts.UserAgent
	}
	if opts.RequestDependentHeaders != nil {
		m.RequestDependentHeaders = opts.RequestDependentHeaders
	}
	return m
}

func (hg *HeaderGenerator) getHeaders(options mergedOptions) (map[string]string, error) {
	requestDependentHeaders := options.RequestDependentHeaders
	if requestDependentHeaders == nil {
		requestDependentHeaders = map[string]string{}
	}

	// Handle browser/http_version overrides
	browsers := options.Browsers
	if browsers == nil {
		browsers = hg.options.Browsers
	}

	if options.HTTPVersion != "" && options.HTTPVersion != hg.options.HTTPVersion {
		browsers = prepareBrowsersConfig(browsers, options.HTTPVersion)
	}

	possibleAttributeValues := hg.getPossibleAttributeValues(browsers, options)

	var http1Values, http2Values map[string][]string
	if len(options.UserAgent) > 0 {
		var err error
		http1Values, err = browserforge.GetPossibleValues(hg.HeaderGeneratorNetwork, map[string][]string{
			"User-Agent": options.UserAgent,
		})
		if err != nil {
			http1Values = map[string][]string{}
		}
		http2Values, err = browserforge.GetPossibleValues(hg.HeaderGeneratorNetwork, map[string][]string{
			"user-agent": options.UserAgent,
		})
		if err != nil {
			http2Values = map[string][]string{}
		}
	}

	constraints := hg.prepareConstraints(possibleAttributeValues, http1Values, http2Values)

	inputSample := hg.InputGeneratorNetwork.GenerateConsistentSampleWhenPossible(constraints)
	if inputSample == nil {
		if options.HTTPVersion == "1" {
			// Try HTTP/2 fallback
			fallbackOpts := options
			fallbackOpts.HTTPVersion = "2"
			headers2, err := hg.getHeaders(fallbackOpts)
			if err != nil {
				return nil, err
			}
			return hg.orderHeaders(PascalizeHeaders(headers2))
		}

		// Try relaxation
		relaxationIndex := -1
		for i, key := range relaxationOrder {
			switch key {
			case "locales":
				if options.Locales != nil {
					relaxationIndex = i
				}
			case "devices":
				if options.Devices != nil {
					relaxationIndex = i
				}
			case "operatingSystems":
				if options.OS != nil {
					relaxationIndex = i
				}
			case "browsers":
				if options.Browsers != nil {
					relaxationIndex = i
				}
			}
			if relaxationIndex >= 0 {
				break
			}
		}

		if options.Strict || relaxationIndex == -1 {
			return nil, fmt.Errorf(
				"no headers based on this input can be generated. Please relax or change some of the requirements you specified",
			)
		}

		relaxedOpts := options
		switch relaxationOrder[relaxationIndex] {
		case "locales":
			relaxedOpts.Locales = nil
		case "devices":
			relaxedOpts.Devices = nil
		case "operatingSystems":
			relaxedOpts.OS = nil
		case "browsers":
			relaxedOpts.Browsers = nil
		}
		return hg.getHeaders(relaxedOpts)
	}

	generatedSample := hg.HeaderGeneratorNetwork.GenerateSample(inputSample)
	browserHTTPStr, _ := generatedSample["*BROWSER_HTTP"].(string)
	generatedHTTPAndBrowser := hg.prepareHTTPBrowserObject(browserHTTPStr)

	// Add Accept-Language header
	acceptLanguageFieldName := "Accept-Language"
	if generatedHTTPAndBrowser.IsHTTP2() {
		acceptLanguageFieldName = "accept-language"
	}
	generatedSample[acceptLanguageFieldName] = getAcceptLanguageHeader(options.Locales)

	// Add Sec headers
	if shouldAddSecFetch(generatedHTTPAndBrowser) {
		if generatedHTTPAndBrowser.IsHTTP2() {
			for k, v := range http2SecFetchAttributes {
				generatedSample[k] = v
			}
		} else {
			for k, v := range http1SecFetchAttributes {
				generatedSample[k] = v
			}
		}
	}

	// Filter out connection close, missing values, and internal keys
	result := make(map[string]string)
	for k, v := range generatedSample {
		vStr, ok := v.(string)
		if !ok {
			continue
		}
		if strings.ToLower(k) == "connection" && vStr == "close" {
			continue
		}
		if strings.HasPrefix(k, "*") {
			continue
		}
		if vStr == MissingValueDatasetToken {
			continue
		}
		result[k] = vStr
	}

	// Merge request-dependent headers
	for k, v := range requestDependentHeaders {
		result[k] = v
	}

	return hg.orderHeaders(result)
}

func (hg *HeaderGenerator) getPossibleAttributeValues(browsers []Browser, options mergedOptions) map[string][]string {
	browserHTTPOptions := hg.getBrowserHTTPOptions(browsers)

	possibleAttributeValues := map[string][]string{
		"*BROWSER_HTTP":     browserHTTPOptions,
		"*OPERATING_SYSTEM": options.OS,
	}
	if options.Devices != nil {
		possibleAttributeValues["*DEVICE"] = options.Devices
	}

	return possibleAttributeValues
}

func (hg *HeaderGenerator) getBrowserHTTPOptions(browsers []Browser) []string {
	var result []string
	for _, browser := range browsers {
		for _, browserOption := range hg.uniqueBrowsers {
			if browser.Name != browserOption.Name {
				continue
			}
			if browser.MinVersion > 0 && len(browserOption.Version) > 0 && browser.MinVersion > browserOption.Version[0] {
				continue
			}
			if browser.MaxVersion > 0 && len(browserOption.Version) > 0 && browser.MaxVersion < browserOption.Version[0] {
				continue
			}
			if browser.HTTPVersion != "" && browser.HTTPVersion != browserOption.HTTPVersion {
				continue
			}
			result = append(result, browserOption.CompleteString)
		}
	}
	return result
}

func (hg *HeaderGenerator) orderHeaders(headers map[string]string) (map[string]string, error) {
	ua, ok := GetUserAgent(headers)
	if !ok {
		return nil, fmt.Errorf("failed to find User-Agent in generated response")
	}
	browserName := GetBrowser(ua)
	if browserName == "" {
		return nil, fmt.Errorf("failed to find browser in User-Agent")
	}

	headerOrder, ok := hg.headersOrder[browserName]
	if !ok {
		return headers, nil
	}

	ordered := make(map[string]string, len(headers))
	for _, key := range headerOrder {
		if val, exists := headers[key]; exists {
			ordered[key] = val
		}
	}
	// Add any remaining headers not in the order
	for k, v := range headers {
		if _, exists := ordered[k]; !exists {
			ordered[k] = v
		}
	}
	return ordered, nil
}

func (hg *HeaderGenerator) prepareConstraints(
	possibleAttributeValues map[string][]string,
	http1Values, http2Values map[string][]string,
) map[string][]string {
	constraints := make(map[string][]string)

	for key, values := range possibleAttributeValues {
		var filtered []string
		for _, x := range values {
			if key == "*BROWSER_HTTP" {
				if filterBrowserHTTP(x, http1Values, http2Values) {
					filtered = append(filtered, x)
				}
			} else {
				if filterOtherValues(x, http1Values, http2Values, key) {
					filtered = append(filtered, x)
				}
			}
		}
		constraints[key] = filtered
	}

	return constraints
}

func filterBrowserHTTP(value string, http1Values, http2Values map[string][]string) bool {
	parts := strings.SplitN(value, "|", 2)
	if len(parts) != 2 {
		return false
	}
	browserName := parts[0]
	httpVersion := parts[1]

	if httpVersion == "1" {
		if len(http1Values) == 0 {
			return true
		}
		return containsString(http1Values["*BROWSER"], browserName)
	}
	if len(http2Values) == 0 {
		return true
	}
	return containsString(http2Values["*BROWSER"], browserName)
}

func filterOtherValues(value string, http1Values, http2Values map[string][]string, key string) bool {
	if len(http1Values) > 0 || len(http2Values) > 0 {
		return containsString(http1Values[key], value) || containsString(http2Values[key], value)
	}
	return true
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func shouldAddSecFetch(browser *HttpBrowserObject) bool {
	if browser.Name == "chrome" && len(browser.Version) > 0 && browser.Version[0] >= 76 {
		return true
	}
	if browser.Name == "firefox" && len(browser.Version) > 0 && browser.Version[0] >= 90 {
		return true
	}
	if browser.Name == "edge" && len(browser.Version) > 0 && browser.Version[0] >= 79 {
		return true
	}
	return false
}

func getAcceptLanguageHeader(locales []string) string {
	parts := make([]string, len(locales))
	for i, locale := range locales {
		parts[i] = fmt.Sprintf("%s;q=%.1f", locale, 1.0-float64(i)*0.1)
	}
	return strings.Join(parts, ", ")
}

func prepareBrowsersConfig(browsers []Browser, httpVersion string) []Browser {
	result := make([]Browser, len(browsers))
	for i, browser := range browsers {
		result[i] = browser
		if browser.HTTPVersion == "" {
			result[i].HTTPVersion = httpVersion
		}
	}
	return result
}

func loadHeadersOrder(path string) (map[string][]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var result map[string][]string
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (hg *HeaderGenerator) loadUniqueBrowsers(path string) ([]HttpBrowserObject, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var browserStrings []string
	if err := json.Unmarshal(data, &browserStrings); err != nil {
		return nil, err
	}

	var result []HttpBrowserObject
	for _, browserStr := range browserStrings {
		if browserStr == MissingValueDatasetToken {
			continue
		}
		obj := hg.prepareHTTPBrowserObject(browserStr)
		result = append(result, *obj)
	}
	return result, nil
}

func (hg *HeaderGenerator) prepareHTTPBrowserObject(httpBrowserString string) *HttpBrowserObject {
	parts := strings.SplitN(httpBrowserString, "|", 2)
	if len(parts) != 2 {
		return &HttpBrowserObject{
			CompleteString: httpBrowserString,
		}
	}

	browserString := parts[0]
	httpVersion := parts[1]

	if browserString == MissingValueDatasetToken {
		return &HttpBrowserObject{
			CompleteString: MissingValueDatasetToken,
		}
	}

	browserParts := strings.SplitN(browserString, "/", 2)
	if len(browserParts) != 2 {
		return &HttpBrowserObject{
			Name:           browserString,
			CompleteString: httpBrowserString,
			HTTPVersion:    httpVersion,
		}
	}

	browserName := browserParts[0]
	versionString := browserParts[1]
	versionParts := strings.Split(versionString, ".")

	version := make([]int, len(versionParts))
	for i, part := range versionParts {
		v, _ := strconv.Atoi(part)
		version[i] = v
	}

	return &HttpBrowserObject{
		Name:           browserName,
		Version:        version,
		CompleteString: httpBrowserString,
		HTTPVersion:    httpVersion,
	}
}
