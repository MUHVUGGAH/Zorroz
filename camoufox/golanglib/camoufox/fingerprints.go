package camoufox

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/MUHVUGGAH/zorroz/camoufox/golanglib/camoufox/internal/browserforge/fingerprints"
	"github.com/MUHVUGGAH/zorroz/camoufox/golanglib/camoufox/internal/browserforge/headers"

	"github.com/playwright-community/playwright-go"
)

var (
	firefoxVersionPattern   = regexp.MustCompile(`Firefox/\d+\.0`)
	firefoxRVPattern        = regexp.MustCompile(`rv:\d+\.0`)
	bfVersionReplacePattern = regexp.MustCompile(`(?:^|\D)(1[0-9]{2})(\.0)(?:\D|$)`)
)

// BrowserForgeDataDir is the directory containing Bayesian network data files.
// Set this before calling GenerateFingerprint or GenerateContextFingerprint.
// Defaults to the BROWSERFORGE_DATA_DIR environment variable, or fallback to common locations.
var BrowserForgeDataDir string

var (
	bfGeneratorOnce sync.Once
	bfGenerator     *fingerprints.FingerprintGenerator
	bfGeneratorErr  error
)

// BrowserForge YAML mapping loaded from browserforge.yml
var (
	bfYAMLOnce sync.Once
	bfYAMLData map[string]interface{}
	bfYAMLErr  error
)

func loadBFYAML() (map[string]interface{}, error) {
	bfYAMLOnce.Do(func() {
		bfYAMLData = make(map[string]interface{})
		bfYAMLErr = LoadYAML("browserforge.yml", &bfYAMLData)
	})
	return bfYAMLData, bfYAMLErr
}

// resolveBFDataDir locates the Bayesian network data files.
func resolveBFDataDir() string {
	if BrowserForgeDataDir != "" {
		return BrowserForgeDataDir
	}
	if dir := os.Getenv("BROWSERFORGE_DATA_DIR"); dir != "" {
		return dir
	}
	// Try common locations relative to the package
	candidates := []string{
		filepath.Join(localCamoufoxPath(""), "data"),
		filepath.Join(localCamoufoxPath(""), "..", "..", "..", "browserforge", "data"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "fingerprint-network-definition.zip")); err == nil {
			return c
		}
	}
	return ""
}

// initBFGenerator initializes the BrowserForge fingerprint generator.
func initBFGenerator() (*fingerprints.FingerprintGenerator, error) {
	bfGeneratorOnce.Do(func() {
		dataDir := resolveBFDataDir()
		if dataDir == "" {
			bfGeneratorErr = fmt.Errorf("BrowserForge data directory not found. Set BrowserForgeDataDir or BROWSERFORGE_DATA_DIR env var")
			return
		}

		config := fingerprints.FingerprintGeneratorConfig{
			FingerprintNetworkPath: filepath.Join(dataDir, "fingerprint-network-definition.zip"),
			HeaderGeneratorConfig: headers.HeaderGeneratorConfig{
				InputNetworkPath:      filepath.Join(dataDir, "input-network-definition.zip"),
				HeaderNetworkPath:     filepath.Join(dataDir, "header-network-definition.zip"),
				HeadersOrderPath:      filepath.Join(dataDir, "headers-order.json"),
				BrowserHelperFilePath: filepath.Join(dataDir, "browser-helper-file.json"),
			},
		}

		bfGenerator, bfGeneratorErr = fingerprints.NewFingerprintGenerator(
			config,
			nil,                                  // screen
			false,                                // strict
			false,                                // mockWebRTC
			false,                                // slim
			[]headers.Browser{{Name: "firefox"}}, // browser filter
			[]string{"linux", "macos", "windows"},
			nil, // device
			nil, // locale
			2,   // httpVersion
			false,
		)
	})
	return bfGenerator, bfGeneratorErr
}

var (
	fingerprintPresetsPath = localCamoufoxPath("fingerprint-presets.json")
	fontsPath              = localCamoufoxPath("fonts.json")
	voicesPath             = localCamoufoxPath("voices.json")
)

var (
	presetsOnce sync.Once
	presetsData *FingerprintPresetFile
	presetsErr  error

	fontsOnce sync.Once
	fontsData map[string][]string
	fontsErr  error

	voicesOnce sync.Once
	voicesData map[string][]string
	voicesErr  error
)

