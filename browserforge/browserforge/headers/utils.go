package headers

import "strings"

// GetUserAgent retrieves the User-Agent from the headers dictionary.
// Returns the value and true if found, or empty string and false otherwise.
func GetUserAgent(headers map[string]string) (string, bool) {
	if ua, ok := headers["User-Agent"]; ok {
		return ua, true
	}
	if ua, ok := headers["user-agent"]; ok {
		return ua, true
	}
	return "", false
}

// GetBrowser determines the browser name from the User-Agent string.
// Returns empty string if no known browser is detected.
func GetBrowser(userAgent string) string {
	if strings.Contains(userAgent, "Firefox") || strings.Contains(userAgent, "FxiOS") {
		return "firefox"
	}
	if strings.Contains(userAgent, "Chrome") || strings.Contains(userAgent, "CriOS") {
		return "chrome"
	}
	if strings.Contains(userAgent, "Safari") {
		return "safari"
	}
	if strings.Contains(userAgent, "Edge") || strings.Contains(userAgent, "EdgA") ||
		strings.Contains(userAgent, "Edg") || strings.Contains(userAgent, "EdgiOS") {
		return "edge"
	}
	return ""
}

var pascalizeUpper = map[string]bool{
	"dnt": true,
	"rtt": true,
	"ect": true,
}

// Pascalize converts a header name to Pascal-Case.
func Pascalize(name string) string {
	if strings.HasPrefix(name, ":") || strings.HasPrefix(name, "sec-ch-ua") {
		return name
	}
	if pascalizeUpper[name] {
		return strings.ToUpper(name)
	}
	return titleCase(name)
}

// titleCase converts a hyphen-separated string to Title-Case.
func titleCase(s string) string {
	parts := strings.Split(s, "-")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "-")
}

// PascalizeHeaders converts all header names to Pascal-Case.
func PascalizeHeaders(headers map[string]string) map[string]string {
	result := make(map[string]string, len(headers))
	for key, value := range headers {
		result[Pascalize(key)] = value
	}
	return result
}

// Tuplify converts a single string to a slice, passes through slices,
// and returns nil for nil input.
func Tuplify(obj interface{}) []string {
	if obj == nil {
		return nil
	}
	switch v := obj.(type) {
	case string:
		return []string{v}
	case []string:
		return v
	}
	return nil
}
