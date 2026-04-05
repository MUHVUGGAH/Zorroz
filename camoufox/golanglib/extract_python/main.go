// extract_python is a tool to extract all Python files from the camoufox project
// into a separate archive folder, and undo the extraction to restore them.
//
// Run from the golanglib/ directory:
//
//	go run ./extract_python              # Extract Python files
//	go run ./extract_python --undo       # Restore Python files
//	go run ./extract_python --dry-run    # Preview what would be extracted
package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	// Archive directory where Python files will be moved (under golanglib/)
	archiveDir = "_python_archive"
	// Manifest file that tracks all moved files for undo
	manifestFile = "_python_archive/manifest.json"
)

// ManifestEntry records one file that was moved.
type ManifestEntry struct {
	OriginalPath string `json:"original_path"` // Relative to golanglib/
	ArchivePath  string `json:"archive_path"`  // Relative to golanglib/
}

// Manifest is the full record of an extraction.
type Manifest struct {
	Entries []ManifestEntry `json:"entries"`
}

// pythonExtensions are file extensions to extract.
var pythonExtensions = map[string]bool{
	".py": true,
}

// guiOnlyExtensions are non-Python files inside gui/ that are only used by the Python GUI.
var guiOnlyExtensions = map[string]bool{
	".qml": true,
	".ico": true,
	".ttf": true,
}

// isPythonFileInCamoufox checks if a file under camoufox/ should be extracted.
func isPythonFileInCamoufox(relPath string) bool {
	ext := strings.ToLower(filepath.Ext(relPath))

	if pythonExtensions[ext] {
		return true
	}

	// All files under gui/ (QML, assets, etc.) — entirely Python-specific
	parts := strings.Split(filepath.ToSlash(relPath), "/")
	if len(parts) >= 2 && parts[1] == "gui" && guiOnlyExtensions[ext] {
		return true
	}

	return false
}

// topLevelPythonFiles are Python-package files at the golanglib/ level to extract.
var topLevelPythonFiles = []string{
	"pyproject.toml",
	"publish.sh",
}

