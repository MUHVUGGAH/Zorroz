package browserforge

import "fmt"

// Download is DEPRECATED.
// As of v1.2.4, model files are included in their own package dependency.
func Download(headers, fingerprints bool) {
	fmt.Println("BrowserForge model files are now bundled in their own Python package dependency. This command is deprecated.")
}

// DownloadIfNotExists is DEPRECATED.
// As of v1.2.4, model files are included in their own package dependency.
func DownloadIfNotExists() {}

// IsDownloaded is DEPRECATED.
// As of v1.2.4, model files are included in their own package dependency.
// Returns true by default.
func IsDownloaded() bool {
	return true
}

// Remove is DEPRECATED.
// As of v1.2.4, model files are included in their own package dependency.
func Remove() {}
