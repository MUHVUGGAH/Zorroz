package camoufox

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// browsersDir returns the path to the browsers subdirectory.
func browsersDir() string {
	return filepath.Join(InstallDir(), "browsers")
}

// configFile returns the path to the user config file.
func configFile() string {
	return filepath.Join(InstallDir(), "config.json")
}

// repoCacheFile returns the path to the repo cache file.
func repoCacheFile() string {
	return filepath.Join(InstallDir(), "repo_cache.json")
}

// compatFlag returns the path to the compatibility flag file.
func compatFlag() string {
	return filepath.Join(InstallDir(), ".0.5_FLAG")
}

// LoadUserConfig loads user config from disk.
func LoadUserConfig() map[string]interface{} {
	data, err := os.ReadFile(configFile())
	if err != nil {
		return map[string]interface{}{}
	}
	var config map[string]interface{}
	if json.Unmarshal(data, &config) != nil {
		return map[string]interface{}{}
	}
	return config
}

// SaveUserConfig saves user config to disk.
func SaveUserConfig(config map[string]interface{}) error {
	os.MkdirAll(InstallDir(), 0o755)
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFile(), data, 0o644)
}

// GetDefaultChannel returns the default channel string.
func GetDefaultChannel() string {
	return strings.ToLower(GetDefaultRepoName()) + "/stable"
}

// LoadRepoCache loads cached repo data from disk.
func LoadRepoCache() map[string]interface{} {
	data, err := os.ReadFile(repoCacheFile())
	if err != nil {
		return map[string]interface{}{}
	}
	var cache map[string]interface{}
	if json.Unmarshal(data, &cache) != nil {
		return map[string]interface{}{}
	}
	return cache
}

// SaveRepoCache saves repo cache to disk.
func SaveRepoCache(cache map[string]interface{}) error {
	os.MkdirAll(InstallDir(), 0o755)
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(repoCacheFile(), data, 0o644)
}

// InstalledVersion holds info about an installed Camoufox version.
type InstalledVersion struct {
	RepoName       string
	VersionInfo    Version
	Path           string
	IsActive       bool
	IsPrerelease   bool
	AssetID        *int
	AssetSize      *int
	AssetUpdatedAt string
}

// RelativePath returns the relative path from InstallDir.
func (iv InstalledVersion) RelativePath() string {
	return fmt.Sprintf("browsers/%s/%s", iv.RepoName, iv.VersionInfo.FullString())
}

// ChannelPath returns the channel display string.
func (iv InstalledVersion) ChannelPath() string {
	ctype := "stable"
	if iv.IsPrerelease {
		ctype = "prerelease"
	}
	return fmt.Sprintf("%s/%s/%s", iv.RepoName, ctype, iv.VersionInfo.FullString())
}

// ListInstalled scans browsers/ for installed versions.
func ListInstalled() []InstalledVersion {
	var installed []InstalledVersion
	config := LoadUserConfig()
	active, _ := config["active_version"].(string)

	bDir := browsersDir()
	repoDirs, err := os.ReadDir(bDir)
	if err != nil {
		return installed
	}

	for _, repoDir := range repoDirs {
		if !repoDir.IsDir() || strings.HasPrefix(repoDir.Name(), ".") {
			continue
		}
		repoPath := filepath.Join(bDir, repoDir.Name())
		versionDirs, err := os.ReadDir(repoPath)
		if err != nil {
			continue
		}
		for _, vd := range versionDirs {
			if !vd.IsDir() {
				continue
			}
			versionPath := filepath.Join(repoPath, vd.Name())
			versionJSON := filepath.Join(versionPath, "version.json")
			if _, err := os.Stat(versionJSON); os.IsNotExist(err) {
				continue
			}
			ver, err := VersionFromPath(versionPath)
			if err != nil {
				continue
			}
			data, err := os.ReadFile(versionJSON)
			if err != nil {
				continue
			}
			var vData map[string]interface{}
			if json.Unmarshal(data, &vData) != nil {
				continue
			}
			relPath := fmt.Sprintf("browsers/%s/%s", repoDir.Name(), ver.FullString())
			iv := InstalledVersion{
				RepoName:    repoDir.Name(),
				VersionInfo: ver,
				Path:        versionPath,
				IsActive:    relPath == active,
			}
			if pre, ok := vData["prerelease"].(bool); ok {
				iv.IsPrerelease = pre
			}
			if id, ok := vData["asset_id"].(float64); ok {
				ival := int(id)
				iv.AssetID = &ival
			}
			if sz, ok := vData["asset_size"].(float64); ok {
				ival := int(sz)
				iv.AssetSize = &ival
			}
			if at, ok := vData["asset_updated_at"].(string); ok {
				iv.AssetUpdatedAt = at
			}
			installed = append(installed, iv)
		}
	}

	sort.Slice(installed, func(i, j int) bool {
		if installed[i].RepoName != installed[j].RepoName {
			return installed[i].RepoName < installed[j].RepoName
		}
		return installed[j].VersionInfo.Less(installed[i].VersionInfo)
	})
	return installed
}

