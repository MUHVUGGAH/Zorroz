package camoufox

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	ipv4Pattern = regexp.MustCompile(`^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$`)
	ipv6Pattern = regexp.MustCompile(`^(([0-9a-fA-F]{0,4}:){1,7}[0-9a-fA-F]{0,4})$`)
)

// Proxy stores proxy information.
type CamoufoxProxy struct {
	Server   string
	Username string
	Password string
	Bypass   string
}

var proxyServerPattern = regexp.MustCompile(`^(?:(?P<schema>\w+)://)?(?P<url>.*?)(?:\:(?P<port>\d+))?$`)

// ParseServer parses the proxy server string.
func (p *CamoufoxProxy) ParseServer() (schema, hostURL, port string, err error) {
	match := proxyServerPattern.FindStringSubmatch(p.Server)
	if match == nil {
		return "", "", "", fmt.Errorf("%w: %s", ErrInvalidProxy, p.Server)
	}
	names := proxyServerPattern.SubexpNames()
	for i, name := range names {
		switch name {
		case "schema":
			schema = match[i]
		case "url":
			hostURL = match[i]
		case "port":
			port = match[i]
		}
	}
	return schema, hostURL, port, nil
}

// AsString builds a full proxy URL string with embedded credentials.
func (p *CamoufoxProxy) AsString() string {
	schema, hostURL, port, err := p.ParseServer()
	if err != nil {
		return p.Server
	}
	if schema == "" {
		schema = "http"
	}
	result := schema + "://"
	if p.Username != "" {
		result += p.Username
		if p.Password != "" {
			result += ":" + p.Password
		}
		result += "@"
	}
	result += hostURL
	if port != "" {
		result += ":" + port
	}
	return result
}

// AsRequestsProxy converts to a map suitable for http.Transport proxying.
func AsRequestsProxy(proxyString string) func(*http.Request) (*url.URL, error) {
	u, err := url.Parse(proxyString)
	if err != nil {
		return nil
	}
	return http.ProxyURL(u)
}

// ValidIPv4 checks if a string is a valid IPv4 address.
func ValidIPv4(ip string) bool {
	return ipv4Pattern.MatchString(ip)
}

// ValidIPv6 checks if a string is a valid IPv6 address.
func ValidIPv6(ip string) bool {
	return ipv6Pattern.MatchString(ip)
}

// ValidateIP validates an IP address string.
func ValidateIP(ip string) error {
	if !ValidIPv4(ip) && !ValidIPv6(ip) {
		return fmt.Errorf("%w: %s", ErrInvalidIP, ip)
	}
	return nil
}

// PublicIP sends a request to public IP APIs to get the exit IP.
// An optional proxy URL string can be provided.
func PublicIP(proxy string) (string, error) {
	urls := []string{
		"https://api.ipify.org",
		"https://checkip.amazonaws.com",
		"https://ipinfo.io/ip",
		"https://icanhazip.com",
		"https://ifconfig.co/ip",
		"https://ipecho.net/plain",
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
	}
	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}
	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: transport,
	}

	var lastErr error
	for _, u := range urls {
		resp, err := client.Get(u)
		if err != nil {
			lastErr = err
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		ip := strings.TrimSpace(string(body))
		if err := ValidateIP(ip); err != nil {
			lastErr = err
			continue
		}
		return ip, nil
	}
	return "", fmt.Errorf("%w: %v", ErrInvalidIP, lastErr)
}
