package camoufox

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// GeoIP directory and config paths.
func geoIPDir() string {
	dir, _ := os.UserCacheDir()
	return filepath.Join(dir, "camoufox", "geoip")
}

func mmdbDir() string {
	return filepath.Join(geoIPDir(), "mmdb")
}

func geoIPConfigPath() string {
	return filepath.Join(geoIPDir(), "config.yml")
}

// loadGeoIPRepos loads GeoIP repos and default name from repos.yml.
func loadGeoIPRepos() ([]map[string]interface{}, string, error) {
	var raw map[string]interface{}
	if err := LoadYAML("repos.yml", &raw); err != nil {
		return nil, "", err
	}
	repos, _ := raw["geoip"].([]interface{})
	var result []map[string]interface{}
	for _, r := range repos {
		if m, ok := r.(map[string]interface{}); ok {
			result = append(result, m)
		}
	}
	defaults, _ := raw["default"].(map[string]interface{})
	defaultName := "GeoLite2"
	if n, ok := defaults["geoip"].(string); ok {
		defaultName = n
	}
	return result, defaultName, nil
}

// getGeoIPConfigByName finds a GeoIP config by name from repos.yml.
func getGeoIPConfigByName(name string) (map[string]interface{}, error) {
	repos, defaultName, err := loadGeoIPRepos()
	if err != nil {
		return nil, err
	}
	targetName := name
	if targetName == "" {
		targetName = defaultName
	}
	for _, repo := range repos {
		rn, _ := repo["name"].(string)
		if strings.EqualFold(rn, targetName) {
			return repo, nil
		}
	}
	if name != "" {
		return nil, fmt.Errorf("GeoIP database '%s' not found", name)
	}
	if len(repos) > 0 {
		return repos[0], nil
	}
	return nil, fmt.Errorf("no GeoIP repos configured")
}

// LoadGeoIPConfig loads active GeoIP config from disk.
func LoadGeoIPConfig() (map[string]interface{}, error) {
	cfgPath := geoIPConfigPath()
	if _, err := os.Stat(cfgPath); err == nil {
		data, err := os.ReadFile(cfgPath)
		if err == nil {
			var saved map[string]interface{}
			if yaml.Unmarshal(data, &saved) == nil {
				if n, ok := saved["name"].(string); ok {
					cfg, err := getGeoIPConfigByName(n)
					if err == nil {
						return cfg, nil
					}
				}
				return saved, nil
			}
		}
	}
	return getGeoIPConfigByName("")
}

// SaveGeoIPConfig saves the active GeoIP source name.
func SaveGeoIPConfig(config map[string]interface{}) error {
	os.MkdirAll(geoIPDir(), 0o755)
	name, _ := config["name"].(string)
	data, err := yaml.Marshal(map[string]string{"name": name})
	if err != nil {
		return err
	}
	return os.WriteFile(geoIPConfigPath(), data, 0o644)
}

// GetMMDBPath returns the path to the mmdb file for the specified IP version.
func GetMMDBPath(ipVersion string, config map[string]interface{}) string {
	if config == nil {
		config, _ = LoadGeoIPConfig()
	}
	name := strings.ToLower(stringFromMap(config, "name"))
	if name == "" {
		name = "geolite2"
	}
	urls, _ := config["urls"].(map[string]interface{})
	if _, ok := urls["combined"]; ok {
		return filepath.Join(mmdbDir(), name+"-combined.mmdb")
	}
	return filepath.Join(mmdbDir(), name+"-"+ipVersion+".mmdb")
}

// NeedsUpdate checks if the GeoIP database is older than 30 days.
func NeedsUpdate(config map[string]interface{}) bool {
	if config == nil {
		config, _ = LoadGeoIPConfig()
	}
	ipv4Path := GetMMDBPath("ipv4", config)
	info, err := os.Stat(ipv4Path)
	if err != nil {
		return true
	}
	return time.Since(info.ModTime()) > 30*24*time.Hour
}

// DownloadMMDB downloads the GeoIP database(s).
func DownloadMMDB(source string) error {
	var config map[string]interface{}
	var err error
	if source != "" {
		config, err = getGeoIPConfigByName(source)
	} else {
		config, err = LoadGeoIPConfig()
	}
	if err != nil {
		return err
	}
	urls, _ := config["urls"].(map[string]interface{})
	if urls == nil {
		return fmt.Errorf("no URLs in GeoIP config")
	}
	os.MkdirAll(mmdbDir(), 0o755)
	name := strings.ToLower(stringFromMap(config, "name"))

	for ipVer, urlList := range urls {
		mmdbPath := filepath.Join(mmdbDir(), name+"-"+ipVer+".mmdb")
		var urlSlice []string
		switch v := urlList.(type) {
		case string:
			urlSlice = []string{v}
		case []interface{}:
			for _, u := range v {
				if s, ok := u.(string); ok {
					urlSlice = append(urlSlice, s)
				}
			}
		}
		var lastErr error
		for _, u := range urlSlice {
			data, err := WebDL(u)
			if err != nil {
				lastErr = err
				continue
			}
			if err := os.WriteFile(mmdbPath, data, 0o644); err != nil {
				lastErr = err
				continue
			}
			lastErr = nil
			break
		}
		if lastErr != nil {
			return fmt.Errorf("failed to download %s: %w", ipVer, lastErr)
		}
	}
	return SaveGeoIPConfig(config)
}

// RemoveMMDB removes the GeoIP database and config.
func RemoveMMDB() {
	dir := geoIPDir()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		fmt.Println("GeoIP database not found.")
		return
	}
	os.RemoveAll(dir)
	fmt.Println("GeoIP database removed.")
}

// GetGeolocation looks up geo data for an IP address.
// This is a simplified version that queries ip-api.com since maxminddb
// requires an optional dependency. For full maxminddb support, use the
// Python library or add a maxminddb Go dependency.
func GetGeolocation(ip string, geoipDB string) (Geolocation, error) {
	if err := ValidateIP(ip); err != nil {
		return Geolocation{}, err
	}

	// Use ip-api.com as fallback (the Python version uses maxminddb)
	apiURL := fmt.Sprintf("http://ip-api.com/json/%s?fields=query,lat,lon,timezone,countryCode", url.QueryEscape(ip))
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return Geolocation{}, fmt.Errorf("%w: %v", ErrUnknownIPLocation, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Geolocation{}, err
	}
	var data struct {
		Query       string  `json:"query"`
		Lat         float64 `json:"lat"`
		Lon         float64 `json:"lon"`
		Timezone    string  `json:"timezone"`
		CountryCode string  `json:"countryCode"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return Geolocation{}, err
	}
	if data.CountryCode == "" {
		return Geolocation{}, fmt.Errorf("%w: IP not found: %s", ErrUnknownIPLocation, ip)
	}

	locale, err := DefaultSelector().FromRegion(data.CountryCode)
	if err != nil {
		locale = Locale{Language: "en", Region: data.CountryCode}
	}

	return Geolocation{
		Loc:       locale,
		Longitude: data.Lon,
		Latitude:  data.Lat,
		Timezone:  data.Timezone,
	}, nil
}
