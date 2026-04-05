package webgl

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"path/filepath"
	"runtime"
	"strings"

	_ "modernc.org/sqlite"
)

// DBPath resolves to the bundled WebGL fingerprint database beside this file.
var DBPath = func() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "webgl_data.db"
	}
	return filepath.Join(filepath.Dir(file), "webgl_data.db")
}()

// validOSColumns matches the OS keys in pkgman.OSArchMatrix.
var validOSColumns = map[string]struct{}{
	"win": {},
	"mac": {},
	"lin": {},
}

var osDisplayName = map[string]string{
	"win": "Win",
	"mac": "Mac",
	"lin": "Lin",
}

type weightedWebGLRow struct {
	Data        string
	Probability float64
}

// VendorRendererPair identifies a concrete WebGL vendor/renderer combination.
type VendorRendererPair struct {
	Vendor   string
	Renderer string
}

// SampleWebGL returns a weighted random WebGL fingerprint for the requested OS.
// If vendor and renderer are both provided, it validates and returns that exact pair.
func SampleWebGL(osName string, vendor, renderer *string) (map[string]any, error) {
	if _, ok := validOSColumns[osName]; !ok {
		return nil, fmt.Errorf("invalid OS: %s. Must be one of: win, mac, lin", osName)
	}

	db, err := sql.Open("sqlite", DBPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if vendor != nil && renderer != nil {
		query := fmt.Sprintf(
			"SELECT vendor, renderer, data, %s FROM webgl_fingerprints WHERE vendor = ? AND renderer = ?",
			osName,
		)
		var resultVendor, resultRenderer, data string
		var probability float64
		err := db.QueryRow(query, *vendor, *renderer).Scan(&resultVendor, &resultRenderer, &data, &probability)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no WebGL data found for vendor %q and renderer %q", *vendor, *renderer)
		}
		if err != nil {
			return nil, err
		}
		if probability <= 0 {
			pairs, err := possiblePairsForOS(db, osName)
			if err != nil {
				return nil, err
			}
			return nil, fmt.Errorf(
				"vendor %q and renderer %q combination not valid for %s.\nPossible pairs: %s",
				*vendor,
				*renderer,
				osDisplayName[osName],
				formatPairs(pairs),
			)
		}
		return decodeWebGLData(data)
	}

	query := fmt.Sprintf(
		"SELECT vendor, renderer, data, %s FROM webgl_fingerprints WHERE %s > 0",
		osName,
		osName,
	)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []weightedWebGLRow
	for rows.Next() {
		var vendorName, rendererName, data string
		var probability float64
		if err := rows.Scan(&vendorName, &rendererName, &data, &probability); err != nil {
			return nil, err
		}
		results = append(results, weightedWebGLRow{
			Data:        data,
			Probability: probability,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no WebGL data found for OS: %s", osName)
	}

	index, err := weightedChoice(results)
	if err != nil {
		return nil, err
	}
	return decodeWebGLData(results[index].Data)
}

// GetPossiblePairs lists all vendor/renderer pairs that are valid for each OS.
func GetPossiblePairs() (map[string][]VendorRendererPair, error) {
	db, err := sql.Open("sqlite", DBPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	result := make(map[string][]VendorRendererPair, len(validOSColumns))
	for osName := range validOSColumns {
		pairs, err := possiblePairsForOS(db, osName)
		if err != nil {
			return nil, err
		}
		result[osName] = pairs
	}

	return result, nil
}

func possiblePairsForOS(db *sql.DB, osName string) ([]VendorRendererPair, error) {
	query := fmt.Sprintf(
		"SELECT DISTINCT vendor, renderer FROM webgl_fingerprints WHERE %s > 0 ORDER BY %s DESC",
		osName,
		osName,
	)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pairs []VendorRendererPair
	for rows.Next() {
		var pair VendorRendererPair
		if err := rows.Scan(&pair.Vendor, &pair.Renderer); err != nil {
			return nil, err
		}
		pairs = append(pairs, pair)
	}
	return pairs, rows.Err()
}

func decodeWebGLData(data string) (map[string]any, error) {
	var result map[string]any
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, err
	}
	return result, nil
}

func weightedChoice(results []weightedWebGLRow) (int, error) {
	var total float64
	for _, result := range results {
		if result.Probability > 0 {
			total += result.Probability
		}
	}
	if total <= 0 {
		return 0, fmt.Errorf("no positive WebGL sampling probabilities found")
	}

	target := rand.Float64() * total
	var cumulative float64
	for index, result := range results {
		if result.Probability <= 0 {
			continue
		}
		cumulative += result.Probability
		if target < cumulative {
			return index, nil
		}
	}

	return len(results) - 1, nil
}

func formatPairs(pairs []VendorRendererPair) string {
	formatted := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		formatted = append(formatted, fmt.Sprintf("(%s, %s)", pair.Vendor, pair.Renderer))
	}
	return strings.Join(formatted, ", ")
}
