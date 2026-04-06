package camoufox

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/MUHVUGGAH/zorroz/camoufox/golanglib/camoufox/webgl"
	"github.com/playwright-community/playwright-go"
)

// ListOrString represents a value that can be a single string or a slice.
type ListOrString = interface{} // string or []string

// CachePrefs are Firefox preferences for caching.
var CachePrefs = map[string]interface{}{
	"browser.sessionhistory.max_entries":       10,
	"browser.sessionhistory.max_total_viewers": -1,
	"browser.cache.memory.enable":              true,
	"browser.cache.disk_cache_ssl":             true,
	"browser.cache.disk.smart_size.enabled":    true,
}

// LaunchOptionsConfig holds all the parameters for LaunchOptions.
type LaunchOptionsConfig struct {
	Config            map[string]interface{}
	OS                ListOrString // string or []string
	BlockImages       bool
	BlockWebRTC       bool
	BlockWebGL        bool
	DisableCOOP       bool
	WebGLConfig       *[2]string  // (vendor, renderer)
	GeoIP             interface{} // string(IP) or bool
	GeoIPDB           string
	Humanize          interface{} // bool or float64
	Locale            interface{} // string or []string
	Addons            []string
	Fonts             []string
	CustomFontsOnly   bool
	ExcludeAddons     map[string]bool
	Window            *[2]int
	FingerprintPreset interface{} // bool or map[string]interface{}
	FFVersion         *int
	Headless          *bool
	MainWorldEval     bool
	ExecutablePath    string
	Browser           string
	FirefoxUserPrefs  map[string]interface{}
	Proxy             map[string]string
	EnableCache       bool
	Args              []string
	Env               map[string]string
	IKnowWhatImDoing  bool
	Debug             bool
	VirtualDisplay    string
	ExtraLaunchOpts   map[string]interface{}
}

// LaunchOptionsResult holds the computed Playwright launch options.
type LaunchOptionsResult struct {
	ExecutablePath   string
	Args             []string
	Env              map[string]string
	FirefoxUserPrefs map[string]interface{}
	Headless         bool
	Proxy            map[string]string
}

// GetEnvVars builds a map of CAMOU_CONFIG environment variables.
func GetEnvVars(configMap map[string]interface{}, userAgentOS string) (map[string]string, error) {
	data, err := json.Marshal(configMap)
	if err != nil {
		return nil, fmt.Errorf("error encoding config: %w", err)
	}
	configStr := string(data)

	chunkSize := 32767
	if OSName() == "win" {
		chunkSize = 2047
	}

	envVars := map[string]string{}
	for i := 0; i < len(configStr); i += chunkSize {
		end := i + chunkSize
		if end > len(configStr) {
			end = len(configStr)
		}
		chunk := configStr[i:end]
		envName := fmt.Sprintf("CAMOU_CONFIG_%d", (i/chunkSize)+1)
		envVars[envName] = chunk
	}
	return envVars, nil
}

// LoadProperties loads properties.json from the given path or the active install.
func LoadProperties(path string) (map[string]string, error) {
	var propFile string
	if path != "" {
		propFile = filepath.Join(filepath.Dir(path), "properties.json")
	} else {
		p, err := GetPath("properties.json")
		if err != nil {
			return nil, err
		}
		propFile = p
	}
	data, err := os.ReadFile(propFile)
	if err != nil {
		return nil, err
	}
	var props []struct {
		Property string `json:"property"`
		Type     string `json:"type"`
	}
	if err := json.Unmarshal(data, &props); err != nil {
		return nil, err
	}
	result := make(map[string]string, len(props))
	for _, p := range props {
		result[p.Property] = p.Type
	}
	return result, nil
}

// ValidateConfig validates config keys against properties.json types.
func ValidateConfig(configMap map[string]interface{}, execPath string) error {
	propTypes, err := LoadProperties(execPath)
	if err != nil {
		return err // Skip validation if properties.json not found
	}
	for key, value := range configMap {
		expected, ok := propTypes[key]
		if !ok {
			continue // Skip unknown properties silently
		}
		if !validateType(value, expected) {
			return fmt.Errorf("%w: %s expected %s, got %T", ErrInvalidPropertyType, key, expected, value)
		}
	}
	return nil
}

