package camoufox

import (
	"errors"
	"fmt"
)

// Sentinel errors mirroring the Python exception hierarchy in exceptions.py.
var (
	ErrUnsupportedVersion         = errors.New("camoufox executable is outdated")
	ErrMissingRelease             = errors.New("required GitHub release asset is missing")
	ErrUnsupportedArchitecture    = errors.New("architecture is not supported")
	ErrUnsupportedOS              = errors.New("OS is not supported")
	ErrUnknownProperty            = errors.New("unknown property")
	ErrInvalidPropertyType        = errors.New("invalid property type")
	ErrInvalidAddonPath           = errors.New("invalid addon path")
	ErrInvalidDebugPort           = errors.New("invalid debug port")
	ErrMissingDebugPort           = errors.New("missing debug port")
	ErrLocaleError                = errors.New("locale error")
	ErrInvalidIP                  = errors.New("invalid IP address")
	ErrInvalidProxy               = errors.New("invalid proxy")
	ErrUnknownIPLocation          = errors.New("IP location unknown")
	ErrInvalidLocale              = errors.New("invalid locale")
	ErrUnknownTerritory           = errors.New("unknown territory")
	ErrUnknownLanguage            = errors.New("unknown language")
	ErrNotInstalledGeoIPExtra     = errors.New("geoip extra not installed")
	ErrNonFirefoxFingerprint      = errors.New("non-Firefox fingerprint")
	ErrInvalidOS                  = errors.New("invalid target OS")
	ErrVirtualDisplay             = errors.New("virtual display error")
	ErrCannotFindXvfb             = fmt.Errorf("%w: cannot find Xvfb", ErrVirtualDisplay)
	ErrCannotExecuteXvfb          = fmt.Errorf("%w: cannot execute Xvfb", ErrVirtualDisplay)
	ErrVirtualDisplayNotSupported = fmt.Errorf("%w: only supported on Linux", ErrVirtualDisplay)
	ErrCamoufoxNotInstalled       = errors.New("camoufox is not installed")
)

// InvalidLocaleError wraps ErrInvalidLocale with a specific message.
func InvalidLocaleError(locale string) error {
	return fmt.Errorf(
		"%w: '%s'. Must be either a region, language, language-region, or language-script-region",
		ErrInvalidLocale,
		locale,
	)
}
