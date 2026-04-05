package camoufox

// CONSTRAINTS mirrors __version__.py's CONSTRAINTS class.
var CONSTRAINTS = struct {
	MinVersion string
	MaxVersion string
}{
	MinVersion: "alpha.1",
	MaxVersion: "1",
}

// ConstraintsRange returns the version range as a string.
func ConstraintsRange() string {
	return ">=" + CONSTRAINTS.MinVersion + ", <" + CONSTRAINTS.MaxVersion
}
