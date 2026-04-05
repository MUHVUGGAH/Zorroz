package camoufox

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"
)

// CamelCase converts a snake_case string to camelCase.
func CamelCase(s string) string {
	if len(s) < 2 {
		return s
	}
	parts := strings.Split(strings.ToLower(s), "_")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			runes := []rune(parts[i])
			runes[0] = unicode.ToUpper(runes[0])
			parts[i] = string(runes)
		}
	}
	return strings.Join(parts, "")
}

// ToCamelCaseDict converts a map's keys to camelCase.
func ToCamelCaseDict(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(data))
	for k, v := range data {
		result[CamelCase(k)] = v
	}
	return result
}

// LaunchServer launches a Playwright server.
// Takes the same arguments as LaunchOptions, prints the websocket endpoint.
func LaunchServer(opts LaunchOptionsConfig) error {
	config, err := LaunchOptions(opts)
	if err != nil {
		return err
	}

	launchScriptPath := filepath.Join(LocalDataDir(), "launchServer.js")
	if _, err := os.Stat(launchScriptPath); os.IsNotExist(err) {
		return fmt.Errorf("launchServer.js not found at %s", launchScriptPath)
	}

	// Build config dict for the Node.js script
	configDict := map[string]interface{}{
		"executablePath":   config.ExecutablePath,
		"args":             config.Args,
		"env":              config.Env,
		"firefoxUserPrefs": config.FirefoxUserPrefs,
		"headless":         config.Headless,
	}
	if len(config.Proxy) > 0 {
		configDict["proxy"] = config.Proxy
	}

	camelConfig := ToCamelCaseDict(configDict)
	data, err := json.Marshal(camelConfig)
	if err != nil {
		return err
	}

	// The Node.js script expects base64-encoded JSON on stdin
	encoded := base64.StdEncoding.EncodeToString(data)

	// Find the Node.js executable (bundled with Playwright)
	nodejs := findNodeJS()
	if nodejs == "" {
		return fmt.Errorf("Node.js executable not found")
	}

	cmd := exec.Command(nodejs, launchScriptPath)
	cmd.Dir = filepath.Join(filepath.Dir(nodejs), "package")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	_, err = stdin.Write([]byte(encoded))
	stdin.Close()
	if err != nil {
		return err
	}

	return cmd.Wait()
}

// findNodeJS attempts to find a Node.js executable.
func findNodeJS() string {
	// Try common names
	for _, name := range []string{"node", "nodejs"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	return ""
}
