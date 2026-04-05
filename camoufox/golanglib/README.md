<div align="center">

# Camoufox Go Interface

#### Lightweight wrapper around the [Playwright Go](https://github.com/playwright-community/playwright-go) API to help launch Camoufox.

</div>

> [!NOTE]
> This is the **Go** port of the Camoufox client library. For the Python version, see the [original pythonlib](https://camoufox.com/python).

---

## What is this?

This Go library wraps around [playwright-go](https://github.com/playwright-community/playwright-go) to automatically generate & inject unique device characteristics (OS, CPU info, navigator, fonts, headers, screen dimensions, viewport size, WebGL, addons, etc.) into Camoufox.

It uses [BrowserForge](https://github.com/MUHVUGGAH/zorroz/tree/main/browserforge) under the hood to generate fingerprints that mimic the statistical distribution of device characteristics in real-world traffic.

In addition, it will also calculate your target geolocation, timezone, and locale to avoid proxy protection ([see demo](https://i.imgur.com/UhSHfaV.png)).

---

## Installation

### 1. Add the Go module

```bash
go get github.com/MUHVUGGAH/zorroz/camoufox/golanglib/camoufox
```

### 2. Download the Camoufox browser binary

The Camoufox browser binary must be installed before use:

```bash
go run github.com/MUHVUGGAH/zorroz/camoufox/golanglib/camoufox/cmd/camoufox-fetch@latest
```

This downloads and installs the latest compatible Camoufox browser to your local cache. No Python or pip required.

Other commands:

```bash
# Show install path
go run github.com/MUHVUGGAH/zorroz/camoufox/golanglib/camoufox/cmd/camoufox-fetch@latest path

# List installed versions
go run github.com/MUHVUGGAH/zorroz/camoufox/golanglib/camoufox/cmd/camoufox-fetch@latest list

# Remove a version
go run github.com/MUHVUGGAH/zorroz/camoufox/golanglib/camoufox/cmd/camoufox-fetch@latest remove <version>
```

### 3. Install Playwright browsers (one-time)

```bash
go run github.com/playwright-community/playwright-go/cmd/playwright@latest install --with-deps
```

To uninstall the browser binary, run `camoufox-fetch remove`.

---

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    camoufox "github.com/MUHVUGGAH/zorroz/camoufox/golanglib/camoufox"
)

func main() {
    headless := false
    opts := camoufox.LaunchOptionsConfig{
        Headless: &headless,
    }

    launchResult, err := camoufox.LaunchOptions(opts)
    if err != nil {
        log.Fatal(err)
    }

    cf, err := camoufox.NewCamoufox(camoufox.NewBrowserOptions{
        LaunchOptions: launchResult.ToPlaywrightLaunchOptionsPtr(),
    })
    if err != nil {
        log.Fatal(err)
    }
    defer cf.Close()

    page, err := cf.Browser.NewPage()
    if err != nil {
        log.Fatal(err)
    }

    if _, err := page.Goto("https://example.com"); err != nil {
        log.Fatal(err)
    }

    title, _ := page.Title()
    fmt.Printf("Page title: %s\n", title)
}
```

See the [`example/`](../example/) directory for more complete examples including async usage and custom headers.

---

## Configuration

### Launch Options

Use `LaunchOptionsConfig` to control browser launch behavior:

```go
headless := true
opts := camoufox.LaunchOptionsConfig{
    Headless: &headless,
    // Proxy: &playwright.Proxy{Server: "http://proxy:8080"},
}
```

### Custom Fingerprint Properties

Pass a config map to override individual fingerprint properties:

```go
cf, err := camoufox.NewCamoufox(camoufox.NewBrowserOptions{
    LaunchOptions: launchResult.ToPlaywrightLaunchOptionsPtr(),
    Config: map[string]interface{}{
        "property": "value",
    },
})
```

Config data not set by the user will be automatically populated using BrowserForge fingerprints.

[[See implemented properties](https://camoufox.com/fingerprint/)]

---

## BrowserForge Integration

Camoufox uses BrowserForge to generate realistic browser fingerprints. The data files are auto-detected from standard locations. If needed, set the `BROWSERFORGE_DATA_DIR` environment variable.

---

## Virtual Display (Linux)

On headless Linux servers, use Xvfb for a virtual display:

```go
vd := camoufox.NewVirtualDisplay(false)
display, err := vd.Get()
if err != nil {
    log.Fatal(err)
}
// display is automatically configured for use
```

---

## Platform Support

| Platform | Status |
|----------|--------|
| Windows  | Supported |
| Linux    | Supported |
| macOS    | Supported |

The Go library is cross-platform. The Camoufox browser binary must be downloaded for each target OS via `camoufox fetch`.

---

## API Reference

| Function | Description |
|----------|-------------|
| `LaunchOptions(cfg)` | Build Playwright launch options with Camoufox settings |
| `NewCamoufox(opts)` | Launch a new Camoufox browser instance |
| `NewVirtualDisplay(debug)` | Create a virtual display (Linux/Xvfb) |
| `LaunchPath(path)` | Get the Camoufox executable path |
| `InstallDir()` | Get the Camoufox installation directory |

---

## Browser Management CLI

The browser binary is managed through the Python `camoufox` CLI. Even though this is a Go library, the CLI is the standard way to manage browser versions:

```bash
camoufox fetch                    # Download latest browser
camoufox list                     # List installed versions
camoufox set official/stable      # Set active channel
camoufox remove                   # Remove browser data
camoufox version                  # Show version info
```

For full CLI documentation, see the [main Camoufox README](../README.md).

---

## Full Documentation

Available at [camoufox.com](https://camoufox.com).
