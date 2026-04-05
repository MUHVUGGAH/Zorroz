package camoufox

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// OS/arch mapping tables mirroring pkgman.py.
var archMap = map[string]string{
	"amd64":   "x86_64",
	"x86_64":  "x86_64",
	"x86":     "x86_64",
	"i686":    "i686",
	"i386":    "i686",
	"arm64":   "arm64",
	"aarch64": "arm64",
	"armv5l":  "arm64",
	"armv6l":  "arm64",
	"armv7l":  "arm64",
}

var osMap = map[string]string{
	"darwin":  "mac",
	"linux":   "lin",
	"windows": "win",
}

var osArchMatrix = map[string][]string{
	"win": {"x86_64", "i686"},
	"mac": {"x86_64", "arm64"},
	"lin": {"x86_64", "arm64", "i686"},
}

var launchFiles = map[string]string{
	"win": "camoufox.exe",
	"mac": "../MacOS/camoufox",
	"lin": "camoufox-bin",
}

// OSName returns the short Camoufox OS identifier for the current platform.
func OSName() string {
	return osMap[runtime.GOOS]
}

// InstallDir returns the user-cache Camoufox directory.
// Matches Python's platformdirs.user_cache_dir("camoufox") behavior:
//   - Windows: %LOCALAPPDATA%\camoufox\camoufox\Cache
//   - macOS:   ~/Library/Caches/camoufox
//   - Linux:   ~/.cache/camoufox
func InstallDir() string {
	if runtime.GOOS == "windows" {
		local := os.Getenv("LOCALAPPDATA")
		if local == "" {
			local, _ = os.UserCacheDir()
		}
		return filepath.Join(local, "camoufox", "camoufox", "Cache")
	}
	dir, _ := os.UserCacheDir()
	return filepath.Join(dir, "camoufox")
}

// LocalDataDir returns the path to data files co-located with this Go source.
func LocalDataDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Dir(file)
}

// LoadYAML loads a YAML file from the local data directory.
func LoadYAML(name string, out interface{}) error {
	data, err := os.ReadFile(filepath.Join(LocalDataDir(), name))
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, out)
}

