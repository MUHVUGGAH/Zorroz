package camoufox

import (
	"os"
	"path/filepath"
	"testing"
)

// ==========================================================================
// These tests require the data files co-located with the Go source.
// They exercise real file I/O but do NOT need a browser or network.
// ==========================================================================

// ---------- warnings.go ----------

func TestLoadWarnings(t *testing.T) {
	data := loadWarnings()
	if data == nil {
		t.Fatal("loadWarnings returned nil")
	}
	// warnings.yml should have at least one key
	if len(data) == 0 {
		t.Fatal("warnings.yml loaded but is empty")
	}
}

func TestLeakWarningKnown(t *testing.T) {
	data := loadWarnings()
	if len(data) == 0 {
		t.Skip("no warnings loaded")
	}
	// Pick the first key and make sure LeakWarning doesn't panic
	for key := range data {
		LeakWarning(key, nil) // should log but not panic
		break
	}
}

func TestLeakWarningSuppressed(t *testing.T) {
	bTrue := true
	// With IKnowWhatImDoing=true, should not print anything
	LeakWarning("locale", &bTrue)
}

// ---------- locales.go — file-dependent ----------

func TestDefaultSelector(t *testing.T) {
	sel := DefaultSelector()
	if sel == nil {
		t.Fatal("DefaultSelector returned nil")
	}
	if len(sel.territories) == 0 {
		t.Fatal("DefaultSelector has no territories loaded from territoryInfo.xml")
	}
}

func TestFromRegion(t *testing.T) {
	sel := DefaultSelector()
	l, err := sel.FromRegion("US")
	if err != nil {
		t.Fatalf("FromRegion(US) error: %v", err)
	}
	if l.Region != "US" {
		t.Fatalf("expected region US, got %s", l.Region)
	}
	if l.Language == "" {
		t.Fatal("expected a language for US region")
	}
}

func TestFromRegionUnknown(t *testing.T) {
	sel := DefaultSelector()
	_, err := sel.FromRegion("ZZ")
	if err == nil {
		t.Fatal("expected error for unknown region ZZ")
	}
}

func TestFromLanguage(t *testing.T) {
	sel := DefaultSelector()
	l, err := sel.FromLanguage("en")
	if err != nil {
		t.Fatalf("FromLanguage(en) error: %v", err)
	}
	if l.Language != "en" {
		t.Fatalf("expected language 'en', got %q", l.Language)
	}
	if l.Region == "" {
		t.Fatal("expected a region from FromLanguage(en)")
	}
}

func TestHandleLocaleWithSelector(t *testing.T) {
	// Short code should use the selector
	l, err := HandleLocale("US", false)
	if err != nil {
		t.Fatalf("HandleLocale(US) error: %v", err)
	}
	if l.Region != "US" {
		t.Fatalf("expected region US, got %s", l.Region)
	}
}

// ---------- pkgman.go — file-dependent ----------