func validateType(value interface{}, expected string) bool {
	switch expected {
	case "str":
		_, ok := value.(string)
		return ok
	case "int":
		switch v := value.(type) {
		case int:
			return true
		case int64:
			return true
		case float64:
			return v == float64(int(v))
		default:
			return false
		}
	case "uint":
		switch v := value.(type) {
		case int:
			return v >= 0
		case int64:
			return v >= 0
		case float64:
			return v >= 0 && v == float64(int(v))
		default:
			return false
		}
	case "double":
		switch value.(type) {
		case float64, int, int64:
			return true
		default:
			return false
		}
	case "bool":
		_, ok := value.(bool)
		return ok
	case "array":
		switch value.(type) {
		case []interface{}, []string:
			return true
		default:
			return false
		}
	case "dict":
		_, ok := value.(map[string]interface{})
		return ok
	default:
		return false
	}
}

// GetTargetOS derives the OS from config or defaults to the current system OS.
func GetTargetOS(config map[string]interface{}) string {
	if ua, ok := config["navigator.userAgent"].(string); ok && ua != "" {
		return DetermineUAOS(ua)
	}
	return OSName()
}

var uaOSPatterns = []struct {
	prefix string
	os     string
}{
	{"Mac", "mac"},
	{"Windows", "win"},
}

// DetermineUAOS extracts the OS from a user-agent string.
func DetermineUAOS(userAgent string) string {
	ua := strings.ToLower(userAgent)
	if strings.Contains(ua, "mac") || strings.Contains(ua, "darwin") {
		return "mac"
	}
	if strings.Contains(ua, "windows") || strings.Contains(ua, "win") {
		return "win"
	}
	return "lin"
}

// CheckValidOS validates the target OS values.
func CheckValidOS(os interface{}) error {
	switch v := os.(type) {
	case string:
		if v != strings.ToLower(v) {
			return fmt.Errorf("%w: OS values must be lowercase: '%s'", ErrInvalidOS, v)
		}
		if v != "windows" && v != "macos" && v != "linux" {
			return fmt.Errorf("%w: Camoufox does not support: '%s'", ErrInvalidOS, v)
		}
	case []string:
		for _, s := range v {
			if err := CheckValidOS(s); err != nil {
				return err
			}
		}
	}
	return nil
}

// IsDomainSet checks if any of the properties are set in the config.
func IsDomainSet(config map[string]interface{}, properties ...string) bool {
	for _, prop := range properties {
		if strings.HasSuffix(prop, ".") || strings.HasSuffix(prop, ":") {
			for key := range config {
				if strings.HasPrefix(key, prop) {
					return true
				}
			}
		} else {
			if _, ok := config[prop]; ok {
				return true
			}
		}
	}
	return false
}

// WarnManualConfig warns about manual settings that Camoufox handles internally.
func WarnManualConfig(config map[string]interface{}) {
	bFalse := false
	if IsDomainSet(config, "navigator.language", "navigator.languages", "headers.Accept-Language", "locale:") {
		LeakWarning("locale", &bFalse)
	}
	if IsDomainSet(config, "geolocation:", "timezone") {
		LeakWarning("geolocation", &bFalse)
	}
	if IsDomainSet(config, "headers.User-Agent") {
		LeakWarning("header-ua", &bFalse)
	}
	if IsDomainSet(config, "navigator.") {
		LeakWarning("navigator", &bFalse)
	}
	if IsDomainSet(config, "screen.", "window.", "document.body.") {
		LeakWarning("viewport", &bFalse)
	}
}

// MergeInto merges source keys into target where key doesn't already exist.
func MergeInto(target, source map[string]interface{}) {
	for k, v := range source {
		if _, exists := target[k]; !exists {
			target[k] = v
		}
	}
}

// SetInto sets a key into the target only if it doesn't exist.
func SetInto(target map[string]interface{}, key string, value interface{}) {
	if _, exists := target[key]; !exists {
		target[key] = value
	}
}

// UpdateFonts loads fonts from fonts.json for the target OS and merges them.
func UpdateFonts(config map[string]interface{}, targetOS string) error {
	data, err := os.ReadFile(filepath.Join(LocalDataDir(), "fonts.json"))
	if err != nil {
		return err
	}
	var fontsByOS map[string][]string
	if err := json.Unmarshal(data, &fontsByOS); err != nil {
		return err
	}
	fonts := fontsByOS[targetOS]
	if existing, ok := config["fonts"].([]string); ok {
		merged := appendUnique(fonts, existing)
		config["fonts"] = merged
	} else {
		config["fonts"] = fonts
	}
	return nil
}

func appendUnique(a, b []string) []string {
	seen := map[string]bool{}
	for _, s := range a {
		seen[s] = true
	}
	result := append([]string(nil), a...)
	for _, s := range b {
		if !seen[s] {
			result = append(result, s)
			seen[s] = true
		}
	}
	return result
}

