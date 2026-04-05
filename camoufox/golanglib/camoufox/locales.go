package camoufox

import (
	"encoding/xml"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
)

// Locale stores locale, region, and script information.
type Locale struct {
	Language string
	Region   string
	Script   string
}

// AsString returns "language-region" or just "language".
func (l Locale) AsString() string {
	if l.Region != "" {
		return l.Language + "-" + l.Region
	}
	return l.Language
}

// AsConfig converts the locale to Camoufox intl config keys.
func (l Locale) AsConfig() map[string]string {
	data := map[string]string{
		"locale:region":   l.Region,
		"locale:language": l.Language,
	}
	if l.Script != "" {
		data["locale:script"] = l.Script
	}
	return data
}

// Geolocation stores geolocation information.
type Geolocation struct {
	Loc       Locale
	Longitude float64
	Latitude  float64
	Timezone  string
	Accuracy  float64
}

// GeoAsConfig converts the geolocation to a config dictionary.
func (g Geolocation) GeoAsConfig() map[string]interface{} {
	data := map[string]interface{}{
		"geolocation:longitude": g.Longitude,
		"geolocation:latitude":  g.Latitude,
		"timezone":              g.Timezone,
	}
	for k, v := range g.Loc.AsConfig() {
		data[k] = v
	}
	if g.Accuracy > 0 {
		data["geolocation:accuracy"] = g.Accuracy
	}
	return data
}

// VerifyLocale validates that a locale string is well-formed.
// Accepts "language", "language-region", "language-script-region".
func VerifyLocale(loc string) error {
	parts := strings.Split(loc, "-")
	if len(parts) == 0 || len(parts[0]) < 2 {
		return InvalidLocaleError(loc)
	}
	return nil
}

// NormalizeLocale normalizes and validates a locale code.
// Expects formats: "language-region" or "language-script-region".
func NormalizeLocale(locale string) (Locale, error) {
	if err := VerifyLocale(locale); err != nil {
		return Locale{}, err
	}
	parts := strings.Split(locale, "-")
	l := Locale{Language: strings.ToLower(parts[0])}
	switch len(parts) {
	case 2:
		l.Region = strings.ToUpper(parts[1])
	case 3:
		l.Script = parts[1]
		l.Region = strings.ToUpper(parts[2])
	default:
		if len(parts) == 1 {
			return Locale{}, InvalidLocaleError(locale)
		}
		l.Region = strings.ToUpper(parts[len(parts)-1])
	}
	return l, nil
}

// HandleLocale handles a locale input, normalizing it if necessary.
func HandleLocale(locale string, ignoreRegion bool) (Locale, error) {
	if len(locale) > 3 {
		return NormalizeLocale(locale)
	}
	// Try as region
	l, err := DefaultSelector().FromRegion(locale)
	if err == nil {
		return l, nil
	}
	if ignoreRegion {
		if err := VerifyLocale(locale); err != nil {
			return Locale{}, err
		}
		return Locale{Language: locale}, nil
	}
	// Try as language
	l, err = DefaultSelector().FromLanguage(locale)
	if err == nil {
		return l, nil
	}
	return Locale{}, InvalidLocaleError(locale)
}

// HandleLocales processes a list of locale strings into the config.
func HandleLocales(locales interface{}, config map[string]interface{}) error {
	var localeList []string
	switch v := locales.(type) {
	case string:
		for _, s := range strings.Split(v, ",") {
			localeList = append(localeList, strings.TrimSpace(s))
		}
	case []string:
		localeList = v
	default:
		return fmt.Errorf("locales must be a string or []string")
	}
	if len(localeList) == 0 {
		return nil
	}
	intlLocale, err := HandleLocale(localeList[0], false)
	if err != nil {
		return err
	}
	for k, v := range intlLocale.AsConfig() {
		config[k] = v
	}
	if len(localeList) < 2 {
		return nil
	}
	seen := map[string]bool{}
	var parts []string
	for _, loc := range localeList {
		l, err := HandleLocale(loc, true)
		if err != nil {
			return err
		}
		s := l.AsString()
		if !seen[s] {
			seen[s] = true
			parts = append(parts, s)
		}
	}
	config["locale:all"] = strings.Join(parts, ", ")
	return nil
}

