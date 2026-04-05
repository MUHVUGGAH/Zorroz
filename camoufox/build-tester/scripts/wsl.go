package scripts

import (
	"context"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var (
	wslUNCPathPattern   = regexp.MustCompile(`(?i)^[\\/]{2}(?:wsl\$|wsl\.localhost)[\\/][^\\/]+[\\/](.*)`)
	windowsDrivePattern = regexp.MustCompile(`^([A-Za-z]):\\`)
	hostIPPattern       = regexp.MustCompile(`via\s+(\d+\.\d+\.\d+\.\d+)`)
)

// IsELFBinary reports whether the file starts with the ELF magic header.
func IsELFBinary(filePath string) bool {
	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()

	header := make([]byte, 4)
	if _, err := f.Read(header); err != nil {
		return false
	}

	return string(header) == "\x7fELF"
}

// GetWindowsHostIP asks WSL for the default route and extracts the host IP.
func GetWindowsHostIP() string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "wsl", "bash", "-lc", "ip route show default").Output()
	if err != nil {
		return "localhost"
	}

	match := hostIPPattern.FindStringSubmatch(string(out))
	if len(match) < 2 {
		return "localhost"
	}

	return match[1]
}

// WindowsToWSLPath converts Windows or WSL UNC paths into Linux-style paths.
func WindowsToWSLPath(winPath string) string {
	if match := wslUNCPathPattern.FindStringSubmatch(winPath); len(match) >= 2 {
		return "/" + strings.ReplaceAll(match[1], `\`, "/")
	}

	match := windowsDrivePattern.FindStringSubmatch(winPath)
	if len(match) < 2 {
		return strings.ReplaceAll(winPath, `\`, "/")
	}

	return "/mnt/" + strings.ToLower(match[1]) + "/" + strings.ReplaceAll(winPath[3:], `\`, "/")
}
