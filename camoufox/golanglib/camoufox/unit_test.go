package camoufox

import (
	"errors"
	"testing"
)

// ==========================================================================
// version.go
// ==========================================================================

func TestConstraintsRange(t *testing.T) {
	r := ConstraintsRange()
	if r == "" {
		t.Fatal("ConstraintsRange returned empty string")
	}
	if r != ">="+CONSTRAINTS.MinVersion+", <"+CONSTRAINTS.MaxVersion {
		t.Fatalf("unexpected range: %s", r)
	}
}

// ==========================================================================
// pkgman.go — Version
// ==========================================================================

func TestNewVersion(t *testing.T) {
	v := NewVersion("135.0.1", "135.0.1")
	if v.Build != "135.0.1" {
		t.Fatalf("expected Build=135.0.1, got %s", v.Build)
	}
	if v.VersionStr != "135.0.1" {
		t.Fatalf("expected VersionStr=135.0.1, got %s", v.VersionStr)
	}
}

func TestVersionFullString(t *testing.T) {
	v := NewVersion("beta.24", "135.0.1")
	want := "135.0.1-beta.24"
	if got := v.FullString(); got != want {
		t.Fatalf("FullString: got %q, want %q", got, want)
	}
}

func TestVersionLess(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"130.0.0", "135.0.0", true},
		{"135.0.0", "130.0.0", false},
		{"135.0.0", "135.0.0", false},
		{"135.0.0", "135.0.1", true},
		{"135.1.0", "135.0.1", false},
	}
	for _, tt := range tests {
		va := NewVersion(tt.a, "")
		vb := NewVersion(tt.b, "")
		if got := va.Less(vb); got != tt.want {
			t.Errorf("(%s).Less(%s) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestVersionEqual(t *testing.T) {
	a := NewVersion("135.0.1", "")
	b := NewVersion("135.0.1", "")
	if !a.Equal(b) {
		t.Fatal("equal versions reported as not equal")
	}
	c := NewVersion("135.0.2", "")
	if a.Equal(c) {
		t.Fatal("different versions reported as equal")
	}
}

func TestVersionLessOrEqual(t *testing.T) {
	a := NewVersion("135.0.1", "")
	b := NewVersion("135.0.1", "")
	if !a.LessOrEqual(b) {
		t.Fatal("equal versions: LessOrEqual should be true")
	}
	c := NewVersion("136.0.0", "")
	if !a.LessOrEqual(c) {
		t.Fatal("lesser version: LessOrEqual should be true")
	}
	if c.LessOrEqual(a) {
		t.Fatal("greater version: LessOrEqual should be false")
	}
}

func TestVersionIsSupported(t *testing.T) {
	// A version within the constraint range should be supported
	v := NewVersion("alpha.5", "135.0")
	if !v.IsSupported() {
		t.Fatalf("version %s should be supported (range: %s)", v.FullString(), ConstraintsRange())
	}
}

func TestVersionMinMax(t *testing.T) {
	min := VersionMin()
	max := VersionMax()
	if !min.Less(max) {
		t.Fatal("VersionMin should be less than VersionMax")
	}
}

func TestAvailableVersionDisplay(t *testing.T) {
	av := AvailableVersion{
		Version:      NewVersion("beta.24", "135.0.1"),
		IsPrerelease: true,
	}
	got := av.Display()
	if got != "v135.0.1-beta.24 (prerelease)" {
		t.Fatalf("Display: got %q", got)
	}
	av.IsPrerelease = false
	got = av.Display()
	if got != "v135.0.1-beta.24" {
		t.Fatalf("Display (non-prerelease): got %q", got)
	}
}

func TestAvailableVersionToMetadata(t *testing.T) {
	id := 42
	sz := 1024
	av := AvailableVersion{
		Version:        NewVersion("beta.24", "135.0.1"),
		IsPrerelease:   true,
		AssetID:        &id,
		AssetSize:      &sz,
		AssetUpdatedAt: "2026-01-01",
	}
	m := av.ToMetadata()
	if m["version"] != "135.0.1" {
		t.Fatalf("metadata version: %v", m["version"])
	}
	if m["build"] != "beta.24" {
		t.Fatalf("metadata build: %v", m["build"])
	}
	if m["prerelease"] != true {
		t.Fatal("metadata prerelease should be true")
	}
	if m["asset_id"] != 42 {
		t.Fatalf("metadata asset_id: %v", m["asset_id"])
	}
}

// ==========================================================================
// pkgman.go — OSName, InstallDir
// ==========================================================================

func TestOSName(t *testing.T) {
	name := OSName()
	if name == "" {
		t.Fatal("OSName returned empty string")
	}
	valid := map[string]bool{"win": true, "mac": true, "lin": true}
	if !valid[name] {
		t.Fatalf("unexpected OSName: %s", name)
	}
}

func TestInstallDir(t *testing.T) {
	dir := InstallDir()
	if dir == "" {
		t.Fatal("InstallDir returned empty string")
	}
	// On Windows, it should contain the platformdirs-style path
	if OSName() == "win" {
		if !contains(dir, "camoufox") {
			t.Fatalf("InstallDir on Windows should contain 'camoufox': %s", dir)
		}
		if !contains(dir, "Cache") {
			t.Fatalf("InstallDir on Windows should contain 'Cache': %s", dir)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsSubstring(s, sub))
}
func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// ==========================================================================
// exceptions.go
// ==========================================================================

func TestSentinelErrors(t *testing.T) {
	// Verify error wrapping works
	if !errors.Is(ErrCannotFindXvfb, ErrVirtualDisplay) {
		t.Fatal("ErrCannotFindXvfb should wrap ErrVirtualDisplay")
	}
	if !errors.Is(ErrCannotExecuteXvfb, ErrVirtualDisplay) {
		t.Fatal("ErrCannotExecuteXvfb should wrap ErrVirtualDisplay")
	}
	if !errors.Is(ErrVirtualDisplayNotSupported, ErrVirtualDisplay) {
		t.Fatal("ErrVirtualDisplayNotSupported should wrap ErrVirtualDisplay")
	}
}

func TestInvalidLocaleError(t *testing.T) {
	err := InvalidLocaleError("xyz")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrInvalidLocale) {
		t.Fatal("should wrap ErrInvalidLocale")
	}
	if !containsSubstring(err.Error(), "xyz") {
		t.Fatalf("error should mention locale: %s", err.Error())
	}
}

// ==========================================================================
// ip.go
// ==========================================================================

func TestValidIPv4(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		{"192.168.1.1", true},
		{"0.0.0.0", true},
		{"255.255.255.255", true},
		{"not-an-ip", false},
		{"192.168.1", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := ValidIPv4(tt.ip); got != tt.want {
			t.Errorf("ValidIPv4(%q) = %v, want %v", tt.ip, got, tt.want)
		}
	}
}

func TestValidIPv6(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		{"::1", true},
		{"2001:db8::1", true},
		{"fe80::1", true},
		{"not-ipv6", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := ValidIPv6(tt.ip); got != tt.want {
			t.Errorf("ValidIPv6(%q) = %v, want %v", tt.ip, got, tt.want)
		}
	}
}

func TestValidateIP(t *testing.T) {
	if err := ValidateIP("192.168.1.1"); err != nil {
		t.Fatalf("valid IPv4 should pass: %v", err)
	}
	if err := ValidateIP("::1"); err != nil {
		t.Fatalf("valid IPv6 should pass: %v", err)
	}
	if err := ValidateIP("garbage"); err == nil {
		t.Fatal("invalid IP should fail")
	}
	if err := ValidateIP("garbage"); !errors.Is(err, ErrInvalidIP) {
		t.Fatalf("should wrap ErrInvalidIP: %v", err)
	}
}

func TestProxyParseServer(t *testing.T) {
	tests := []struct {
		server     string
		wantSchema string
		wantHost   string
		wantPort   string
	}{
		{"http://proxy.example.com:8080", "http", "proxy.example.com", "8080"},
		{"socks5://127.0.0.1:1080", "socks5", "127.0.0.1", "1080"},
		{"proxy.example.com:3128", "", "proxy.example.com", "3128"},
	}
	for _, tt := range tests {
		p := CamoufoxProxy{Server: tt.server}
		schema, host, port, err := p.ParseServer()
		if err != nil {
			t.Fatalf("ParseServer(%q) error: %v", tt.server, err)
		}
		if schema != tt.wantSchema {
			t.Errorf("ParseServer(%q) schema=%q, want %q", tt.server, schema, tt.wantSchema)
		}
		if host != tt.wantHost {
			t.Errorf("ParseServer(%q) host=%q, want %q", tt.server, host, tt.wantHost)
		}
		if port != tt.wantPort {
			t.Errorf("ParseServer(%q) port=%q, want %q", tt.server, port, tt.wantPort)
		}
	}
}

func TestProxyAsString(t *testing.T) {
	p := CamoufoxProxy{
		Server:   "http://proxy.example.com:8080",
		Username: "user",
		Password: "pass",
	}
	got := p.AsString()
	want := "http://user:pass@proxy.example.com:8080"
	if got != want {
		t.Fatalf("AsString: got %q, want %q", got, want)
	}
}

func TestProxyAsStringNoAuth(t *testing.T) {
	p := CamoufoxProxy{Server: "socks5://127.0.0.1:1080"}
	got := p.AsString()
	want := "socks5://127.0.0.1:1080"
	if got != want {
		t.Fatalf("AsString: got %q, want %q", got, want)
	}
}

// ==========================================================================
// server.go
// ==========================================================================

func TestCamelCase(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"hello_world", "helloWorld"},
		{"snake_case_here", "snakeCaseHere"},
		{"already", "already"},
		{"a", "a"},
		{"", ""},
		{"UPPER_CASE", "upperCase"},
	}
	for _, tt := range tests {
		if got := CamelCase(tt.input); got != tt.want {
			t.Errorf("CamelCase(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToCamelCaseDict(t *testing.T) {
	input := map[string]interface{}{
		"user_name": "alice",
		"age_years": 30,
	}
	got := ToCamelCaseDict(input)
	if got["userName"] != "alice" {
		t.Errorf("expected 'userName' key, got keys: %v", got)
	}
	if got["ageYears"] != 30 {
		t.Errorf("expected 'ageYears'=30, got: %v", got)
	}
}

// ==========================================================================
// utils.go — config helpers
// ==========================================================================

func TestMergeInto(t *testing.T) {
	target := map[string]interface{}{"a": 1, "b": 2}
	source := map[string]interface{}{"b": 99, "c": 3}
	MergeInto(target, source)
	if target["a"] != 1 {
		t.Fatal("a should stay 1")
	}
	if target["b"] != 2 {
		t.Fatal("b should not be overwritten")
	}
	if target["c"] != 3 {
		t.Fatal("c should be added")
	}
}

func TestSetInto(t *testing.T) {
	target := map[string]interface{}{"existing": "yes"}
	SetInto(target, "existing", "no")
	if target["existing"] != "yes" {
		t.Fatal("SetInto should not overwrite existing keys")
	}
	SetInto(target, "new", "value")
	if target["new"] != "value" {
		t.Fatal("SetInto should add new keys")
	}
}

func TestIsDomainSet(t *testing.T) {
	config := map[string]interface{}{
		"navigator.userAgent": "ua",
		"screen.width":        1920,
		"geolocation:lat":     51.5,
	}
	if !IsDomainSet(config, "navigator.") {
		t.Fatal("should detect prefix domain 'navigator.'")
	}
	if !IsDomainSet(config, "geolocation:") {
		t.Fatal("should detect prefix domain 'geolocation:'")
	}
	if !IsDomainSet(config, "screen.width") {
		t.Fatal("should detect exact key 'screen.width'")
	}
	if IsDomainSet(config, "fonts") {
		t.Fatal("should not detect missing key 'fonts'")
	}
}

func TestDetermineUAOS(t *testing.T) {
	tests := []struct {
		ua   string
		want string
	}{
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64)", "win"},
		{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15)", "mac"},
		{"Mozilla/5.0 (X11; Linux x86_64)", "lin"},
		{"Mozilla/5.0 (X11; Ubuntu; Linux x86_64)", "lin"},
	}
	for _, tt := range tests {
		if got := DetermineUAOS(tt.ua); got != tt.want {
			t.Errorf("DetermineUAOS(%q) = %q, want %q", tt.ua, got, tt.want)
		}
	}
}

func TestCheckValidOS(t *testing.T) {
	// Valid OS values
	for _, os := range []string{"windows", "macos", "linux"} {
		if err := CheckValidOS(os); err != nil {
			t.Errorf("CheckValidOS(%q) should pass: %v", os, err)
		}
	}
	// Invalid: uppercase
	if err := CheckValidOS("Windows"); err == nil {
		t.Fatal("uppercase OS should fail")
	}
	// Invalid: unsupported
	if err := CheckValidOS("freebsd"); err == nil {
		t.Fatal("unsupported OS should fail")
	}
	// Valid: slice
	if err := CheckValidOS([]string{"windows", "linux"}); err != nil {
		t.Fatalf("valid OS slice should pass: %v", err)
	}
}

func TestGetEnvVars(t *testing.T) {
	config := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}
	envVars, err := GetEnvVars(config, "win")
	if err != nil {
		t.Fatalf("GetEnvVars error: %v", err)
	}
	if len(envVars) == 0 {
		t.Fatal("GetEnvVars returned empty map")
	}
	// First chunk should exist
	if _, ok := envVars["CAMOU_CONFIG_1"]; !ok {
		t.Fatal("expected CAMOU_CONFIG_1 key")
	}
}

func TestGetTargetOS(t *testing.T) {
	config := map[string]interface{}{
		"navigator.userAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15)",
	}
	if got := GetTargetOS(config); got != "mac" {
		t.Fatalf("expected 'mac', got %q", got)
	}
	// Without userAgent, should return current OS
	if got := GetTargetOS(map[string]interface{}{}); got == "" {
		t.Fatal("GetTargetOS with empty config should return current OS")
	}
}

func TestValidateType(t *testing.T) {
	tests := []struct {
		value    interface{}
		expected string
		want     bool
	}{
		{"hello", "str", true},
		{42, "str", false},
		{42, "int", true},
		{42.0, "int", true},
		{42.5, "int", false},
		{-1, "uint", false},
		{0, "uint", true},
		{3.14, "double", true},
		{42, "double", true},
		{true, "bool", true},
		{"yes", "bool", false},
		{[]interface{}{"a"}, "array", true},
		{[]string{"a"}, "array", true},
		{map[string]interface{}{}, "dict", true},
	}
	for _, tt := range tests {
		if got := validateType(tt.value, tt.expected); got != tt.want {
			t.Errorf("validateType(%v, %q) = %v, want %v", tt.value, tt.expected, got, tt.want)
		}
	}
}

// ==========================================================================
// utils.go — ToPlaywrightLaunchOptions
// ==========================================================================

func TestToPlaywrightLaunchOptions(t *testing.T) {
	r := &LaunchOptionsResult{
		ExecutablePath: "/path/to/firefox",
		Headless:       true,
		Args:           []string{"--kiosk"},
		Env:            map[string]string{"FOO": "bar"},
		Proxy:          map[string]string{"server": "http://proxy:8080"},
	}
	opts := r.ToPlaywrightLaunchOptions()
	if *opts.ExecutablePath != "/path/to/firefox" {
		t.Fatal("ExecutablePath mismatch")
	}
	if *opts.Headless != true {
		t.Fatal("Headless mismatch")
	}
	if len(opts.Args) != 1 || opts.Args[0] != "--kiosk" {
		t.Fatal("Args mismatch")
	}
	if opts.Proxy == nil || opts.Proxy.Server != "http://proxy:8080" {
		t.Fatal("Proxy mismatch")
	}
}

func TestToPlaywrightLaunchOptionsPtr(t *testing.T) {
	r := &LaunchOptionsResult{ExecutablePath: "/bin/ff", Headless: false}
	ptr := r.ToPlaywrightLaunchOptionsPtr()
	if ptr == nil {
		t.Fatal("ToPlaywrightLaunchOptionsPtr returned nil")
	}
}

// ==========================================================================
// locales.go — pure logic
// ==========================================================================

func TestLocaleAsString(t *testing.T) {
	l := Locale{Language: "en", Region: "US"}
	if got := l.AsString(); got != "en-US" {
		t.Fatalf("got %q, want 'en-US'", got)
	}
	l2 := Locale{Language: "fr"}
	if got := l2.AsString(); got != "fr" {
		t.Fatalf("got %q, want 'fr'", got)
	}
}

func TestLocaleAsConfig(t *testing.T) {
	l := Locale{Language: "en", Region: "US", Script: "Latn"}
	cfg := l.AsConfig()
	if cfg["locale:language"] != "en" {
		t.Fatal("language mismatch")
	}
	if cfg["locale:region"] != "US" {
		t.Fatal("region mismatch")
	}
	if cfg["locale:script"] != "Latn" {
		t.Fatal("script mismatch")
	}
}

func TestGeolocationGeoAsConfig(t *testing.T) {
	g := Geolocation{
		Loc:       Locale{Language: "en", Region: "GB"},
		Longitude: -0.1276,
		Latitude:  51.5074,
		Timezone:  "Europe/London",
		Accuracy:  100,
	}
	cfg := g.GeoAsConfig()
	if cfg["geolocation:longitude"] != -0.1276 {
		t.Fatal("longitude mismatch")
	}
	if cfg["timezone"] != "Europe/London" {
		t.Fatal("timezone mismatch")
	}
	if cfg["geolocation:accuracy"] != 100.0 {
		t.Fatal("accuracy mismatch")
	}
}

func TestVerifyLocale(t *testing.T) {
	if err := VerifyLocale("en-US"); err != nil {
		t.Fatalf("valid locale should pass: %v", err)
	}
	if err := VerifyLocale("en"); err != nil {
		t.Fatalf("language-only should pass: %v", err)
	}
	if err := VerifyLocale("x"); err == nil {
		t.Fatal("single char locale should fail")
	}
	if err := VerifyLocale(""); err == nil {
		t.Fatal("empty locale should fail")
	}
}

func TestNormalizeLocale(t *testing.T) {
	tests := []struct {
		input    string
		wantLang string
		wantReg  string
		wantErr  bool
	}{
		{"en-US", "en", "US", false},
		{"en-Latn-US", "en", "US", false},
		{"fr-FR", "fr", "FR", false},
		{"en", "", "", true}, // single part without region — error in current logic
	}
	for _, tt := range tests {
		l, err := NormalizeLocale(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("NormalizeLocale(%q) should error", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("NormalizeLocale(%q) error: %v", tt.input, err)
			continue
		}
		if l.Language != tt.wantLang {
			t.Errorf("NormalizeLocale(%q).Language = %q, want %q", tt.input, l.Language, tt.wantLang)
		}
		if l.Region != tt.wantReg {
			t.Errorf("NormalizeLocale(%q).Region = %q, want %q", tt.input, l.Region, tt.wantReg)
		}
	}
}

func TestHandleLocales(t *testing.T) {
	config := map[string]interface{}{}
	err := HandleLocales("en-US, fr-FR", config)
	if err != nil {
		t.Fatalf("HandleLocales error: %v", err)
	}
	if config["locale:language"] != "en" {
		t.Fatalf("expected locale language 'en', got %v", config["locale:language"])
	}
	if config["locale:region"] != "US" {
		t.Fatalf("expected locale region 'US', got %v", config["locale:region"])
	}
	// With multiple locales, locale:all should be set
	if _, ok := config["locale:all"]; !ok {
		t.Fatal("expected locale:all to be set for multiple locales")
	}
}

// ==========================================================================
// addons.go
// ==========================================================================

func TestAllDefaultAddons(t *testing.T) {
	addons := AllDefaultAddons()
	if _, ok := addons["UBO"]; !ok {
		t.Fatal("expected UBO addon")
	}
}

func TestAddonsDir(t *testing.T) {
	dir := AddonsDir()
	if dir == "" {
		t.Fatal("AddonsDir returned empty string")
	}
}

func TestConfirmPathsInvalidDir(t *testing.T) {
	err := ConfirmPaths([]string{"/nonexistent/path/abc123"})
	if err == nil {
		t.Fatal("should error for nonexistent path")
	}
	if !errors.Is(err, ErrInvalidAddonPath) {
		t.Fatalf("should wrap ErrInvalidAddonPath: %v", err)
	}
}

// ==========================================================================
// fingerprints.go — pure logic helpers
// ==========================================================================

func TestFromPreset(t *testing.T) {
	preset := FingerprintPreset{
		Navigator: PresetNavigator{
			UserAgent:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:135.0) Gecko/20100101 Firefox/135.0",
			Platform:            "Win32",
			HardwareConcurrency: 8,
			MaxTouchPoints:      0,
		},
		Screen: PresetScreen{
			Width:      1920,
			Height:     1080,
			ColorDepth: 24,
		},
		WebGL: PresetWebGL{
			UnmaskedVendor:   "Google Inc.",
			UnmaskedRenderer: "ANGLE (Intel)",
		},
		Timezone: "America/New_York",
	}
	config := FromPreset(preset, nil)
	if config["navigator.userAgent"] != preset.Navigator.UserAgent {
		t.Fatal("userAgent mismatch")
	}
	if config["navigator.platform"] != "Win32" {
		t.Fatal("platform mismatch")
	}
	if config["screen.width"] != 1920 {
		t.Fatal("screen.width mismatch")
	}
	if config["webGl:vendor"] != "Google Inc." {
		t.Fatal("webGl:vendor mismatch")
	}
	if config["timezone"] != "America/New_York" {
		t.Fatal("timezone mismatch")
	}
	// Seeds should be non-zero
	if config["fonts:spacing_seed"] == nil || config["fonts:spacing_seed"] == 0 {
		t.Fatal("fonts:spacing_seed should be set")
	}
	if config["audio:seed"] == nil || config["audio:seed"] == 0 {
		t.Fatal("audio:seed should be set")
	}
	if config["canvas:seed"] == nil || config["canvas:seed"] == 0 {
		t.Fatal("canvas:seed should be set")
	}
}

func TestFromPresetWithFFVersion(t *testing.T) {
	preset := FingerprintPreset{
		Navigator: PresetNavigator{
			UserAgent: "Mozilla/5.0 (Windows NT 10.0; rv:135.0) Gecko/20100101 Firefox/135.0",
		},
	}
	ver := "140"
	config := FromPreset(preset, &ver)
	ua, ok := config["navigator.userAgent"].(string)
	if !ok {
		t.Fatal("userAgent should be a string")
	}
	if !containsSubstring(ua, "Firefox/140.0") {
		t.Fatalf("expected Firefox/140.0 in UA, got: %s", ua)
	}
	if !containsSubstring(ua, "rv:140.0") {
		t.Fatalf("expected rv:140.0 in UA, got: %s", ua)
	}
}

func TestFromPresetOscpuDerivation(t *testing.T) {
	tests := []struct {
		platform  string
		wantOscpu string
	}{
		{"Win32", "Windows NT 10.0; Win64; x64"},
		{"MacIntel", "Intel Mac OS X 10.15"},
		{"Linux x86_64", "Linux x86_64"},
	}
	for _, tt := range tests {
		preset := FingerprintPreset{
			Navigator: PresetNavigator{Platform: tt.platform},
		}
		config := FromPreset(preset, nil)
		if config["navigator.oscpu"] != tt.wantOscpu {
			t.Errorf("platform=%q: oscpu=%q, want %q", tt.platform, config["navigator.oscpu"], tt.wantOscpu)
		}
	}
}

func TestBuildInitScript(t *testing.T) {
	script, err := BuildInitScript(initScriptValues{
		FontSpacingSeed:      42,
		AudioFingerprintSeed: 99,
		CanvasSeed:           7,
		NavigatorPlatform:    "Win32",
		NavigatorUserAgent:   "TestAgent",
		WebGLVendor:          "TestVendor",
		WebGLRenderer:        "TestRenderer",
		ScreenWidth:          1920,
		ScreenHeight:         1080,
		ScreenColorDepth:     24,
		Timezone:             "UTC",
		FontList:             []string{"Arial", "Verdana"},
		SpeechVoices:         []string{"Alex"},
		WebRTCIP:             "1.2.3.4",
	})
	if err != nil {
		t.Fatalf("BuildInitScript error: %v", err)
	}
	if script == "" {
		t.Fatal("BuildInitScript returned empty")
	}
	// Should contain key function calls
	for _, fn := range []string{
		"setFontSpacingSeed",
		"setAudioFingerprintSeed",
		"setCanvasSeed",
		"setNavigatorPlatform",
		"setNavigatorUserAgent",
		"setWebGLVendor",
		"setWebGLRenderer",
		"setScreenDimensions",
		"setScreenColorDepth",
		"setTimezone",
		"setWebRTCIPv4",
		"setFontList",
		"setSpeechVoices",
	} {
		if !containsSubstring(script, fn) {
			t.Errorf("init script missing function call: %s", fn)
		}
	}
}

func TestBuildInitScriptEmpty(t *testing.T) {
	// Minimal — should still produce valid JS with timezone fallback
	script, err := BuildInitScript(initScriptValues{})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !containsSubstring(script, "(function(v)") {
		t.Fatal("should start with IIFE")
	}
	if !containsSubstring(script, "setTimezone") {
		t.Fatal("should have timezone fallback")
	}
	if !containsSubstring(script, "setWebRTCIPv4") {
		t.Fatal("should have WebRTC default")
	}
}
