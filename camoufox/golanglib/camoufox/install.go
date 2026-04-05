package camoufox

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// DownloadAndInstall downloads the browser zip and installs it to the local cache.
// Returns the path to the installed version directory.
func (f *CamoufoxFetcher) DownloadAndInstall() (string, error) {
	if f.URL == "" || f.VersionObj == nil {
		return "", fmt.Errorf("no version selected; call FetchLatest first")
	}

	repoName := GetRepoName(f.GithubRepo)
	installPath := filepath.Join(browsersDir(), repoName, f.VersionObj.FullString())

	// Check if already installed
	vj := filepath.Join(installPath, "version.json")
	if _, err := os.Stat(vj); err == nil {
		fmt.Printf("Already installed: %s\n", installPath)
		relPath, _ := filepath.Rel(InstallDir(), installPath)
		if relPath != "" {
			_ = SetActive(relPath)
		}
		return installPath, nil
	}

	fmt.Printf("Downloading %s...\n", f.URL)

	// Download
	data, err := WebDL(f.URL)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}

	fmt.Printf("Extracting to %s...\n", installPath)

	// Create install directory
	if err := os.MkdirAll(installPath, 0o755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Extract zip
	if err := extractZip(data, installPath); err != nil {
		os.RemoveAll(installPath)
		return "", fmt.Errorf("extraction failed: %w", err)
	}

	// Write version.json
	av := f.toAvailableVersion()
	meta := av.ToMetadata()
	metaJSON, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal version.json: %w", err)
	}
	if err := os.WriteFile(vj, metaJSON, 0o644); err != nil {
		return "", fmt.Errorf("failed to write version.json: %w", err)
	}

	// Set as active
	relPath, _ := filepath.Rel(InstallDir(), installPath)
	if relPath != "" {
		_ = SetActive(relPath)
	}

	// Touch compatibility flag
	flagPath := compatFlag()
	os.MkdirAll(filepath.Dir(flagPath), 0o755)
	os.WriteFile(flagPath, []byte{}, 0o644)

	// Set executable permissions on Linux/macOS
	if runtime.GOOS != "windows" {
		setExecutablePerms(installPath)
	}

	fmt.Printf("Installed: %s\n", f.VersionObj.FullString())
	return installPath, nil
}

// toAvailableVersion converts the fetcher state to an AvailableVersion.
func (f *CamoufoxFetcher) toAvailableVersion() AvailableVersion {
	if f.SelectedVersion != nil {
		return *f.SelectedVersion
	}
	return AvailableVersion{
		Version:      *f.VersionObj,
		URL:          f.URL,
		IsPrerelease: f.IsPrerelease,
	}
}

// extractZip extracts a zip archive from bytes into destDir.
func extractZip(data []byte, destDir string) error {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}

	cleanDest := filepath.Clean(destDir) + string(os.PathSeparator)

	for _, f := range r.File {
		// Sanitize path to prevent zip-slip
		name := filepath.FromSlash(f.Name)
		if strings.Contains(name, "..") {
			continue
		}
		target := filepath.Join(destDir, name)
		cleanTarget := filepath.Clean(target)
		if cleanTarget != filepath.Clean(destDir) && !strings.HasPrefix(cleanTarget, cleanDest) {
			continue
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(target, 0o755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}

		_, copyErr := io.Copy(out, rc)
		out.Close()
		rc.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}

// setExecutablePerms makes binaries executable on Unix systems.
func setExecutablePerms(dir string) {
	for _, name := range []string{"camoufox-bin", "camoufox"} {
		p := filepath.Join(dir, name)
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			os.Chmod(p, 0o755)
		}
	}
	// macOS app bundle
	macBin := filepath.Join(dir, "Camoufox.app", "Contents", "MacOS", "camoufox")
	if info, err := os.Stat(macBin); err == nil && !info.IsDir() {
		os.Chmod(macBin, 0o755)
	}
}