// firefoxVersionRegex matches a 3-digit Firefox major version (1xx) followed by ".0"
// at a word/non-digit boundary. Go regexp doesn't support lookbehind/lookahead,
// so we capture optional surrounding digits and check in code if needed.
var firefoxVersionRegex = regexp.MustCompile(`(?:^|\D)(1[0-9]{2})(\.0)(?:\D|$)`)

// LaunchOptions builds the Playwright Firefox launch options (mirrors utils.py:launch_options).
func LaunchOptions(opts LaunchOptionsConfig) (*LaunchOptionsResult, error) {
	config := opts.Config
	if config == nil {
		config = map[string]interface{}{}
	}

	headless := false
	if opts.Headless != nil {
		headless = *opts.Headless
	}

	addons := opts.Addons
	if addons == nil {
		addons = []string{}
	}
	args := opts.Args
	if args == nil {
		args = []string{}
	}
	firefoxUserPrefs := opts.FirefoxUserPrefs
	if firefoxUserPrefs == nil {
		firefoxUserPrefs = map[string]interface{}{}
	}
	env := opts.Env
	if env == nil {
		env = map[string]string{}
	}

	// Handle virtual display
	if opts.VirtualDisplay != "" {
		env["DISPLAY"] = opts.VirtualDisplay
	}

	// Warn manual config settings
	if !opts.IKnowWhatImDoing {
		WarnManualConfig(config)
	}

	// Validate OS
	if opts.OS != nil {
		if err := CheckValidOS(opts.OS); err != nil {
			return nil, err
		}
	} else if opts.WebGLConfig != nil {
		return nil, fmt.Errorf("OS must be set when using WebGLConfig")
	}

	// Add default addons
	AddDefaultAddons(&addons, opts.ExcludeAddons)

	// Confirm addon paths
	if len(addons) > 0 {
		if err := ConfirmPaths(addons); err != nil {
			return nil, err
		}
		config["addons"] = addons
	}

	// Get Firefox version
	var ffVersionStr string
	if opts.FFVersion != nil {
		ffVersionStr = fmt.Sprintf("%d", *opts.FFVersion)
		bFalse := false
		LeakWarning("ff_version", &bFalse)
	} else {
		verStr, err := InstalledVerStr()
		if err != nil {
			ffVersionStr = "134"
		} else {
			parts := strings.SplitN(verStr, ".", 2)
			ffVersionStr = parts[0]
		}
	}

	// Handle fingerprint preset
	usedPreset := false
	if opts.FingerprintPreset != nil {
		switch p := opts.FingerprintPreset.(type) {
		case map[string]interface{}:
			preset, err := decodePreset(p)
			if err == nil {
				MergeInto(config, FromPreset(*preset, &ffVersionStr))
				usedPreset = true
			}
		case bool:
			if p {
				var osNames []string
				switch v := opts.OS.(type) {
				case string:
					osNames = []string{v}
				case []string:
					osNames = v
				}
				preset, err := GetRandomPreset(osNames...)
				if err == nil && preset != nil {
					MergeInto(config, FromPreset(*preset, &ffVersionStr))
					usedPreset = true
				}
			}
		}
	}

	if !usedPreset {
		// BrowserForge generation path
		// TODO: When full BrowserForge Go generation is ported, call it here.
		// For now, we'll use preset-based generation as fallback.
		var osNames []string
		switch v := opts.OS.(type) {
		case string:
			osNames = []string{v}
		case []string:
			osNames = v
		}
		if len(osNames) == 0 {
			osNames = []string{"macos", "windows", "linux"}
		}
		preset, err := GetRandomPreset(osNames...)
		if err == nil && preset != nil {
			MergeInto(config, FromPreset(*preset, &ffVersionStr))
			usedPreset = true
		}
	}

	targetOS := GetTargetOS(config)

	// Set random window.history.length
	SetInto(config, "window.history.length", rand.Intn(5)+1) //nolint:gosec

	// Handle fonts
	if len(opts.Fonts) > 0 {
		config["fonts"] = opts.Fonts
	}

	if opts.CustomFontsOnly {
		firefoxUserPrefs["gfx.bundled-fonts.activate"] = 0
		if len(opts.Fonts) == 0 {
			return nil, fmt.Errorf("no custom fonts were passed, but CustomFontsOnly is enabled")
		}
	} else if _, ok := config["fonts"]; !ok {
		osName := map[string]string{"win": "windows", "mac": "macos", "lin": "linux"}[targetOS]
		if osName == "" {
			osName = "macos"
		}
		fonts, err := GenerateRandomFontSubset(osName)
		if err != nil {
			UpdateFonts(config, targetOS)
		} else {
			config["fonts"] = fonts
		}
	}

	// Generate voice subset
	if _, ok := config["voices"]; !ok {
		osNameV := map[string]string{"win": "windows", "mac": "macos", "lin": "linux"}[targetOS]
		if osNameV == "" {
			osNameV = "macos"
		}
		if voices, err := GenerateRandomVoiceSubset(osNameV); err == nil {
			config["voices"] = voices
		}
	}

	// Set random seeds
	SetInto(config, "fonts:spacing_seed", randomNonZeroUint32())
	SetInto(config, "audio:seed", randomNonZeroUint32())
	SetInto(config, "canvas:seed", randomNonZeroUint32())

	// GeoIP
	if opts.GeoIP != nil {
		switch geoip := opts.GeoIP.(type) {
		case bool:
			if geoip {
				var ip string
				var err error
				if len(opts.Proxy) > 0 {
					p := &CamoufoxProxy{
						Server:   opts.Proxy["server"],
						Username: opts.Proxy["username"],
						Password: opts.Proxy["password"],
					}
					ip, err = PublicIP(p.AsString())
				} else {
					ip, err = PublicIP("")
				}
				if err != nil {
					return nil, err
				}
				if err := setGeoIPConfig(config, firefoxUserPrefs, ip, opts.GeoIPDB, opts.BlockWebRTC); err != nil {
					return nil, err
				}
			}
		case string:
			if err := setGeoIPConfig(config, firefoxUserPrefs, geoip, opts.GeoIPDB, opts.BlockWebRTC); err != nil {
				return nil, err
			}
		}
	} else if len(opts.Proxy) > 0 &&
		!strings.Contains(opts.Proxy["server"], "localhost") &&
		!IsDomainSet(config, "geolocation") {
		LeakWarning("proxy_without_geoip", nil)
	}

	// Locale
	if opts.Locale != nil {
		if err := HandleLocales(opts.Locale, config); err != nil {
			return nil, err
		}
	}

	// Humanize
	if opts.Humanize != nil {
		switch h := opts.Humanize.(type) {
		case bool:
			if h {
				SetInto(config, "humanize", true)
			}
		case float64:
			SetInto(config, "humanize", true)
			SetInto(config, "humanize:maxTime", h)
		case int:
			SetInto(config, "humanize", true)
			SetInto(config, "humanize:maxTime", float64(h))
		}
	}

	// Main world eval
	if opts.MainWorldEval {
		SetInto(config, "allowMainWorld", true)
	}

	// Firefox user prefs
	if opts.BlockImages {
		bFalse := opts.IKnowWhatImDoing
		LeakWarning("block_images", &bFalse)
		firefoxUserPrefs["permissions.default.image"] = 2
	}
	if opts.BlockWebRTC {
		firefoxUserPrefs["media.peerconnection.enabled"] = false
	}
	if opts.DisableCOOP {
		bFalse := opts.IKnowWhatImDoing
		LeakWarning("disable_coop", &bFalse)
		firefoxUserPrefs["browser.tabs.remote.useCrossOriginOpenerPolicy"] = false
	}

	if opts.BlockWebGL {
		firefoxUserPrefs["webgl.disabled"] = true
		bFalse := opts.IKnowWhatImDoing
		LeakWarning("block_webgl", &bFalse)
	} else {
		var webglFP map[string]interface{}
		var err error
		if opts.WebGLConfig != nil {
			vendor := opts.WebGLConfig[0]
			renderer := opts.WebGLConfig[1]
			webglFP, err = sampleWebGLHelper(targetOS, &vendor, &renderer)
		} else if v, ok := config["webGl:vendor"].(string); ok && v != "" {
			r, _ := config["webGl:renderer"].(string)
			webglFP, err = sampleWebGLHelper(targetOS, &v, &r)
		} else {
			webglFP, err = sampleWebGLHelper(targetOS, nil, nil)
		}
		if err == nil && webglFP != nil {
			enableWebGL2, _ := webglFP["webGl2Enabled"].(bool)
			delete(webglFP, "webGl2Enabled")
			MergeInto(config, webglFP)
			mergeFirefoxPrefs(firefoxUserPrefs, map[string]interface{}{
				"webgl.enable-webgl2": enableWebGL2,
				"webgl.force-enabled": true,
			})
		}
	}

	// Cache
	if opts.EnableCache {
		for k, v := range CachePrefs {
			if _, ok := firefoxUserPrefs[k]; !ok {
				firefoxUserPrefs[k] = v
			}
		}
	}

	// Debug
	if opts.Debug {
		fmt.Println("[DEBUG] Config:")
		data, _ := json.MarshalIndent(config, "", "  ")
		fmt.Println(string(data))
	}

	// Validate config
	if err := ValidateConfig(config, opts.ExecutablePath); err != nil {
		return nil, err
	}

	// Build env vars
	envVars, err := GetEnvVars(config, targetOS)
	if err != nil {
		return nil, err
	}
	for k, v := range env {
		envVars[k] = v
	}

	// Resolve executable path
	execPath := opts.ExecutablePath
	if execPath == "" {
		if opts.Browser != "" {
			browserPath := FindInstalledVersion(opts.Browser)
			if browserPath == "" {
				return nil, fmt.Errorf("browser version '%s' not found", opts.Browser)
			}
			p, err := LaunchPath(browserPath)
			if err != nil {
				return nil, err
			}
			execPath = p
		} else {
			p, err := LaunchPath("")
			if err != nil {
				return nil, err
			}
			execPath = p
		}
	}

	result := &LaunchOptionsResult{
		ExecutablePath:   execPath,
		Args:             args,
		Env:              envVars,
		FirefoxUserPrefs: firefoxUserPrefs,
		Headless:         headless,
	}
	if len(opts.Proxy) > 0 {
		result.Proxy = opts.Proxy
	}
	return result, nil
}