// loadYAMLMap loads a YAML file as a generic map.
func loadYAMLMap(name string) (map[string]interface{}, error) {
	var m map[string]interface{}
	if err := LoadYAML(name, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// Version is a comparable version string (up to 5 parts).
type Version struct {
	Build      string
	VersionStr string // "version" field; named VersionStr to avoid colliding with the type
	sortedRel  [5]int
}

// NewVersion constructs a Version.
func NewVersion(build string, version string) Version {
	v := Version{Build: build, VersionStr: version}
	parts := strings.Split(build, ".")
	for i := 0; i < 5; i++ {
		if i < len(parts) {
			p := parts[i]
			var n int
			isNum := true
			for _, c := range p {
				if c >= '0' && c <= '9' {
					n = n*10 + int(c-'0')
				} else {
					isNum = false
					break
				}
			}
			if isNum {
				v.sortedRel[i] = n
			} else if len(p) > 0 {
				v.sortedRel[i] = int(p[0]) - 1024
			}
		}
	}
	return v
}

// FullString returns "version-build".
func (v Version) FullString() string {
	return v.VersionStr + "-" + v.Build
}

// Less returns true if v < other.
func (v Version) Less(other Version) bool {
	return v.sortedRel[0] < other.sortedRel[0] ||
		(v.sortedRel[0] == other.sortedRel[0] && v.sortedRel[1] < other.sortedRel[1]) ||
		(v.sortedRel[0] == other.sortedRel[0] && v.sortedRel[1] == other.sortedRel[1] && v.sortedRel[2] < other.sortedRel[2]) ||
		(v.sortedRel[0] == other.sortedRel[0] && v.sortedRel[1] == other.sortedRel[1] && v.sortedRel[2] == other.sortedRel[2] && v.sortedRel[3] < other.sortedRel[3]) ||
		(v.sortedRel[0] == other.sortedRel[0] && v.sortedRel[1] == other.sortedRel[1] && v.sortedRel[2] == other.sortedRel[2] && v.sortedRel[3] == other.sortedRel[3] && v.sortedRel[4] < other.sortedRel[4])
}

// Equal returns true if v == other.
func (v Version) Equal(other Version) bool {
	return v.sortedRel == other.sortedRel
}

// LessOrEqual returns true if v <= other.
func (v Version) LessOrEqual(other Version) bool {
	return v.Less(other) || v.Equal(other)
}

// IsSupported returns true if the version is within[VERSION_MIN, VERSION_MAX).
func (v Version) IsSupported() bool {
	return VersionMin().LessOrEqual(v) && v.Less(VersionMax())
}

// VersionMin returns the minimum supported version.
func VersionMin() Version { return NewVersion(CONSTRAINTS.MinVersion, "") }

// VersionMax returns the maximum supported version.
func VersionMax() Version { return NewVersion(CONSTRAINTS.MaxVersion, "") }

// VersionFromPath reads version.json from the given directory.
func VersionFromPath(path string) (Version, error) {
	vpath := filepath.Join(path, "version.json")
	data, err := os.ReadFile(vpath)
	if err != nil {
		return Version{}, fmt.Errorf("version information not found at %s: %w", vpath, err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return Version{}, err
	}
	build := ""
	if b, ok := raw["build"].(string); ok {
		build = b
	} else if b, ok := raw["release"].(string); ok {
		build = b
	} else if b, ok := raw["tag"].(string); ok {
		build = b
	}
	ver := ""
	if v, ok := raw["version"].(string); ok {
		ver = v
	}
	return NewVersion(build, ver), nil
}

// RepoConfig holds configuration for a Camoufox repository.
type RepoConfig struct {
	Repos    []string
	Name     string
	Pattern  string
	BuildMin string
	BuildMax string
}

// loadReposYAML loads repos.yml into a raw structure.
func loadReposYAML() (map[string]interface{}, error) {
	return loadYAMLMap("repos.yml")
}

// LoadRepoConfigs loads all repository configurations from repos.yml.
func LoadRepoConfigs() ([]RepoConfig, error) {
	raw, err := loadReposYAML()
	if err != nil {
		return nil, err
	}
	browsers, _ := raw["browsers"].([]interface{})
	var configs []RepoConfig
	for _, b := range browsers {
		bm, _ := b.(map[string]interface{})
		if bm == nil {
			continue
		}
		rc := RepoConfig{
			Name:    stringFromMap(bm, "name"),
			Pattern: stringFromMap(bm, "pattern"),
		}
		repoStr := stringFromMap(bm, "repo")
		for _, r := range strings.Split(repoStr, ",") {
			rc.Repos = append(rc.Repos, strings.TrimSpace(r))
		}
		configs = append(configs, rc)
	}
	return configs, nil
}

// GetDefaultRepoName returns the default repo name from repos.yml.
func GetDefaultRepoName() string {
	raw, err := loadReposYAML()
	if err != nil {
		return "Official"
	}
	defaults, _ := raw["default"].(map[string]interface{})
	if n, ok := defaults["browser"].(string); ok {
		return n
	}
	return "Official"
}

// GetDefaultRepoConfig returns the default repo config.
func GetDefaultRepoConfig() (RepoConfig, error) {
	configs, err := LoadRepoConfigs()
	if err != nil {
		return RepoConfig{}, err
	}
	name := strings.ToLower(GetDefaultRepoName())
	for _, c := range configs {
		if strings.ToLower(c.Name) == name {
			return c, nil
		}
	}
	if len(configs) > 0 {
		return configs[0], nil
	}
	return RepoConfig{}, fmt.Errorf("no repos configured")
}

// IsVersionSupported checks if a version is within the repo's build range.
func (rc RepoConfig) IsVersionSupported(v Version) bool {
	if rc.BuildMin == "" || rc.BuildMax == "" {
		return true
	}
	bMin := NewVersion(rc.BuildMin, "")
	bMax := NewVersion(rc.BuildMax, "")
	return bMin.LessOrEqual(v) && v.LessOrEqual(bMax)
}

// BuildPattern returns a compiled regex for matching release asset names.
func (rc RepoConfig) BuildPattern(spoofOS, spoofArch string) (*regexp.Regexp, error) {
	osName := spoofOS
	if osName == "" {
		osName = OSName()
	}
	arch := spoofArch
	if arch == "" {
		plat := strings.ToLower(runtime.GOARCH)
		a, ok := archMap[plat]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrUnsupportedArchitecture, plat)
		}
		arch = a
	}
	replacements := map[string]string{
		"name":    `(?P<name>\w+)`,
		"version": `(?P<version>[^-]+)`,
		"build":   `(?P<build>[^-]+)`,
		"os":      regexp.QuoteMeta(osName),
		"arch":    regexp.QuoteMeta(arch),
	}
	pattern := strings.ReplaceAll(rc.Pattern, ".", `\.`)
	for k, v := range replacements {
		pattern = strings.ReplaceAll(pattern, "{"+k+"}", v)
	}
	return regexp.Compile(pattern)
}

// AvailableVersion holds info about a release version from GitHub.
type AvailableVersion struct {
	Version        Version
	URL            string
	IsPrerelease   bool
	AssetID        *int
	AssetSize      *int
	AssetUpdatedAt string
}

// Display returns a display-friendly version string.
func (av AvailableVersion) Display() string {
	pre := ""
	if av.IsPrerelease {
		pre = " (prerelease)"
	}
	return fmt.Sprintf("v%s%s", av.Version.FullString(), pre)
}

// ToMetadata returns a metadata dict suitable for version.json.
func (av AvailableVersion) ToMetadata() map[string]interface{} {
	m := map[string]interface{}{
		"version":    av.Version.VersionStr,
		"build":      av.Version.Build,
		"prerelease": av.IsPrerelease,
	}
	if av.AssetID != nil {
		m["asset_id"] = *av.AssetID
	}
	if av.AssetSize != nil {
		m["asset_size"] = *av.AssetSize
	}
	if av.AssetUpdatedAt != "" {
		m["asset_updated_at"] = av.AssetUpdatedAt
	}
	return m
}

// GitHubDownloader fetches release assets with fallback repos.
type GitHubDownloader struct {
	GithubRepos  []string
	GithubRepo   string
	IsPrerelease bool
}

// NewGitHubDownloader constructs a downloader.
func NewGitHubDownloader(repos []string) *GitHubDownloader {
	return &GitHubDownloader{
		GithubRepos: repos,
		GithubRepo:  repos[0],
	}
}

// getReleases fetches release data from a single GitHub repo.
func (gd *GitHubDownloader) getReleases(repo string) ([]map[string]interface{}, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases", repo)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d for %s", resp.StatusCode, repo)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var releases []map[string]interface{}
	return releases, json.Unmarshal(body, &releases)
}