// findPythonFiles collects all Python-related file paths (relative to golanglib/).
func findPythonFiles(base string) ([]string, error) {
	var files []string

	// 1. Top-level Python package files in golanglib/
	for _, name := range topLevelPythonFiles {
		p := filepath.Join(base, name)
		if _, err := os.Stat(p); err == nil {
			files = append(files, name)
		}
	}

	// 2. Walk the camoufox/ source directory (Python lib code + gui assets)
	camoufoxDir := filepath.Join(base, "camoufox")
	if _, err := os.Stat(camoufoxDir); err == nil {
		err := filepath.WalkDir(camoufoxDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			relPath, err := filepath.Rel(base, path)
			if err != nil {
				return err
			}
			if isPythonFileInCamoufox(relPath) {
				files = append(files, relPath)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	// 3. Example Python files: ../example/*.py (relative to golanglib/)
	repoRoot := filepath.Dir(base) // one level up from golanglib/
	exampleDir := filepath.Join(repoRoot, "example")
	if _, err := os.Stat(exampleDir); err == nil {
		err := filepath.WalkDir(exampleDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if strings.ToLower(filepath.Ext(path)) == ".py" {
				relPath, err := filepath.Rel(base, path)
				if err != nil {
					return err
				}
				files = append(files, relPath)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	// 4. build-tester/scripts/*.py
	buildTesterDir := filepath.Join(repoRoot, "build-tester", "scripts")
	if _, err := os.Stat(buildTesterDir); err == nil {
		err := filepath.WalkDir(buildTesterDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if strings.ToLower(filepath.Ext(path)) != ".py" {
				return nil
			}
			relPath, err := filepath.Rel(base, path)
			if err != nil {
				return err
			}
			files = append(files, relPath)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	// 5. scripts/*.py and scripts/**/*.py (build scripts at repo root)
	scriptsDir := filepath.Join(repoRoot, "scripts")
	if _, err := os.Stat(scriptsDir); err == nil {
		err := filepath.WalkDir(scriptsDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if strings.ToLower(filepath.Ext(path)) != ".py" {
				return nil
			}
			relPath, err := filepath.Rel(base, path)
			if err != nil {
				return err
			}
			files = append(files, relPath)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	// 6. Root-level Python files (multibuild.py, etc.)
	rootPyFiles, _ := filepath.Glob(filepath.Join(repoRoot, "*.py"))
	for _, p := range rootPyFiles {
		relPath, err := filepath.Rel(base, p)
		if err == nil {
			files = append(files, relPath)
		}
	}

	// 7. Additional directories with Python files
	extraDirs := []string{"jsonvv", "patches", "service-tester", "tests"}
	for _, dir := range extraDirs {
		dirPath := filepath.Join(repoRoot, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			continue
		}
		err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if strings.ToLower(filepath.Ext(path)) != ".py" {
				return nil
			}
			relPath, err := filepath.Rel(base, path)
			if err != nil {
				return err
			}
			files = append(files, relPath)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return files, nil
}

// extract moves Python files to the archive directory and writes the manifest.
func extract(base string, dryRun bool) error {
	mPath := filepath.Join(base, manifestFile)

	// Load existing manifest if appending to a previous extraction
	var existingManifest Manifest
	existingPaths := map[string]bool{}
	if data, err := os.ReadFile(mPath); err == nil {
		json.Unmarshal(data, &existingManifest)
		for _, e := range existingManifest.Entries {
			existingPaths[e.OriginalPath] = true
		}
	}

	files, err := findPythonFiles(base)
	if err != nil {
		return fmt.Errorf("scanning for Python files: %w", err)
	}

	// Filter out files already in the archive
	var newFiles []string
	for _, f := range files {
		if !existingPaths[f] {
			newFiles = append(newFiles, f)
		}
	}

	if len(newFiles) == 0 {
		fmt.Println("No new Python files found to extract.")
		if len(existingManifest.Entries) > 0 {
			fmt.Printf("(%d files already in archive)\n", len(existingManifest.Entries))
		}
		return nil
	}

	fmt.Printf("Found %d Python files to extract:\n\n", len(newFiles))
	for _, f := range newFiles {
		fmt.Printf("  %s\n", f)
	}
	if len(existingManifest.Entries) > 0 {
		fmt.Printf("\n(%d files already in archive)\n", len(existingManifest.Entries))
	}
	fmt.Println()

	if dryRun {
		fmt.Println("[dry-run] No files were moved.")
		return nil
	}

	// Create archive directory
	archivePath := filepath.Join(base, archiveDir)
	if err := os.MkdirAll(archivePath, 0o755); err != nil {
		return fmt.Errorf("creating archive dir: %w", err)
	}

	// Start with existing manifest entries
	manifest := existingManifest

	for _, relPath := range newFiles {
		srcPath := filepath.Join(base, relPath)

		// For paths with ".." (files outside golanglib/), compute a safe archive path
		// by resolving the absolute path and making it relative to the repo root,
		// then placing under _python_archive/_repo/
		var dstRel string
		cleanRel := filepath.Clean(relPath)
		if strings.HasPrefix(cleanRel, "..") {
			// File is outside golanglib/ — resolve to repo-root-relative path
			absPath := filepath.Join(base, relPath)
			repoRoot := filepath.Dir(base)
			repoRel, err := filepath.Rel(repoRoot, absPath)
			if err != nil {
				return fmt.Errorf("computing repo-relative path for %s: %w", relPath, err)
			}
			dstRel = filepath.Join(archiveDir, "_repo", repoRel)
		} else {
			dstRel = filepath.Join(archiveDir, relPath)
		}
		dstPath := filepath.Join(base, dstRel)

		// Create destination directory
		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			return fmt.Errorf("creating dir for %s: %w", dstRel, err)
		}

		// Read file content
		content, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("reading %s: %w", relPath, err)
		}

		// Write to archive
		if err := os.WriteFile(dstPath, content, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", dstRel, err)
		}

		// Remove original
		if err := os.Remove(srcPath); err != nil {
			return fmt.Errorf("removing %s: %w", relPath, err)
		}

		manifest.Entries = append(manifest.Entries, ManifestEntry{
			OriginalPath: relPath,
			ArchivePath:  dstRel,
		})

		fmt.Printf("  Moved: %s\n", relPath)
	}

	// Clean up empty directories left behind
	cleanEmptyDirs(filepath.Join(base, "camoufox"))
	repoRoot := filepath.Dir(base)
	for _, dir := range []string{"example", "build-tester", "scripts", "jsonvv", "patches", "service-tester", "tests"} {
		cleanEmptyDirs(filepath.Join(repoRoot, dir))
	}

	// Write manifest
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}
	if err := os.WriteFile(mPath, data, 0o644); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	fmt.Printf("\nExtracted %d files to %s/\n", len(manifest.Entries), archiveDir)
	fmt.Println("Manifest saved. Run with --undo to restore.")
	return nil
}

// undo restores Python files from the archive using the manifest.
func undo(base string) error {
	mPath := filepath.Join(base, manifestFile)

	data, err := os.ReadFile(mPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no manifest found at %s — nothing to undo", manifestFile)
		}
		return fmt.Errorf("reading manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("parsing manifest: %w", err)
	}

	if len(manifest.Entries) == 0 {
		fmt.Println("Manifest is empty — nothing to restore.")
		return nil
	}

	fmt.Printf("Restoring %d files...\n\n", len(manifest.Entries))

	for _, entry := range manifest.Entries {
		srcPath := filepath.Join(base, entry.ArchivePath)
		dstPath := filepath.Join(base, entry.OriginalPath)

		// Create destination directory
		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			return fmt.Errorf("creating dir for %s: %w", entry.OriginalPath, err)
		}

		// Read from archive
		content, err := os.ReadFile(srcPath)
		if err != nil {
			fmt.Printf("  SKIP (missing): %s\n", entry.ArchivePath)
			continue
		}

		// Write back to original location
		if err := os.WriteFile(dstPath, content, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", entry.OriginalPath, err)
		}

		// Remove archive copy
		if err := os.Remove(srcPath); err != nil {
			fmt.Printf("  Warning: could not remove archive copy %s: %v\n", entry.ArchivePath, err)
		}

		fmt.Printf("  Restored: %s\n", entry.OriginalPath)
	}

	// Remove manifest
	os.Remove(mPath)

	// Clean up empty archive directories
	cleanEmptyDirs(filepath.Join(base, archiveDir))
	// Try to remove the archive dir itself if empty
	os.Remove(filepath.Join(base, archiveDir))

	fmt.Printf("\nRestored %d files. Archive cleaned up.\n", len(manifest.Entries))
	return nil
}

// cleanEmptyDirs removes empty directories bottom-up.
func cleanEmptyDirs(root string) {
	// Walk bottom-up by collecting dirs first
	var dirs []string
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			dirs = append(dirs, path)
		}
		return nil
	})

	// Remove bottom-up (reverse order)
	for i := len(dirs) - 1; i >= 0; i-- {
		entries, err := os.ReadDir(dirs[i])
		if err == nil && len(entries) == 0 {
			os.Remove(dirs[i])
		}
	}
}

func main() {
	// Determine base directory (golanglib/)
	// This script expects to be run from the golanglib/ directory
	// or via: go run ./extract_python from golanglib/
	base, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Verify we're in the right directory (golanglib/)
	if _, err := os.Stat(filepath.Join(base, "camoufox")); err != nil {
		fmt.Fprintf(os.Stderr, "Error: 'camoufox/' directory not found.\nRun this from the golanglib/ directory.\n")
		os.Exit(1)
	}

	// Parse args
	dryRun := false
	undoMode := false

	for _, arg := range os.Args[1:] {
		switch arg {
		case "--undo", "-u":
			undoMode = true
		case "--dry-run", "-n":
			dryRun = true
		case "--help", "-h":
			fmt.Println("Usage: go run ./extract_python [flags]")
			fmt.Println()
			fmt.Println("Flags:")
			fmt.Println("  --dry-run, -n    Preview what would be extracted (no changes)")
			fmt.Println("  --undo, -u       Restore previously extracted Python files")
			fmt.Println("  --help, -h       Show this help")
			return
		default:
			fmt.Fprintf(os.Stderr, "Unknown flag: %s\n", arg)
			os.Exit(1)
		}
	}

	if undoMode {
		if err := undo(base); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := extract(base, dryRun); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
