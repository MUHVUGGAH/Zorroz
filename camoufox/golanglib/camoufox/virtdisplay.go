package camoufox

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// VirtualDisplay is a minimal virtual display implementation for Linux.
type VirtualDisplay struct {
	Debug   bool
	proc    *exec.Cmd
	display int
	mu      sync.Mutex
}

// NewVirtualDisplay creates a new virtual display manager.
func NewVirtualDisplay(debug bool) *VirtualDisplay {
	return &VirtualDisplay{
		Debug:   debug,
		display: -1,
	}
}

var xvfbArgs = []string{
	"-screen", "0", "1x1x24",
	"-ac",
	"-nolisten", "tcp",
	"-extension", "RENDER",
	"+extension", "GLX",
	"-extension", "COMPOSITE",
	"-extension", "XVideo",
	"-extension", "XVideo-MotionCompensation",
	"-extension", "XINERAMA",
	"-shmem",
	"-fp", "built-ins",
	"-nocursor",
	"-br",
}

// xvfbPath finds the Xvfb executable.
func xvfbPath() (string, error) {
	path, err := exec.LookPath("Xvfb")
	if err != nil {
		return "", fmt.Errorf("%w: please install Xvfb to use headless mode", ErrCannotFindXvfb)
	}
	return path, nil
}

// Get returns the display string (e.g. ":99").
func (vd *VirtualDisplay) Get() (string, error) {
	if OSName() != "lin" {
		return "", ErrVirtualDisplayNotSupported
	}
	vd.mu.Lock()
	defer vd.mu.Unlock()
	if vd.proc == nil {
		if err := vd.executeXvfb(); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf(":%d", vd.display), nil
}

// Kill terminates the Xvfb process.
func (vd *VirtualDisplay) Kill() {
	vd.mu.Lock()
	defer vd.mu.Unlock()
	if vd.proc != nil && vd.proc.Process != nil {
		if vd.Debug {
			fmt.Println("Terminating virtual display:", vd.display)
		}
		vd.proc.Process.Kill()
		vd.proc = nil
	}
}

func (vd *VirtualDisplay) executeXvfb() error {
	xvfb, err := xvfbPath()
	if err != nil {
		return err
	}
	if vd.display < 0 {
		vd.display = freeDisplay()
	}
	args := append([]string{fmt.Sprintf(":%d", vd.display)}, xvfbArgs...)
	cmd := exec.Command(xvfb, args...)
	if vd.Debug {
		fmt.Println("Starting virtual display:", xvfb, strings.Join(args, " "))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stdout = nil
		cmd.Stderr = nil
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("%w: %v", ErrCannotExecuteXvfb, err)
	}
	vd.proc = cmd
	return nil
}

func freeDisplay() int {
	tmpDir := os.Getenv("TMPDIR")
	if tmpDir == "" {
		tmpDir = "/tmp"
	}
	matches, _ := filepath.Glob(filepath.Join(tmpDir, ".X*-lock"))
	maxDisplay := 99
	for _, m := range matches {
		base := filepath.Base(m)
		// Parse display number from ".X<num>-lock"
		base = strings.TrimPrefix(base, ".X")
		base = strings.TrimSuffix(base, "-lock")
		if n, err := strconv.Atoi(base); err == nil && n > maxDisplay {
			maxDisplay = n
		}
	}
	return maxDisplay + 3 + rand.Intn(17) //nolint:gosec
}