// CamoufoxFetcher handles fetching and installing Camoufox.
type CamoufoxFetcher struct {
	*GitHubDownloader
	RepoConfig      RepoConfig
	Arch            string
	Pattern         *regexp.Regexp
	VersionObj      *Version
	URL             string
	SelectedVersion *AvailableVersion
}

// NewCamoufoxFetcher constructs a fetcher and finds the latest release.
func NewCamoufoxFetcher(rc *RepoConfig, selected *AvailableVersion) (*CamoufoxFetcher, error) {
	if rc == nil {
		def, err := GetDefaultRepoConfig()
		if err != nil {
			return nil, err
		}
		rc = &def
	}
	pat, err := rc.BuildPattern("", "")
	if err != nil {
		return nil, err
	}
	arch := strings.ToLower(runtime.GOARCH)
	mappedArch, ok := archMap[arch]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedArchitecture, arch)
	}

	f := &CamoufoxFetcher{
		GitHubDownloader: NewGitHubDownloader(rc.Repos),
		RepoConfig:       *rc,
		Arch:             mappedArch,
		Pattern:          pat,
	}

	if selected != nil {
		f.SelectedVersion = selected
		v := selected.Version
		f.VersionObj = &v
		f.URL = selected.URL
		f.IsPrerelease = selected.IsPrerelease
	} else {
		if err := f.FetchLatest(); err != nil {
			return nil, err
		}
	}
	return f, nil
}