var osToPresetKey = map[string]string{
	"windows": "windows",
	"macos":   "macos",
	"linux":   "linux",
	"win":     "windows",
	"mac":     "macos",
	"lin":     "linux",
}

var (
	macOSMarkerFonts = []string{
		"Helvetica Neue", "PingFang HK", "PingFang SC", "PingFang TC",
	}
	linuxMarkerFonts = []string{
		"Arimo", "Cousine", "Tinos", "Twemoji Mozilla",
	}
	windowsMarkerFonts = []string{
		"Segoe UI", "Tahoma", "Cambria Math", "Nirmala UI",
	}
)

var (
	essentialFontsMacOS = []string{
		"Arial", "Helvetica", "Times New Roman", "Courier New", "Verdana",
		"Georgia", "Trebuchet MS", "Tahoma", "Helvetica Neue", "Lucida Grande",
		"Menlo", "Monaco", "Geneva", "PingFang HK", "PingFang SC", "PingFang TC",
	}
	essentialFontsWindows = []string{
		"Arial", "Times New Roman", "Courier New", "Verdana", "Georgia",
		"Trebuchet MS", "Tahoma", "Segoe UI", "Calibri", "Cambria Math",
		"Nirmala UI", "Consolas",
	}
	essentialFontsLinux = []string{
		"Arimo", "Cousine", "Tinos", "Twemoji Mozilla",
		"Noto Sans Devanagari", "Noto Sans JP", "Noto Sans KR",
		"Noto Sans SC", "Noto Sans TC",
	}
)

var (
	essentialVoicesMacOS = []string{
		"Samantha", "Alex", "Fred", "Victoria", "Karen", "Daniel",
	}
	essentialVoicesWindows = []string{
		"Microsoft David - English (United States)",
		"Microsoft Zira - English (United States)",
		"Microsoft Mark - English (United States)",
	}
)

// FingerprintPresetFile is the bundled preset JSON structure.
type FingerprintPresetFile struct {
	Version     int                            `json:"version"`
	GeneratedAt string                         `json:"generated_at"`
	Presets     map[string][]FingerprintPreset `json:"presets"`
}

// FingerprintPreset is the subset of preset fields Camoufox uses today.
type FingerprintPreset struct {
	Navigator    PresetNavigator `json:"navigator"`
	Screen       PresetScreen    `json:"screen"`
	WebGL        PresetWebGL     `json:"webgl"`
	Fonts        []string        `json:"fonts"`
	SpeechVoices []string        `json:"speechVoices"`
	Timezone     string          `json:"timezone"`
}

type PresetNavigator struct {
	UserAgent           string `json:"userAgent"`
	Platform            string `json:"platform"`
	HardwareConcurrency int    `json:"hardwareConcurrency"`
	Oscpu               string `json:"oscpu"`
	MaxTouchPoints      int    `json:"maxTouchPoints"`
}

type PresetScreen struct {
	Width            int     `json:"width"`
	Height           int     `json:"height"`
	ColorDepth       int     `json:"colorDepth"`
	AvailWidth       int     `json:"availWidth"`
	AvailHeight      int     `json:"availHeight"`
	DevicePixelRatio float64 `json:"devicePixelRatio"`
}

type PresetWebGL struct {
	UnmaskedVendor   string `json:"unmaskedVendor"`
	UnmaskedRenderer string `json:"unmaskedRenderer"`
}

type initScriptValues struct {
	FontSpacingSeed      any
	AudioFingerprintSeed any
	CanvasSeed           any
	NavigatorPlatform    string
	NavigatorOscpu       string
	NavigatorUserAgent   string
	HardwareConcurrency  any
	WebGLVendor          string
	WebGLRenderer        string
	ScreenWidth          int
	ScreenHeight         int
	ScreenColorDepth     int
	Timezone             string
	FontList             []string
	SpeechVoices         []string
	WebRTCIP             string
}

// LoadPresets loads the bundled fingerprint preset file.
func LoadPresets() (*FingerprintPresetFile, error) {
	presetsOnce.Do(func() {
		data, err := osReadFile(fingerprintPresetsPath)
		if err != nil {
			presetsErr = err
			return
		}

		var parsed FingerprintPresetFile
		if err := json.Unmarshal(data, &parsed); err != nil {
			presetsErr = err
			return
		}
		presetsData = &parsed
	})

	return presetsData, presetsErr
}