func TestLoadYAML(t *testing.T) {
	var data map[string]interface{}
	err := LoadYAML("warnings.yml", &data)
	if err != nil {
		t.Fatalf("LoadYAML(warnings.yml) error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("warnings.yml loaded as empty map")
	}
}

func TestLoadRepoConfigs(t *testing.T) {
	configs, err := LoadRepoConfigs()
	if err != nil {
		t.Fatalf("LoadRepoConfigs error: %v", err)
	}
	if len(configs) == 0 {
		t.Fatal("no repo configs loaded from repos.yml")
	}
	// Each config should have a name and repos
	for _, c := range configs {
		if c.Name == "" {
			t.Fatal("repo config with empty name")
		}
		if len(c.Repos) == 0 {
			t.Fatalf("repo config %q has no repos", c.Name)
		}
	}
}

func TestGetDefaultRepoName(t *testing.T) {
	name := GetDefaultRepoName()
	if name == "" {
		t.Fatal("GetDefaultRepoName returned empty")
	}
}

func TestGetDefaultRepoConfig(t *testing.T) {
	config, err := GetDefaultRepoConfig()
	if err != nil {
		t.Fatalf("GetDefaultRepoConfig error: %v", err)
	}
	if config.Name == "" {
		t.Fatal("default repo config has empty name")
	}
}

func TestRepoConfigBuildPattern(t *testing.T) {
	config, err := GetDefaultRepoConfig()
	if err != nil {
		t.Skipf("no default config: %v", err)
	}
	if config.Pattern == "" {
		t.Skip("no pattern in default config")
	}
	re, err := config.BuildPattern("", "")
	if err != nil {
		t.Fatalf("BuildPattern error: %v", err)
	}
	if re == nil {
		t.Fatal("BuildPattern returned nil regex")
	}
}

// ---------- multiversion.go — file-dependent ----------

func TestLoadUserConfig(t *testing.T) {
	config := LoadUserConfig()
	if config == nil {
		t.Fatal("LoadUserConfig returned nil")
	}
	// On a system with camoufox installed, active_version should be set
	if av, ok := config["active_version"].(string); ok {
		if av == "" {
			t.Log("active_version is empty string")
		}
	}
}

func TestListInstalled(t *testing.T) {
	installed := ListInstalled()
	// Should not panic; may be empty if nothing installed
	if installed == nil {
		t.Fatal("ListInstalled returned nil, expected empty slice")
	}
}

func TestGetActivePath(t *testing.T) {
	path := GetActivePath()
	// On a system with camoufox installed, path should be non-empty
	if path == "" {
		t.Skip("no active camoufox version installed")
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("active path does not exist: %s", path)
	}
}

// ---------- fingerprints.go — file-dependent ----------

func TestLoadPresets(t *testing.T) {
	presets, err := LoadPresets()
	if err != nil {
		t.Fatalf("LoadPresets error: %v", err)
	}
	if presets == nil {
		t.Fatal("LoadPresets returned nil")
	}
	if len(presets.Presets) == 0 {
		t.Fatal("no presets loaded")
	}
	// Should have at least one OS key
	for _, key := range []string{"windows", "macos", "linux"} {
		if _, ok := presets.Presets[key]; !ok {
			t.Errorf("missing preset OS key: %s", key)
		}
	}
}

func TestGetRandomPreset(t *testing.T) {
	preset, err := GetRandomPreset()
	if err != nil {
		t.Fatalf("GetRandomPreset error: %v", err)
	}
	if preset == nil {
		t.Fatal("GetRandomPreset returned nil")
	}
	if preset.Navigator.UserAgent == "" {
		t.Fatal("preset has empty UserAgent")
	}
}

func TestGetRandomPresetByOS(t *testing.T) {
	for _, os := range []string{"windows", "macos", "linux"} {
		preset, err := GetRandomPreset(os)
		if err != nil {
			t.Fatalf("GetRandomPreset(%s) error: %v", os, err)
		}
		if preset == nil {
			t.Fatalf("GetRandomPreset(%s) returned nil", os)
		}
	}
}

func TestGenerateRandomFontSubset(t *testing.T) {
	for _, os := range []string{"windows", "macos", "linux"} {
		fonts, err := GenerateRandomFontSubset(os)
		if err != nil {
			t.Fatalf("GenerateRandomFontSubset(%s) error: %v", os, err)
		}
		if len(fonts) == 0 {
			t.Fatalf("GenerateRandomFontSubset(%s) returned empty", os)
		}
	}
}

func TestGenerateRandomVoiceSubset(t *testing.T) {
	// Windows should return full list; macOS a subset; Linux empty
	voices, err := GenerateRandomVoiceSubset("windows")
	if err != nil {
		t.Fatalf("GenerateRandomVoiceSubset(windows) error: %v", err)
	}
	if len(voices) == 0 {
		t.Log("warning: windows voice list is empty")
	}

	voices, err = GenerateRandomVoiceSubset("macos")
	if err != nil {
		t.Fatalf("GenerateRandomVoiceSubset(macos) error: %v", err)
	}
	// macos should have some voices
	if len(voices) == 0 {
		t.Log("warning: macos voice subset is empty")
	}

	voices, err = GenerateRandomVoiceSubset("linux")
	if err != nil {
		t.Fatalf("GenerateRandomVoiceSubset(linux) error: %v", err)
	}
	// linux returns empty
	if len(voices) != 0 {
		t.Fatalf("expected empty voice list for linux, got %d", len(voices))
	}
}

func TestLoadBFYAML(t *testing.T) {
	data, err := loadBFYAML()
	if err != nil {
		t.Fatalf("loadBFYAML error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("browserforge.yml loaded as empty map")
	}
}

// ---------- utils.go — file-dependent ----------

func TestUpdateFonts(t *testing.T) {
	config := map[string]interface{}{}
	err := UpdateFonts(config, "win")
	if err != nil {
		t.Fatalf("UpdateFonts error: %v", err)
	}
	fonts, ok := config["fonts"]
	if !ok {
		t.Fatal("UpdateFonts did not set fonts key")
	}
	fontsList, ok := fonts.([]string)
	if !ok {
		t.Fatal("fonts should be []string")
	}
	if len(fontsList) == 0 {
		t.Fatal("fonts list is empty")
	}
}

func TestUpdateFontsMerge(t *testing.T) {
	config := map[string]interface{}{
		"fonts": []string{"CustomFont"},
	}
	err := UpdateFonts(config, "win")
	if err != nil {
		t.Fatalf("UpdateFonts error: %v", err)
	}
	fontsList := config["fonts"].([]string)
	found := false
	for _, f := range fontsList {
		if f == "CustomFont" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("existing custom font should be preserved after merge")
	}
}

// ---------- webgl — file-dependent (SQLite) ----------

func TestSampleWebGLFromDB(t *testing.T) {
	dbPath := filepath.Join(LocalDataDir(), "webgl", "webgl_data.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Skipf("webgl_data.db not found at %s", dbPath)
	}

	for _, os := range []string{"win", "mac", "lin"} {
		result, err := SampleWebGLFromDB(os, nil, nil)
		if err != nil {
			t.Fatalf("SampleWebGLFromDB(%s) error: %v", os, err)
		}
		if result == nil {
			t.Fatalf("SampleWebGLFromDB(%s) returned nil", os)
		}
		if _, ok := result["webGl:vendor"]; !ok {
			t.Fatalf("SampleWebGLFromDB(%s) missing webGl:vendor", os)
		}
		if _, ok := result["webGl:renderer"]; !ok {
			t.Fatalf("SampleWebGLFromDB(%s) missing webGl:renderer", os)
		}
	}
}

// ---------- End-to-end preset pipeline ----------

func TestPresetToInitScriptPipeline(t *testing.T) {
	// Full pipeline: load preset -> FromPreset -> BuildInitScript
	preset, err := GetRandomPreset("windows")
	if err != nil {
		t.Fatalf("GetRandomPreset error: %v", err)
	}

	config := FromPreset(*preset, nil)
	if config["navigator.userAgent"] == nil {
		t.Fatal("config missing navigator.userAgent")
	}

	fonts := toStringSlice(config["fonts"])
	voices := toStringSlice(config["voices"])

	script, err := BuildInitScript(initScriptValues{
		FontSpacingSeed:      config["fonts:spacing_seed"],
		AudioFingerprintSeed: config["audio:seed"],
		CanvasSeed:           config["canvas:seed"],
		NavigatorPlatform:    stringValue(config["navigator.platform"]),
		NavigatorOscpu:       stringValue(config["navigator.oscpu"]),
		NavigatorUserAgent:   stringValue(config["navigator.userAgent"]),
		HardwareConcurrency:  config["navigator.hardwareConcurrency"],
		WebGLVendor:          stringValue(config["webGl:vendor"]),
		WebGLRenderer:        stringValue(config["webGl:renderer"]),
		ScreenWidth:          intValue(config["screen.width"]),
		ScreenHeight:         intValue(config["screen.height"]),
		ScreenColorDepth:     intValue(config["screen.colorDepth"]),
		Timezone:             stringValue(config["timezone"]),
		FontList:             fonts,
		SpeechVoices:         voices,
	})
	if err != nil {
		t.Fatalf("BuildInitScript error: %v", err)
	}
	if script == "" {
		t.Fatal("pipeline produced empty init script")
	}
	if !containsSubstring(script, "setNavigatorUserAgent") {
		t.Fatal("init script should set the user agent")
	}
}