// FetchLatest finds the newest compatible release.
func (f *CamoufoxFetcher) FetchLatest() error {
	var lastErr error
	for _, repo := range f.GithubRepos {
		releases, err := f.getReleases(repo)
		if err != nil {
			lastErr = err
			continue
		}
		for _, rel := range releases {
			assets, _ := rel["assets"].([]interface{})
			for _, a := range assets {
				asset, _ := a.(map[string]interface{})
				if asset == nil {
					continue
				}
				name, _ := asset["name"].(string)
				match := f.Pattern.FindStringSubmatch(name)
				if match == nil {
					continue
				}
				names := f.Pattern.SubexpNames()
				parts := map[string]string{}
				for i, n := range names {
					if n != "" {
						parts[n] = match[i]
					}
				}
				v := NewVersion(parts["build"], parts["version"])
				if !f.RepoConfig.IsVersionSupported(v) {
					continue
				}
				url, _ := asset["browser_download_url"].(string)
				f.GithubRepo = repo
				f.VersionObj = &v
				f.URL = url
				f.IsPrerelease, _ = rel["prerelease"].(bool)
				return nil
			}
		}
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("%w: no matching release for %s %s", ErrMissingRelease, OSName(), f.Arch)
}

// ListAvailableVersions fetches all supported versions from GitHub.
func ListAvailableVersions(rc *RepoConfig, includePrerelease bool, spoofOS, spoofArch string) ([]AvailableVersion, error) {
	if rc == nil {
		def, err := GetDefaultRepoConfig()
		if err != nil {
			return nil, err
		}
		rc = &def
	}
	pat, err := rc.BuildPattern(spoofOS, spoofArch)
	if err != nil {
		return nil, err
	}
	gd := NewGitHubDownloader(rc.Repos)
	var releases []map[string]interface{}
	for _, repo := range gd.GithubRepos {
		rels, err := gd.getReleases(repo)
		if err != nil {
			continue
		}
		releases = rels
		break
	}

	var versions []AvailableVersion
	seenBuilds := map[string]bool{}
	for _, rel := range releases {
		isPre, _ := rel["prerelease"].(bool)
		if isPre && !includePrerelease {
			continue
		}
		assets, _ := rel["assets"].([]interface{})
		for _, a := range assets {
			asset, _ := a.(map[string]interface{})
			if asset == nil {
				continue
			}
			name, _ := asset["name"].(string)
			match := pat.FindStringSubmatch(name)
			if match == nil {
				continue
			}
			names := pat.SubexpNames()
			parts := map[string]string{}
			for i, n := range names {
				if n != "" {
					parts[n] = match[i]
				}
			}
			v := NewVersion(parts["build"], parts["version"])
			if !rc.IsVersionSupported(v) {
				continue
			}
			if seenBuilds[v.Build] {
				continue
			}
			seenBuilds[v.Build] = true
			url, _ := asset["browser_download_url"].(string)
			av := AvailableVersion{
				Version:        v,
				URL:            url,
				IsPrerelease:   isPre,
				AssetUpdatedAt: stringFromMapIface(asset, "updated_at"),
			}
			if id, ok := asset["id"].(float64); ok {
				ival := int(id)
				av.AssetID = &ival
			}
			if sz, ok := asset["size"].(float64); ok {
				ival := int(sz)
				av.AssetSize = &ival
			}
			versions = append(versions, av)
		}
	}
	sort.Slice(versions, func(i, j int) bool {
		return versions[j].Version.Less(versions[i].Version)
	})
	return versions, nil
}

// InstalledVerStr returns the full version string of the active install.
func InstalledVerStr() (string, error) {
	active := GetActivePath()
	if active == "" {
		return "", fmt.Errorf("%w: please run 'camoufox fetch' to install", ErrCamoufoxNotInstalled)
	}
	v, err := VersionFromPath(active)
	if err != nil {
		return "", err
	}
	return v.FullString(), nil
}

// GetPath returns the path to a file inside the active camoufox directory.
func GetPath(file string) (string, error) {
	cp, err := CamoufoxPath(false)
	if err != nil {
		return "", err
	}
	if OSName() == "mac" {
		return filepath.Join(cp, "Camoufox.app", "Contents", "Resources", file), nil
	}
	return filepath.Join(cp, file), nil
}

// LaunchPath returns the path to the camoufox executable.
func LaunchPath(browserPath string) (string, error) {
	var execPath string
	if browserPath != "" {
		if OSName() == "mac" {
			execPath = filepath.Join(browserPath, "Camoufox.app", "Contents", "Resources", launchFiles[OSName()])
		} else {
			execPath = filepath.Join(browserPath, launchFiles[OSName()])
		}
	} else {
		p, err := GetPath(launchFiles[OSName()])
		if err != nil {
			return "", err
		}
		execPath = p
	}
	if _, err := os.Stat(execPath); os.IsNotExist(err) {
		return "", fmt.Errorf("%w: at %s", ErrCamoufoxNotInstalled, execPath)
	}
	return execPath, nil
}

// CamoufoxPath returns the full path to the active camoufox folder.
func CamoufoxPath(downloadIfMissing bool) (string, error) {
	active := GetActivePath()
	if active != "" {
		v, err := VersionFromPath(active)
		if err == nil && v.IsSupported() {
			return active, nil
		}
	}
	if !downloadIfMissing {
		return "", fmt.Errorf("%w: please run 'camoufox fetch' to install", ErrCamoufoxNotInstalled)
	}
	// Attempt fetch
	f, err := NewCamoufoxFetcher(nil, nil)
	if err != nil {
		return "", err
	}
	_ = f // Install logic delegated to multiversion
	return CamoufoxPath(false)
}

// WebDL downloads a file from url and returns the body bytes.
func WebDL(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if token := os.Getenv("GITHUB_TOKEN"); token != "" && strings.Contains(url, "api.github") {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// helper to extract string keys from map[string]interface{}.
func stringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func stringFromMapIface(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