// ToPlaywrightLaunchOptions converts LaunchOptionsResult to Playwright launch options.
func (r *LaunchOptionsResult) ToPlaywrightLaunchOptions() playwright.BrowserTypeLaunchOptions {
	opts := playwright.BrowserTypeLaunchOptions{
		ExecutablePath: playwright.String(r.ExecutablePath),
		Headless:       playwright.Bool(r.Headless),
		Args:           r.Args,
	}
	if len(r.FirefoxUserPrefs) > 0 {
		opts.FirefoxUserPrefs = r.FirefoxUserPrefs
	}
	if len(r.Env) > 0 {
		opts.Env = r.envAsMapStringString()
	}
	if len(r.Proxy) > 0 {
		p := &playwright.Proxy{Server: r.Proxy["server"]}
		if u, ok := r.Proxy["username"]; ok {
			p.Username = playwright.String(u)
		}
		if pw, ok := r.Proxy["password"]; ok {
			p.Password = playwright.String(pw)
		}
		opts.Proxy = p
	}
	return opts
}

// ToPlaywrightLaunchOptionsPtr returns a pointer to the Playwright launch options.
func (r *LaunchOptionsResult) ToPlaywrightLaunchOptionsPtr() *playwright.BrowserTypeLaunchOptions {
	opts := r.ToPlaywrightLaunchOptions()
	return &opts
}