// GetRandomPreset returns a random preset for the requested OS values.
// Pass no OS values to sample from all bundled presets.
func GetRandomPreset(osNames ...string) (*FingerprintPreset, error) {
	presets, err := LoadPresets()
	if err != nil {
		return nil, err
	}
	if presets == nil || len(presets.Presets) == 0 {
		return nil, nil
	}

	keys := osNames
	if len(keys) == 0 {
		keys = []string{"macos", "windows", "linux"}
	}

	var candidates []FingerprintPreset
	for _, osName := range keys {
		key := osToPresetKey[osName]
		if key == "" {
			key = osName
		}
		candidates = append(candidates, presets.Presets[key]...)
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	preset := candidates[rand.Intn(len(candidates))]
	return &preset, nil
}

// FromPreset converts a bundled real-world preset into Camoufox config keys.
func FromPreset(preset FingerprintPreset, ffVersion *string) map[string]any {
	config := map[string]any{}

	ua := preset.Navigator.UserAgent
	if ua != "" {
		if ffVersion != nil && *ffVersion != "" {
			ua = firefoxVersionPattern.ReplaceAllString(ua, "Firefox/"+*ffVersion+".0")
			ua = firefoxRVPattern.ReplaceAllString(ua, "rv:"+*ffVersion+".0")
		}
		config["navigator.userAgent"] = ua
	}
	if preset.Navigator.Platform != "" {
		config["navigator.platform"] = preset.Navigator.Platform
	}
	if preset.Navigator.HardwareConcurrency != 0 {
		config["navigator.hardwareConcurrency"] = preset.Navigator.HardwareConcurrency
	}
	if preset.Navigator.Oscpu != "" {
		config["navigator.oscpu"] = preset.Navigator.Oscpu
	} else if preset.Navigator.Platform != "" {
		switch preset.Navigator.Platform {
		case "MacIntel":
			config["navigator.oscpu"] = "Intel Mac OS X 10.15"
		case "Win32":
			config["navigator.oscpu"] = "Windows NT 10.0; Win64; x64"
		default:
			if strings.Contains(strings.ToLower(preset.Navigator.Platform), "linux") {
				config["navigator.oscpu"] = "Linux x86_64"
			}
		}
	}
	config["navigator.maxTouchPoints"] = preset.Navigator.MaxTouchPoints

	if preset.Screen.Width != 0 {
		config["screen.width"] = preset.Screen.Width
	}
	if preset.Screen.Height != 0 {
		config["screen.height"] = preset.Screen.Height
	}
	if preset.Screen.ColorDepth != 0 {
		config["screen.colorDepth"] = preset.Screen.ColorDepth
		config["screen.pixelDepth"] = preset.Screen.ColorDepth
	}
	if preset.Screen.AvailWidth != 0 {
		config["screen.availWidth"] = preset.Screen.AvailWidth
	}
	if preset.Screen.AvailHeight != 0 {
		config["screen.availHeight"] = preset.Screen.AvailHeight
	}

	if preset.WebGL.UnmaskedVendor != "" {
		config["webGl:vendor"] = preset.WebGL.UnmaskedVendor
	}
	if preset.WebGL.UnmaskedRenderer != "" {
		config["webGl:renderer"] = preset.WebGL.UnmaskedRenderer
	}

	config["fonts:spacing_seed"] = randomNonZeroUint32()
	config["audio:seed"] = randomNonZeroUint32()
	config["canvas:seed"] = randomNonZeroUint32()

	if preset.Timezone != "" {
		config["timezone"] = preset.Timezone
	}

	targetOS := detectPresetTargetOS(preset.Navigator.Platform)
	if fonts, err := GenerateRandomFontSubset(targetOS); err == nil {
		config["fonts"] = fonts
	} else if len(preset.Fonts) > 0 {
		fonts := append([]string(nil), preset.Fonts...)
		fonts = ensureMarkerFonts(fonts, markerFontsForOS(targetOS))
		config["fonts"] = fonts
	}

	if voices, err := GenerateRandomVoiceSubset(targetOS); err == nil {
		config["voices"] = voices
	} else if len(preset.SpeechVoices) > 0 {
		config["voices"] = append([]string(nil), preset.SpeechVoices...)
	}

	return config
}

// BuildInitScript assembles the addInitScript payload for a generated fingerprint.
func BuildInitScript(values initScriptValues) (string, error) {
	lines := []string{
		"(function(v) {",
		"  var w = window;",
	}

	type setter struct {
		value any
		name  string
	}

	setters := []setter{
		{value: values.FontSpacingSeed, name: "setFontSpacingSeed"},
		{value: values.AudioFingerprintSeed, name: "setAudioFingerprintSeed"},
		{value: values.CanvasSeed, name: "setCanvasSeed"},
		{value: values.NavigatorPlatform, name: "setNavigatorPlatform"},
		{value: values.NavigatorOscpu, name: "setNavigatorOscpu"},
		{value: values.NavigatorUserAgent, name: "setNavigatorUserAgent"},
		{value: values.HardwareConcurrency, name: "setNavigatorHardwareConcurrency"},
		{value: values.WebGLVendor, name: "setWebGLVendor"},
		{value: values.WebGLRenderer, name: "setWebGLRenderer"},
	}

	for _, setter := range setters {
		if isZeroValue(setter.value) {
			continue
		}
		jsValue, err := jsonLiteral(setter.value)
		if err != nil {
			return "", err
		}
		lines = append(lines, fmt.Sprintf(`  if (typeof w.%s === "function") w.%s(%s);`, setter.name, setter.name, jsValue))
	}

	if values.ScreenWidth > 0 && values.ScreenHeight > 0 {
		lines = append(
			lines,
			fmt.Sprintf(`  if (typeof w.setScreenDimensions === "function") w.setScreenDimensions(%d, %d);`, values.ScreenWidth, values.ScreenHeight),
		)
		if values.ScreenColorDepth > 0 {
			lines = append(
				lines,
				fmt.Sprintf(`  if (typeof w.setScreenColorDepth === "function") w.setScreenColorDepth(%d);`, values.ScreenColorDepth),
			)
		}
	}

	if values.Timezone != "" {
		jsValue, err := jsonLiteral(values.Timezone)
		if err != nil {
			return "", err
		}
		lines = append(lines, fmt.Sprintf(`  if (typeof w.setTimezone === "function") w.setTimezone(%s);`, jsValue))
	} else {
		lines = append(lines, `  if (typeof w.setTimezone === "function") w.setTimezone(Intl.DateTimeFormat().resolvedOptions().timeZone);`)
	}

	if values.WebRTCIP != "" {
		jsValue, err := jsonLiteral(values.WebRTCIP)
		if err != nil {
			return "", err
		}
		lines = append(lines, fmt.Sprintf(`  if (typeof w.setWebRTCIPv4 === "function") w.setWebRTCIPv4(%s);`, jsValue))
	} else {
		lines = append(lines, `  if (typeof w.setWebRTCIPv4 === "function") w.setWebRTCIPv4("");`)
	}

	if len(values.FontList) > 0 {
		jsValue, err := jsonLiteral(strings.Join(values.FontList, ","))
		if err != nil {
			return "", err
		}
		lines = append(lines, fmt.Sprintf(`  if (typeof w.setFontList === "function") w.setFontList(%s);`, jsValue))
	}

	if len(values.SpeechVoices) > 0 {
		jsValue, err := jsonLiteral(strings.Join(values.SpeechVoices, ","))
		if err != nil {
			return "", err
		}
		lines = append(lines, fmt.Sprintf(`  if (typeof w.setSpeechVoices === "function") w.setSpeechVoices(%s);`, jsValue))
	}

	lines = append(lines, "})();")
	return strings.Join(lines, "\n"), nil
}

// GenerateContextFingerprint generates fingerprint values for a single per-context identity.
// Returns a ContextFingerprint with init_script (JS string) and context_options (Playwright options).
// By default, uses BrowserForge for infinite unique synthetic fingerprints.
// Pass a preset dict to use a real fingerprint preset instead.
func GenerateContextFingerprint(options GenerateContextFingerprintOptions) (*ContextFingerprint, error) {
	var config map[string]any
	var nav map[string]any
	var screen map[string]any
	var webgl map[string]any
	var timezone string

	if options.Preset != nil {
		// Preset path
		preset, err := decodePreset(options.Preset)
		if err != nil {
			return nil, err
		}

		config = FromPreset(*preset, options.FFVersion)
		nav = map[string]any{
			"platform":            preset.Navigator.Platform,
			"hardwareConcurrency": preset.Navigator.HardwareConcurrency,
		}
		screen = map[string]any{
			"width":            preset.Screen.Width,
			"height":           preset.Screen.Height,
			"colorDepth":       preset.Screen.ColorDepth,
			"devicePixelRatio": preset.Screen.DevicePixelRatio,
		}
		webgl = map[string]any{
			"unmaskedVendor":   preset.WebGL.UnmaskedVendor,
			"unmaskedRenderer": preset.WebGL.UnmaskedRenderer,
		}
		timezone = preset.Timezone
		if tz, ok := config["timezone"].(string); ok && tz != "" {
			timezone = tz
		}
	} else {
		// BrowserForge synthetic generation path
		fp, err := GenerateFingerprint(options.OS, nil)
		if err != nil {
			return nil, fmt.Errorf("BrowserForge fingerprint generation failed: %w", err)
		}

		config = FromBrowserForge(fp, options.FFVersion)

		// Add seeds (BrowserForge doesn't generate these)
		if _, ok := config["fonts:spacing_seed"]; !ok {
			config["fonts:spacing_seed"] = randomNonZeroUint32()
		}
		if _, ok := config["audio:seed"]; !ok {
			config["audio:seed"] = randomNonZeroUint32()
		}
		if _, ok := config["canvas:seed"]; !ok {
			config["canvas:seed"] = randomNonZeroUint32()
		}

		// Determine target OS from platform for font/voice generation
		plat := stringValue(config["navigator.platform"])
		osName := detectPresetTargetOS(plat)

		// Add fonts (BrowserForge doesn't generate these)
		if _, ok := config["fonts"]; !ok {
			if fonts, err := GenerateRandomFontSubset(osName); err == nil {
				config["fonts"] = fonts
			}
		}

		// Add voices (BrowserForge doesn't generate these)
		if _, ok := config["voices"]; !ok {
			if voices, err := GenerateRandomVoiceSubset(osName); err == nil {
				config["voices"] = voices
			}
		}

		// Derive oscpu if BrowserForge didn't provide it
		if _, ok := config["navigator.oscpu"]; !ok {
			switch plat {
			case "MacIntel":
				config["navigator.oscpu"] = "Intel Mac OS X 10.15"
			case "Win32":
				config["navigator.oscpu"] = "Windows NT 10.0; Win64; x64"
			default:
				if strings.Contains(strings.ToLower(plat), "linux") {
					config["navigator.oscpu"] = "Linux x86_64"
				}
			}
		}

		// Sample WebGL vendor/renderer from database (BrowserForge doesn't generate these)
		if stringValue(config["webGl:vendor"]) == "" || stringValue(config["webGl:renderer"]) == "" {
			osMap := map[string]string{"macos": "mac", "linux": "lin", "windows": "win"}
			targetOSKey := ""
			if options.OS != nil {
				targetOSKey = osMap[*options.OS]
			}
			if targetOSKey == "" {
				switch plat {
				case "Win32":
					targetOSKey = "win"
				default:
					if strings.Contains(strings.ToLower(plat), "linux") {
						targetOSKey = "lin"
					} else {
						targetOSKey = "mac"
					}
				}
			}
			webglFP, err := SampleWebGLFromDB(targetOSKey, nil, nil)
			if err == nil && webglFP != nil {
				delete(webglFP, "webGl2Enabled")
				for k, v := range webglFP {
					config[k] = v
				}
			}
		}

		// Build source dicts from BrowserForge config for init_values
		nav = map[string]any{
			"platform":            config["navigator.platform"],
			"hardwareConcurrency": config["navigator.hardwareConcurrency"],
		}
		screen = map[string]any{
			"width":            config["screen.width"],
			"height":           config["screen.height"],
			"colorDepth":       config["screen.colorDepth"],
			"devicePixelRatio": nil,
		}
		webgl = map[string]any{
			"unmaskedVendor":   config["webGl:vendor"],
			"unmaskedRenderer": config["webGl:renderer"],
		}
		timezone, _ = config["timezone"].(string)
	}

	// Build the values dict for the init script (works for both paths)
	voices := toStringSlice(config["voices"])
	fonts := toStringSlice(config["fonts"])
	initScript, err := BuildInitScript(initScriptValues{
		FontSpacingSeed:      config["fonts:spacing_seed"],
		AudioFingerprintSeed: config["audio:seed"],
		CanvasSeed:           config["canvas:seed"],
		NavigatorPlatform:    stringValue(nav["platform"]),
		NavigatorOscpu:       stringValue(config["navigator.oscpu"]),
		NavigatorUserAgent:   stringValue(config["navigator.userAgent"]),
		HardwareConcurrency:  coalesceAny(nav["hardwareConcurrency"], config["navigator.hardwareConcurrency"]),
		WebGLVendor:          stringValue(webgl["unmaskedVendor"]),
		WebGLRenderer:        stringValue(webgl["unmaskedRenderer"]),
		ScreenWidth:          intValue(screen["width"]),
		ScreenHeight:         intValue(screen["height"]),
		ScreenColorDepth:     intValue(screen["colorDepth"]),
		Timezone:             timezone,
		FontList:             fonts,
		SpeechVoices:         voices,
		WebRTCIP:             derefString(options.WebRTCIP),
	})
	if err != nil {
		return nil, err
	}

	// Playwright context options that must be set at context creation
	contextOptions := playwright.BrowserNewContextOptions{}
	if ua := stringValue(config["navigator.userAgent"]); ua != "" {
		contextOptions.UserAgent = playwright.String(ua)
	}
	sw := intValue(screen["width"])
	sh := intValue(screen["height"])
	if sw > 0 && sh > 0 {
		contextOptions.Viewport = &playwright.Size{
			Width:  sw,
			Height: max(sh-28, 600),
		}
	}
	if dpr, ok := screen["devicePixelRatio"].(float64); ok && dpr != 0 {
		contextOptions.DeviceScaleFactor = float64Ptr(dpr)
	}
	if timezone != "" {
		contextOptions.TimezoneId = playwright.String(timezone)
	}

	return &ContextFingerprint{
		InitScript:     initScript,
		ContextOptions: contextOptions,
	}, nil
}

// GenerateFingerprint generates a Firefox fingerprint using BrowserForge.
// If window is non-nil, it specifies the desired (outerWidth, outerHeight).
func GenerateFingerprint(osName *string, window *[2]int) (*fingerprints.Fingerprint, error) {
	gen, err := initBFGenerator()
	if err != nil {
		return nil, err
	}

	fp, err := gen.Generate(nil)
	if err != nil {
		return nil, err
	}

	if window != nil {
		HandleWindowSize(fp, window[0], window[1])
	}

	return fp, nil
}

// FromBrowserForge converts a BrowserForge fingerprint to a Camoufox config.
func FromBrowserForge(fp *fingerprints.Fingerprint, ffVersion *string) map[string]any {
	bfData, err := loadBFYAML()
	if err != nil {
		return map[string]any{}
	}

	// Convert fingerprint to map for generic traversal
	fpBytes, err := json.Marshal(fp)
	if err != nil {
		return map[string]any{}
	}
	var fpMap map[string]any
	if err := json.Unmarshal(fpBytes, &fpMap); err != nil {
		return map[string]any{}
	}

	camoufoxData := map[string]any{}
	castToProperties(camoufoxData, bfData, fpMap, ffVersion)
	handleScreenXY(camoufoxData, &fp.Screen)
	return camoufoxData
}

// castToProperties casts BrowserForge fingerprints to Camoufox config properties.
func castToProperties(camoufoxData map[string]any, castEnum map[string]interface{}, bfDict map[string]any, ffVersion *string) {
	for key, data := range bfDict {
		if data == nil {
			continue
		}
		typeKey, ok := castEnum[key]
		if !ok {
			continue
		}

		// If the mapping value is a nested dict, recurse
		if subMap, ok := typeKey.(map[string]interface{}); ok {
			if subData, ok := data.(map[string]any); ok {
				castToProperties(camoufoxData, subMap, subData, ffVersion)
			}
			continue
		}

		// The mapping value is a string property name
		propName, ok := typeKey.(string)
		if !ok {
			continue
		}

		// Fix negative screen values
		if strings.HasPrefix(propName, "screen.") {
			if num, ok := toFloat(data); ok && num < 0 {
				data = 0
			}
		}

		// Replace Firefox versions with ffVersion
		if ffVersion != nil && *ffVersion != "" {
			if strVal, ok := data.(string); ok {
				data = bfVersionReplacePattern.ReplaceAllStringFunc(strVal, func(match string) string {
					return strings.Replace(match, match[1:4], *ffVersion, 1)
				})
			}
		}

		camoufoxData[propName] = data
	}
}

// handleScreenXY sets window.screenY based on BrowserForge's screenX value.
func handleScreenXY(camoufoxData map[string]any, fpScreen *fingerprints.ScreenFingerprint) {
	if _, ok := camoufoxData["window.screenY"]; ok {
		return
	}

	screenX := fpScreen.ScreenX
	if screenX == 0 {
		camoufoxData["window.screenX"] = 0
		camoufoxData["window.screenY"] = 0
		return
	}

	if screenX >= -50 && screenX <= 50 {
		camoufoxData["window.screenY"] = screenX
		return
	}

	// BrowserForge thinks the browser is windowed. Randomly generate a screenY.
	screenY := fpScreen.AvailHeight - fpScreen.OuterHeight
	if screenY == 0 {
		camoufoxData["window.screenY"] = 0
	} else if screenY > 0 {
		camoufoxData["window.screenY"] = rand.Intn(screenY) //nolint:gosec
	} else {
		camoufoxData["window.screenY"] = -rand.Intn(-screenY) //nolint:gosec
	}
}

// HandleWindowSize sets a custom outer window size and centers it on screen.
func HandleWindowSize(fp *fingerprints.Fingerprint, outerWidth, outerHeight int) {
	sc := &fp.Screen

	// Center the window on the screen
	sc.ScreenX += (sc.Width - outerWidth) / 2

	// Calculate screenY
	screenY := (sc.Height - outerHeight) / 2

	// Update inner dimensions if set
	if sc.InnerWidth > 0 {
		sc.InnerWidth = max(outerWidth-sc.OuterWidth+sc.InnerWidth, 0)
	}
	if sc.InnerHeight > 0 {
		sc.InnerHeight = max(outerHeight-sc.OuterHeight+sc.InnerHeight, 0)
	}

	// Set outer dimensions
	sc.OuterWidth = outerWidth
	sc.OuterHeight = outerHeight
	_ = screenY // screenY is applied later by handleScreenXY
}

// coalesceAny returns the first non-nil, non-zero value.
func coalesceAny(values ...any) any {
	for _, v := range values {
		if !isZeroValue(v) {
			return v
		}
	}
	return nil
}

// toFloat tries to convert a value to float64.
func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

// GenerateRandomFontSubset ports the preset/font-subset path from fingerprints.py.
func GenerateRandomFontSubset(targetOS string) ([]string, error) {
	osFontsData, err := loadOSFonts()
	if err != nil {
		return nil, err
	}

	osKey := map[string]string{
		"macos":   "mac",
		"windows": "win",
		"linux":   "lin",
	}[targetOS]
	if osKey == "" {
		osKey = "mac"
	}

	fullList := append([]string(nil), osFontsData[osKey]...)
	if len(fullList) == 0 {
		return nil, fmt.Errorf("no font list found for OS %q", targetOS)
	}

	var essential map[string]struct{}
	var markers []string
	switch targetOS {
	case "windows":
		essential = stringSet(essentialFontsWindows)
		markers = windowsMarkerFonts
	case "linux":
		essential = stringSet(essentialFontsLinux)
		markers = linuxMarkerFonts
	default:
		essential = stringSet(essentialFontsMacOS)
		markers = macOSMarkerFonts
	}

	result := make([]string, 0, len(fullList))
	nonEssential := make([]string, 0, len(fullList))
	for _, font := range fullList {
		if _, ok := essential[font]; ok {
			result = append(result, font)
		} else {
			nonEssential = append(nonEssential, font)
		}
	}

	pct := 30 + rand.Intn(49)
	count := int(math.Round((float64(pct) / 100) * float64(len(nonEssential))))
	result = append(result, pickRandomSubset(nonEssential, count)...)
	result = ensureMarkerFonts(result, markers)
	return result, nil
}

// GenerateRandomVoiceSubset ports the preset/voice-subset path from fingerprints.py.
func GenerateRandomVoiceSubset(targetOS string) ([]string, error) {
	osVoicesData, err := loadOSVoices()
	if err != nil {
		return nil, err
	}

	osKey := map[string]string{
		"macos":   "mac",
		"windows": "win",
		"linux":   "lin",
	}[targetOS]
	if osKey == "" {
		osKey = "mac"
	}

	fullList := append([]string(nil), osVoicesData[osKey]...)
	if len(fullList) == 0 {
		return []string{}, nil
	}
	if targetOS == "windows" {
		return fullList, nil
	}
	if targetOS == "linux" {
		return []string{}, nil
	}

	essential := stringSet(essentialVoicesMacOS)
	result := make([]string, 0, len(fullList))
	nonEssential := make([]string, 0, len(fullList))
	for _, voice := range fullList {
		if _, ok := essential[voice]; ok {
			result = append(result, voice)
		} else {
			nonEssential = append(nonEssential, voice)
		}
	}

	pct := 40 + rand.Intn(41)
	count := int(math.Round((float64(pct) / 100) * float64(len(nonEssential))))
	result = append(result, pickRandomSubset(nonEssential, count)...)
	return result, nil
}

func loadOSFonts() (map[string][]string, error) {
	fontsOnce.Do(func() {
		data, err := osReadFile(fontsPath)
		if err != nil {
			fontsErr = err
			return
		}
		if err := json.Unmarshal(data, &fontsData); err != nil {
			fontsErr = err
		}
	})
	return fontsData, fontsErr
}

func loadOSVoices() (map[string][]string, error) {
	voicesOnce.Do(func() {
		data, err := osReadFile(voicesPath)
		if err != nil {
			voicesErr = err
			return
		}

		raw := map[string][]string{}
		if err := json.Unmarshal(data, &raw); err != nil {
			voicesErr = err
			return
		}

		voicesData = make(map[string][]string, len(raw))
		for osKey, entries := range raw {
			voices := make([]string, 0, len(entries))
			for _, entry := range entries {
				parts := strings.SplitN(entry, ":", 2)
				voices = append(voices, parts[0])
			}
			voicesData[osKey] = voices
		}
	})
	return voicesData, voicesErr
}

func decodePreset(data map[string]any) (*FingerprintPreset, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var preset FingerprintPreset
	if err := json.Unmarshal(raw, &preset); err != nil {
		return nil, err
	}
	return &preset, nil
}

func detectPresetTargetOS(platform string) string {
	switch platform {
	case "MacIntel":
		return "macos"
	case "Win32":
		return "windows"
	default:
		if strings.Contains(strings.ToLower(platform), "linux") {
			return "linux"
		}
		return "macos"
	}
}

func markerFontsForOS(targetOS string) []string {
	switch targetOS {
	case "windows":
		return windowsMarkerFonts
	case "linux":
		return linuxMarkerFonts
	default:
		return macOSMarkerFonts
	}
}

func ensureMarkerFonts(fonts []string, markers []string) []string {
	existing := stringSet(fonts)
	for _, marker := range markers {
		if _, ok := existing[marker]; ok {
			continue
		}
		fonts = append(fonts, marker)
		existing[marker] = struct{}{}
	}
	return fonts
}

func pickRandomSubset(values []string, count int) []string {
	if count <= 0 || len(values) == 0 {
		return nil
	}
	if count >= len(values) {
		return append([]string(nil), values...)
	}

	shuffled := append([]string(nil), values...)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	return shuffled[:count]
}

func localCamoufoxPath(name string) string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return name
	}
	return filepath.Join(filepath.Dir(file), name)
}

func osReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func jsonLiteral(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func isZeroValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return typed == ""
	case int:
		return typed == 0
	case int32:
		return typed == 0
	case int64:
		return typed == 0
	case uint32:
		return typed == 0
	case float64:
		return typed == 0
	default:
		return false
	}
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func intValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func toStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func randomNonZeroUint32() uint32 {
	const maxUint32 = uint64(^uint32(0))
	for {
		value := uint32(rand.Int63n(int64(maxUint32 + 1)))
		if value != 0 {
			return value
		}
	}
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func float64Ptr(value float64) *float64 {
	return &value
}

func stringSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

// SampleWebGLFromDB wraps the webgl sub-package for use within the main package.
// This is used by the BrowserForge path in GenerateContextFingerprint.
func SampleWebGLFromDB(osName string, vendor, renderer *string) (map[string]any, error) {
	return sampleWebGLHelper(osName, vendor, renderer)
}
