package fingerprints

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MUHVUGGAH/zorroz/camoufox/golanglib/camoufox/internal/browserforge"
	"github.com/MUHVUGGAH/zorroz/camoufox/golanglib/camoufox/internal/browserforge/headers"
)

// ScreenFingerprint holds screen-related fingerprint data.
type ScreenFingerprint struct {
	AvailHeight      int     `json:"availHeight"`
	AvailWidth       int     `json:"availWidth"`
	AvailTop         int     `json:"availTop"`
	AvailLeft        int     `json:"availLeft"`
	ColorDepth       int     `json:"colorDepth"`
	Height           int     `json:"height"`
	PixelDepth       int     `json:"pixelDepth"`
	Width            int     `json:"width"`
	DevicePixelRatio float64 `json:"devicePixelRatio"`
	PageXOffset      int     `json:"pageXOffset"`
	PageYOffset      int     `json:"pageYOffset"`
	InnerHeight      int     `json:"innerHeight"`
	OuterHeight      int     `json:"outerHeight"`
	OuterWidth       int     `json:"outerWidth"`
	InnerWidth       int     `json:"innerWidth"`
	ScreenX          int     `json:"screenX"`
	ClientWidth      int     `json:"clientWidth"`
	ClientHeight     int     `json:"clientHeight"`
	HasHDR           bool    `json:"hasHDR"`
}

// NavigatorFingerprint holds navigator-related fingerprint data.
type NavigatorFingerprint struct {
	UserAgent           string            `json:"userAgent"`
	UserAgentData       map[string]string `json:"userAgentData"`
	DoNotTrack          *string           `json:"doNotTrack"`
	AppCodeName         string            `json:"appCodeName"`
	AppName             string            `json:"appName"`
	AppVersion          string            `json:"appVersion"`
	Oscpu               string            `json:"oscpu"`
	Webdriver           string            `json:"webdriver"`
	Language            string            `json:"language"`
	Languages           []string          `json:"languages"`
	Platform            string            `json:"platform"`
	DeviceMemory        *int              `json:"deviceMemory"`
	HardwareConcurrency int               `json:"hardwareConcurrency"`
	Product             string            `json:"product"`
	ProductSub          string            `json:"productSub"`
	Vendor              string            `json:"vendor"`
	VendorSub           string            `json:"vendorSub"`
	MaxTouchPoints      int               `json:"maxTouchPoints"`
	ExtraProperties     map[string]string `json:"extraProperties"`
}

// VideoCard holds video card fingerprint data.
type VideoCard struct {
	Renderer string `json:"renderer"`
	Vendor   string `json:"vendor"`
}

// Fingerprint is the output of the fingerprint generator.
type Fingerprint struct {
	Screen            ScreenFingerprint    `json:"screen"`
	Navigator         NavigatorFingerprint `json:"navigator"`
	Headers           map[string]string    `json:"headers"`
	VideoCodecs       map[string]string    `json:"videoCodecs"`
	AudioCodecs       map[string]string    `json:"audioCodecs"`
	PluginsData       map[string]string    `json:"pluginsData"`
	Battery           map[string]string    `json:"battery,omitempty"`
	VideoCard         *VideoCard           `json:"videoCard,omitempty"`
	MultimediaDevices []string             `json:"multimediaDevices"`
	Fonts             []string             `json:"fonts"`
	MockWebRTC        *bool                `json:"mockWebRTC,omitempty"`
	Slim              *bool                `json:"slim,omitempty"`
}