func (r *LaunchOptionsResult) envAsMapStringString() map[string]string {
	return r.Env
}

func setGeoIPConfig(config, firefoxUserPrefs map[string]interface{}, ip, geoipDB string, blockWebRTC bool) error {
	if !blockWebRTC {
		if ValidIPv4(ip) {
			SetInto(config, "webrtc:ipv4", ip)
			firefoxUserPrefs["network.dns.disableIPv6"] = true
		} else if ValidIPv6(ip) {
			SetInto(config, "webrtc:ipv6", ip)
		}
	}
	geo, err := GetGeolocation(ip, geoipDB)
	if err != nil {
		return err
	}
	for k, v := range geo.GeoAsConfig() {
		config[k] = v
	}
	return nil
}

func mergeFirefoxPrefs(target, source map[string]interface{}) {
	for k, v := range source {
		if _, ok := target[k]; !ok {
			target[k] = v
		}
	}
}

// sampleWebGLHelper wraps the webgl.SampleWebGL call.
func sampleWebGLHelper(targetOS string, vendor, renderer *string) (map[string]interface{}, error) {
	result, err := webgl.SampleWebGL(targetOS, vendor, renderer)
	if err != nil {
		return nil, err
	}
	// Convert map[string]any to map[string]interface{}
	out := make(map[string]interface{}, len(result))
	for k, v := range result {
		out[k] = v
	}
	return out, nil
}

// newZipReader creates a zip.Reader from raw bytes.
func newZipReader(data []byte) (*zip.Reader, error) {
	return zip.NewReader(bytes.NewReader(data), int64(len(data)))
}

// copyLimited copies up to limit bytes from src to dst.
func copyLimited(dst io.Writer, src io.Reader, limit uint64) (int64, error) {
	return io.Copy(dst, io.LimitReader(src, int64(limit)))
}