// territoryInfo XML structs for parsing territoryInfo.xml.
type territoryInfoXML struct {
	XMLName     xml.Name       `xml:"territoryInfo"`
	Territories []territoryXML `xml:"territory"`
}

type territoryXML struct {
	Type                string                  `xml:"type,attr"`
	Population          float64                 `xml:"population,attr"`
	LiteracyPercent     float64                 `xml:"literacyPercent,attr"`
	LanguagePopulations []languagePopulationXML `xml:"languagePopulation"`
}

type languagePopulationXML struct {
	Type              string  `xml:"type,attr"`
	PopulationPercent float64 `xml:"populationPercent,attr"`
}

// StatisticalLocaleSelector selects a random locale based on statistical data.
type StatisticalLocaleSelector struct {
	territories []territoryXML
}

var defaultSelector *StatisticalLocaleSelector

// DefaultSelector returns the singleton StatisticalLocaleSelector.
func DefaultSelector() *StatisticalLocaleSelector {
	if defaultSelector == nil {
		s, err := NewStatisticalLocaleSelector()
		if err != nil {
			// Return empty selector on error
			return &StatisticalLocaleSelector{}
		}
		defaultSelector = s
	}
	return defaultSelector
}

// NewStatisticalLocaleSelector loads territoryInfo.xml and creates a selector.
func NewStatisticalLocaleSelector() (*StatisticalLocaleSelector, error) {
	xmlPath := filepath.Join(LocalDataDir(), "territoryInfo.xml")
	data, err := os.ReadFile(xmlPath)
	if err != nil {
		return nil, err
	}
	var info territoryInfoXML
	if err := xml.Unmarshal(data, &info); err != nil {
		return nil, err
	}
	return &StatisticalLocaleSelector{territories: info.Territories}, nil
}

// FromRegion gets a random locale based on the territory ISO code.
func (s *StatisticalLocaleSelector) FromRegion(region string) (Locale, error) {
	region = strings.ToUpper(region)
	for _, terr := range s.territories {
		if terr.Type != region {
			continue
		}
		if len(terr.LanguagePopulations) == 0 {
			return Locale{}, fmt.Errorf("%w: no language data for %s", ErrUnknownTerritory, region)
		}
		// Weighted random selection
		languages := make([]string, len(terr.LanguagePopulations))
		weights := make([]float64, len(terr.LanguagePopulations))
		total := 0.0
		for i, lp := range terr.LanguagePopulations {
			languages[i] = lp.Type
			weights[i] = lp.PopulationPercent
			total += lp.PopulationPercent
		}
		r := rand.Float64() * total
		cumulative := 0.0
		chosen := languages[0]
		for i, w := range weights {
			cumulative += w
			if r <= cumulative {
				chosen = languages[i]
				break
			}
		}
		chosen = strings.ReplaceAll(chosen, "_", "-")
		return NormalizeLocale(chosen + "-" + region)
	}
	return Locale{}, fmt.Errorf("%w: %s", ErrUnknownTerritory, region)
}

// FromLanguage gets a random locale based on the language.
func (s *StatisticalLocaleSelector) FromLanguage(language string) (Locale, error) {
	type regionWeight struct {
		region string
		weight float64
	}
	var candidates []regionWeight
	total := 0.0
	for _, terr := range s.territories {
		for _, lp := range terr.LanguagePopulations {
			if lp.Type == language {
				w := lp.PopulationPercent * terr.LiteracyPercent / 10000 * terr.Population
				candidates = append(candidates, regionWeight{region: terr.Type, weight: w})
				total += w
				break
			}
		}
	}
	if len(candidates) == 0 {
		return Locale{}, fmt.Errorf("%w: %s", ErrUnknownLanguage, language)
	}
	r := rand.Float64() * total
	cumulative := 0.0
	chosen := candidates[0].region
	for _, c := range candidates {
		cumulative += c.weight
		if r <= cumulative {
			chosen = c.region
			break
		}
	}
	return NormalizeLocale(language + "-" + chosen)
}