// GetActivePath returns the path to the active installed version.
func GetActivePath() string {
	config := LoadUserConfig()
	active, _ := config["active_version"].(string)
	if active != "" {
		path := filepath.Join(InstallDir(), active)
		vj := filepath.Join(path, "version.json")
		if _, err := os.Stat(vj); err == nil {
			return path
		}
	}
	// Auto-select if no channel/pin was set
	if _, hasChannel := config["channel"]; !hasChannel {
		if _, hasPinned := config["pinned"]; !hasPinned {
			installed := ListInstalled()
			if len(installed) > 0 {
				config["active_version"] = installed[0].RelativePath()
				SaveUserConfig(config)
				return installed[0].Path
			}
		}
	}
	return ""
}

// SetActive sets the active version by its relative path.
func SetActive(relativePath string) error {
	config := LoadUserConfig()
	config["active_version"] = relativePath
	return SaveUserConfig(config)
}

// FindInstalledVersion finds an installed version by path, build, or full version string.
func FindInstalledVersion(specifier string) string {
	installed := ListInstalled()
	if len(installed) == 0 {
		return ""
	}
	specLower := strings.ToLower(specifier)
	for _, v := range installed {
		if v.RelativePath() == specifier || v.RelativePath() == "browsers/"+specifier {
			return v.Path
		}
		if strings.HasSuffix("browsers/"+v.RepoName+"/"+v.VersionInfo.FullString(), specifier) {
			return v.Path
		}
		if strings.ToLower(v.RepoName+"/"+v.VersionInfo.Build) == specLower {
			return v.Path
		}
		if strings.ToLower(v.VersionInfo.Build) == specLower {
			return v.Path
		}
		if strings.ToLower(v.VersionInfo.FullString()) == specLower {
			return v.Path
		}
		if v.VersionInfo.VersionStr != "" && strings.ToLower(v.VersionInfo.VersionStr) == specLower {
			return v.Path
		}
	}
	return ""
}

// RemoveVersion removes a specific version installation.
func RemoveVersion(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	fmt.Printf("Removing: %s\n", path)
	os.RemoveAll(path)

	parent := filepath.Dir(path)
	if parent != browsersDir() {
		entries, _ := os.ReadDir(parent)
		if len(entries) == 0 {
			os.Remove(parent)
		}
	}
	bDir := browsersDir()
	if entries, err := os.ReadDir(bDir); err == nil && len(entries) == 0 {
		os.Remove(bDir)
	}

	config := LoadUserConfig()
	relPath, err := filepath.Rel(InstallDir(), path)
	if err == nil {
		if config["active_version"] == relPath {
			remaining := ListInstalled()
			if len(remaining) > 0 {
				config["active_version"] = remaining[0].RelativePath()
			} else {
				config["active_version"] = nil
			}
			SaveUserConfig(config)
		}
	}
	return true
}

// GetRepoName returns the display name for a GitHub repo from repos.yml.
func GetRepoName(githubRepo string) string {
	configs, err := LoadRepoConfigs()
	if err == nil {
		for _, rc := range configs {
			for _, r := range rc.Repos {
				if r == githubRepo {
					return strings.ToLower(rc.Name)
				}
			}
		}
	}
	parts := strings.SplitN(githubRepo, "/", 2)
	return strings.ToLower(parts[0])
}
