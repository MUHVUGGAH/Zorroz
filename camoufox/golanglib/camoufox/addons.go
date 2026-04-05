package camoufox

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// DefaultAddons enum values mirroring addons.py.
type DefaultAddon string

const (
	AddonUBO DefaultAddon = "https://addons.mozilla.org/firefox/downloads/latest/ublock-origin/latest.xpi"
)

// AllDefaultAddons returns all default addon definitions.
func AllDefaultAddons() map[string]DefaultAddon {
	return map[string]DefaultAddon{
		"UBO": AddonUBO,
	}
}

// AddonsDir returns the shared addons directory.
func AddonsDir() string {
	return filepath.Join(InstallDir(), "addons")
}

// ConfirmPaths validates that addon paths are valid extracted addon directories.
func ConfirmPaths(paths []string) error {
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil || !info.IsDir() {
			return fmt.Errorf("%w: %s", ErrInvalidAddonPath, p)
		}
		manifest := filepath.Join(p, "manifest.json")
		if _, err := os.Stat(manifest); os.IsNotExist(err) {
			return fmt.Errorf("%w: manifest.json is missing at %s", ErrInvalidAddonPath, p)
		}
	}
	return nil
}

var addonLock sync.Mutex

// AddDefaultAddons downloads and adds default addons to the list, excluding specified ones.
func AddDefaultAddons(addonsList *[]string, excludeList map[string]bool) {
	if excludeList == nil {
		excludeList = map[string]bool{}
	}
	all := AllDefaultAddons()
	var toDownload []struct {
		name string
		url  string
	}
	for name, url := range all {
		if !excludeList[name] {
			toDownload = append(toDownload, struct {
				name string
				url  string
			}{name, string(url)})
		}
	}
	addonLock.Lock()
	defer addonLock.Unlock()
	MaybeDownloadAddons(toDownload, addonsList)
}

// MaybeDownloadAddons downloads addons if they aren't already present.
func MaybeDownloadAddons(addons []struct{ name, url string }, addonsList *[]string) {
	for _, addon := range addons {
		addonPath := filepath.Join(AddonsDir(), addon.name)
		if _, err := os.Stat(addonPath); err == nil {
			if addonsList != nil {
				*addonsList = append(*addonsList, addonPath)
			}
			continue
		}
		if err := os.MkdirAll(addonPath, 0o755); err != nil {
			fmt.Printf("Failed to create addon dir %s: %v\n", addon.name, err)
			continue
		}
		if err := downloadAndExtractAddon(addon.url, addonPath, addon.name); err != nil {
			fmt.Printf("Failed to download and extract %s: %v\n", addon.name, err)
			continue
		}
		if addonsList != nil {
			*addonsList = append(*addonsList, addonPath)
		}
	}
}

// downloadAndExtractAddon fetches an addon xpi and extracts it.
func downloadAndExtractAddon(url, extractPath, name string) error {
	data, err := WebDL(url)
	if err != nil {
		return fmt.Errorf("downloading addon (%s): %w", name, err)
	}
	return unzipBytes(data, extractPath)
}

// unzipBytes extracts a zip archive from bytes to the given directory.
func unzipBytes(data []byte, dest string) error {
	// Use archive/zip
	r, err := newZipReader(data)
	if err != nil {
		return err
	}
	for _, f := range r.File {
		path := filepath.Join(dest, f.Name) //nolint:gosec - paths come from trusted addon sources
		if f.FileInfo().IsDir() {
			os.MkdirAll(path, 0o755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(path)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = copyLimited(out, rc, f.UncompressedSize64)
		out.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