// Dumps serializes the Fingerprint as a JSON string.
func (f *Fingerprint) Dumps() (string, error) {
	data, err := json.Marshal(f)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Screen constrains the screen dimensions of the generated fingerprint.
type Screen struct {
	MinWidth  *int
	MaxWidth  *int
	MinHeight *int
	MaxHeight *int
}

// NewScreen creates a Screen with optional constraints.
func NewScreen(minWidth, maxWidth, minHeight, maxHeight *int) (*Screen, error) {
	s := &Screen{
		MinWidth:  minWidth,
		MaxWidth:  maxWidth,
		MinHeight: minHeight,
		MaxHeight: maxHeight,
	}
	if minWidth != nil && maxWidth != nil && *minWidth > *maxWidth {
		return nil, fmt.Errorf("invalid screen constraints: min values cannot be greater than max values")
	}
	if minHeight != nil && maxHeight != nil && *minHeight > *maxHeight {
		return nil, fmt.Errorf("invalid screen constraints: min values cannot be greater than max values")
	}
	return s, nil
}

// IsSet returns true if any constraints were set.
func (s *Screen) IsSet() bool {
	if s == nil {
		return false
	}
	return s.MinWidth != nil || s.MaxWidth != nil || s.MinHeight != nil || s.MaxHeight != nil
}

// FingerprintGeneratorConfig provides the file paths needed to initialize a FingerprintGenerator.
type FingerprintGeneratorConfig struct {
	FingerprintNetworkPath string
	HeaderGeneratorConfig  headers.HeaderGeneratorConfig
}

// FingerprintGenerator generates realistic browser fingerprints.
type FingerprintGenerator struct {
	fingerprintGeneratorNetwork *browserforge.BayesianNetwork
	headerGenerator             *headers.HeaderGenerator
	screen                      *Screen
	strict                      bool
	mockWebRTC                  bool
	slim                        bool
}

// NewFingerprintGenerator initializes the FingerprintGenerator with the given options.
func NewFingerprintGenerator(
	config FingerprintGeneratorConfig,
	screen *Screen,
	strict bool,
	mockWebRTC bool,
	slim bool,
	browser []headers.Browser,
	os []string,
	device []string,
	locale []string,
	httpVersion int,
	headerStrict bool,
) (*FingerprintGenerator, error) {
	fpNet, err := browserforge.NewBayesianNetwork(config.FingerprintNetworkPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load fingerprint network: %w", err)
	}

	hg, err := headers.NewHeaderGenerator(
		config.HeaderGeneratorConfig,
		browser, os, device, locale, httpVersion, headerStrict,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create header generator: %w", err)
	}

	return &FingerprintGenerator{
		fingerprintGeneratorNetwork: fpNet,
		headerGenerator:             hg,
		screen:                      screen,
		strict:                      strict,
		mockWebRTC:                  mockWebRTC,
		slim:                        slim,
	}, nil
}

// GenerateOptions holds per-call options for fingerprint generation.
type GenerateOptions struct {
	Screen     *Screen
	Strict     *bool
	MockWebRTC *bool
	Slim       *bool
	// HeaderOptions are passed to HeaderGenerator.Generate
	HeaderOptions *headers.GenerateOptions
}

// Generate generates a fingerprint and a matching set of ordered headers.
func (fg *FingerprintGenerator) Generate(opts *GenerateOptions) (*Fingerprint, error) {
	if opts == nil {
		opts = &GenerateOptions{}
	}

	screen := fg.screen
	if opts.Screen != nil {
		screen = opts.Screen
	}

	strict := fg.strict
	if opts.Strict != nil {
		strict = *opts.Strict
	}

	mockWebRTC := fg.mockWebRTC
	if opts.MockWebRTC != nil {
		mockWebRTC = *opts.MockWebRTC
	}

	slim := fg.slim
	if opts.Slim != nil {
		slim = *opts.Slim
	}

	filteredValues := make(map[string][]string)

	partialCSP := fg.partialCSP(strict, screen, filteredValues)

	headerOpts := opts.HeaderOptions
	if headerOpts == nil {
		headerOpts = &headers.GenerateOptions{}
	}

	if partialCSP != nil {
		if ua, ok := partialCSP["userAgent"]; ok && len(ua) > 0 {
			headerOpts.UserAgent = ua
		}
	}

	hdrs, err := fg.headerGenerator.Generate(headerOpts)
	if err != nil {
		return nil, err
	}

	userAgent, ok := headers.GetUserAgent(hdrs)
	if !ok {
		return nil, fmt.Errorf("failed to find User-Agent in generated response")
	}

	var fingerprint map[string]interface{}
	for {
		merged := make(map[string][]string)
		for k, v := range filteredValues {
			merged[k] = v
		}
		merged["userAgent"] = []string{userAgent}

		fingerprint = fg.fingerprintGeneratorNetwork.GenerateConsistentSampleWhenPossible(merged)
		if fingerprint != nil {
			break
		}

		if strict {
			return nil, fmt.Errorf(
				"cannot generate headers. User-Agent may be invalid, or screen constraints are too restrictive",
			)
		}
		// Relax filtered values
		filteredValues = make(map[string][]string)
	}

	// Process fingerprint attributes
	for attr, val := range fingerprint {
		valStr, ok := val.(string)
		if !ok {
			continue
		}
		if valStr == "*MISSING_VALUE*" {
			fingerprint[attr] = nil
		}
		if strings.HasPrefix(valStr, "*STRINGIFIED*") {
			var parsed interface{}
			if err := json.Unmarshal([]byte(valStr[len("*STRINGIFIED*"):]), &parsed); err == nil {
				fingerprint[attr] = parsed
			}
		}
	}

	// Add accepted languages
	acceptLanguageHeaderValue := ""
	if alVal, ok := hdrs["Accept-Language"]; ok {
		acceptLanguageHeaderValue = alVal
	}
	var acceptedLanguages []string
	for _, locale := range strings.Split(acceptLanguageHeaderValue, ",") {
		parts := strings.SplitN(locale, ";", 2)
		acceptedLanguages = append(acceptedLanguages, strings.TrimSpace(parts[0]))
	}
	fingerprint["languages"] = acceptedLanguages

	return transformFingerprint(fingerprint, hdrs, mockWebRTC, slim)
}

func (fg *FingerprintGenerator) partialCSP(strict bool, screen *Screen, filteredValues map[string][]string) map[string][]string {
	if screen == nil || !screen.IsSet() {
		return nil
	}

	screenNode, ok := fg.fingerprintGeneratorNetwork.NodesByName["screen"]
	if !ok {
		return nil
	}

	var screenValues []string
	for _, screenString := range screenNode.PossibleValues() {
		if isScreenWithinConstraints(screenString, screen) {
			screenValues = append(screenValues, screenString)
		}
	}
	filteredValues["screen"] = screenValues

	result, err := browserforge.GetPossibleValues(fg.fingerprintGeneratorNetwork, filteredValues)
	if err != nil {
		if strict {
			return nil // caller should handle strict mode
		}
		delete(filteredValues, "screen")
		return nil
	}
	return result
}

func isScreenWithinConstraints(screenString string, screenOptions *Screen) bool {
	if !strings.HasPrefix(screenString, "*STRINGIFIED*") {
		return false
	}

	var screenData map[string]interface{}
	if err := json.Unmarshal([]byte(screenString[len("*STRINGIFIED*"):]), &screenData); err != nil {
		return false
	}

	width := getIntFromMap(screenData, "width", -1)
	height := getIntFromMap(screenData, "height", -1)

	minWidth := 0
	if screenOptions.MinWidth != nil {
		minWidth = *screenOptions.MinWidth
	}
	minHeight := 0
	if screenOptions.MinHeight != nil {
		minHeight = *screenOptions.MinHeight
	}
	maxWidth := 100000
	if screenOptions.MaxWidth != nil {
		maxWidth = *screenOptions.MaxWidth
	}
	maxHeight := 100000
	if screenOptions.MaxHeight != nil {
		maxHeight = *screenOptions.MaxHeight
	}

	return width >= minWidth && height >= minHeight && width <= maxWidth && height <= maxHeight
}

func getIntFromMap(m map[string]interface{}, key string, defaultVal int) int {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return int(val)
		case int:
			return val
		}
	}
	return defaultVal
}

func getStringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getStringSliceFromMap(m map[string]interface{}, key string) []string {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case []interface{}:
			result := make([]string, len(val))
			for i, item := range val {
				result[i], _ = item.(string)
			}
			return result
		case []string:
			return val
		}
	}
	return nil
}

func getMapFromInterface(v interface{}) map[string]string {
	if m, ok := v.(map[string]interface{}); ok {
		result := make(map[string]string, len(m))
		for k, val := range m {
			if s, ok := val.(string); ok {
				result[k] = s
			}
		}
		return result
	}
	if m, ok := v.(map[string]string); ok {
		return m
	}
	return nil
}

func transformFingerprint(fp map[string]interface{}, hdrs map[string]string, mockWebRTC bool, slim bool) (*Fingerprint, error) {
	navigatorKeys := []string{
		"userAgent", "userAgentData", "doNotTrack", "appCodeName", "appName",
		"appVersion", "oscpu", "webdriver", "platform", "deviceMemory",
		"product", "productSub", "vendor", "vendorSub", "extraProperties",
		"hardwareConcurrency", "languages",
	}
	navigatorData := make(map[string]interface{})
	for _, k := range navigatorKeys {
		if v, ok := fp[k]; ok {
			navigatorData[k] = v
		}
	}

	languages := getStringSliceFromMap(fp, "languages")
	language := ""
	if len(languages) > 0 {
		language = languages[0]
	}

	maxTouchPoints := 0
	if v, ok := fp["maxTouchPoints"]; ok {
		if f, ok := v.(float64); ok {
			maxTouchPoints = int(f)
		}
	}

	var doNotTrack *string
	if v := getStringFromMap(navigatorData, "doNotTrack"); v != "" {
		doNotTrack = &v
	}

	var deviceMemory *int
	if v, ok := navigatorData["deviceMemory"]; ok && v != nil {
		if f, ok := v.(float64); ok {
			dm := int(f)
			deviceMemory = &dm
		}
	}

	hardwareConcurrency := 0
	if v, ok := navigatorData["hardwareConcurrency"]; ok {
		if f, ok := v.(float64); ok {
			hardwareConcurrency = int(f)
		}
	}

	navigator := NavigatorFingerprint{
		UserAgent:           getStringFromMap(navigatorData, "userAgent"),
		UserAgentData:       getMapFromInterface(navigatorData["userAgentData"]),
		DoNotTrack:          doNotTrack,
		AppCodeName:         getStringFromMap(navigatorData, "appCodeName"),
		AppName:             getStringFromMap(navigatorData, "appName"),
		AppVersion:          getStringFromMap(navigatorData, "appVersion"),
		Oscpu:               getStringFromMap(navigatorData, "oscpu"),
		Webdriver:           getStringFromMap(navigatorData, "webdriver"),
		Language:            language,
		Languages:           languages,
		Platform:            getStringFromMap(navigatorData, "platform"),
		DeviceMemory:        deviceMemory,
		HardwareConcurrency: hardwareConcurrency,
		Product:             getStringFromMap(navigatorData, "product"),
		ProductSub:          getStringFromMap(navigatorData, "productSub"),
		Vendor:              getStringFromMap(navigatorData, "vendor"),
		VendorSub:           getStringFromMap(navigatorData, "vendorSub"),
		MaxTouchPoints:      maxTouchPoints,
		ExtraProperties:     getMapFromInterface(navigatorData["extraProperties"]),
	}

	// Parse screen
	var screenFP ScreenFingerprint
	if screenRaw, ok := fp["screen"]; ok {
		screenData, err := json.Marshal(screenRaw)
		if err == nil {
			json.Unmarshal(screenData, &screenFP)
		}
	}

	// Parse video card
	var videoCard *VideoCard
	if vcRaw, ok := fp["videoCard"]; ok && vcRaw != nil {
		if vcMap, ok := vcRaw.(map[string]interface{}); ok {
			videoCard = &VideoCard{
				Renderer: getStringFromMap(vcMap, "renderer"),
				Vendor:   getStringFromMap(vcMap, "vendor"),
			}
		}
	}

	mockWebRTCPtr := &mockWebRTC
	slimPtr := &slim

	return &Fingerprint{
		Screen:            screenFP,
		Navigator:         navigator,
		Headers:           hdrs,
		VideoCodecs:       getMapFromInterface(fp["videoCodecs"]),
		AudioCodecs:       getMapFromInterface(fp["audioCodecs"]),
		PluginsData:       getMapFromInterface(fp["pluginsData"]),
		Battery:           getMapFromInterface(fp["battery"]),
		VideoCard:         videoCard,
		MultimediaDevices: getStringSliceFromMap(fp, "multimediaDevices"),
		Fonts:             getStringSliceFromMap(fp, "fonts"),
		MockWebRTC:        mockWebRTCPtr,
		Slim:              slimPtr,
	}, nil
}
