<h1 align="center">Zorroz</h1>

<h4 align="center">A stealthy, anti-detect browser toolkit for web scraping — written in Go</h4>

<p align="center">
    <a href="https://github.com/MUHVUGGAH/zorroz/blob/main/LICENSE">
        <img src="https://img.shields.io/github/license/MUHVUGGAH/zorroz.svg?color=yellow">
    </a>
    <a href="https://go.dev/">
        <img src="https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white">
    </a>
</p>

---

## What is Zorroz?

Zorroz is a Go monorepo containing two core components for stealth browser automation:

| Component | Description |
|-----------|-------------|
| [**camoufox**](camoufox/) | A custom Firefox build with C++-level fingerprint injection, anti-bot evasion, and a Go Playwright wrapper |
| [**browserforge**](browserforge/) | A Bayesian fingerprint & header generator that mimics real-world browser traffic distributions |

Forked from [daijro/camoufox](https://github.com/daijro/camoufox) and [daijro/browserforge](https://github.com/daijro/browserforge), rewritten as a unified Go project.

---

## Quick Start

### 1. Install the Go module

```bash
go get github.com/MUHVUGGAH/zorroz/camoufox/golanglib/camoufox
```

### 2. Download the Camoufox browser binary

```bash
go run github.com/MUHVUGGAH/zorroz/camoufox/golanglib/camoufox/cmd/camoufox-fetch@latest
```

### 3. Install Playwright browsers (one-time)

```bash
go run github.com/playwright-community/playwright-go/cmd/playwright@latest install --with-deps
```

### 4. Run it

```go
package main

import (
    "fmt"
    "log"

    camoufox "github.com/MUHVUGGAH/zorroz/camoufox/golanglib/camoufox"
)

func main() {
    headless := true
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

---

## Examples

### Custom fingerprint properties

```go
cf, err := camoufox.NewCamoufox(camoufox.NewBrowserOptions{
    LaunchOptions: launchResult.ToPlaywrightLaunchOptionsPtr(),
    Config: map[string]interface{}{
        "property": "value",
    },
})
```

Unset properties are automatically populated using BrowserForge fingerprints. See [implemented properties](https://camoufox.com/fingerprint/).

### Concurrent scraping with goroutines

```go
func main() {
    headless := true
    ac, err := camoufox.NewAsyncCamoufox(camoufox.LaunchOptionsConfig{
        Headless: &headless,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer ac.Close()

    context, err := ac.Browser().NewContext()
    if err != nil {
        log.Fatal(err)
    }

    var wg sync.WaitGroup
    for _, url := range urls {
        wg.Add(1)
        go func(u string) {
            defer wg.Done()
            page, _ := context.NewPage()
            page.Goto(u)
            body, _ := page.Locator("body").InnerText()
            fmt.Printf("%s: %s\n", u, body[:100])
        }(url)
    }
    wg.Wait()
}
```

### Custom headers

```go
page, err := cf.Browser.NewPage(playwright.BrowserNewPageOptions{
    ExtraHttpHeaders: map[string]string{"accept-encoding": "identity"},
})
```

See the [`camoufox/example/`](camoufox/example/) directory for complete working examples.

---

## Highlights

- **Invisible to anti-bot systems** — Page agent is hidden from JavaScript inspection
- **C++-level fingerprint injection** — No JS injection to detect. Navigator, screen, WebGL, fonts, audio, WebRTC, geolocation, timezone, and more
- **Realistic fingerprint rotation** — BrowserForge generates statistically accurate device profiles matching real-world traffic
- **Concurrent-friendly** — Native goroutine support for parallel scraping
- **Human-like mouse movement** — Built-in natural cursor motion algorithm
- **Ad blocking** — Bundled uBlock Origin with privacy filters
- **Cross-platform** — Windows, Linux, macOS

---

## Project Structure

```
zorroz/
├── camoufox/                    # Anti-detect Firefox browser
│   ├── golanglib/camoufox/      # Go Playwright wrapper (main library)
│   ├── patches/                 # Firefox C++ fingerprint patches
│   ├── additions/               # Browser UI & config modifications
│   ├── bundle/                  # System fonts (Win/Mac/Linux)
│   ├── example/                 # Working Go examples
│   └── scripts/                 # Build system
├── browserforge/                # Fingerprint generator
│   └── browserforge/            # Go package (headers, fingerprints, injectors)
└── LICENSE                      # MPL-2.0
```

---

## Documentation

| Topic | Link |
|-------|------|
| Go library API | [camoufox/golanglib/](camoufox/golanglib/) |
| BrowserForge API | [browserforge/](browserforge/) |
| Fingerprint properties | [camoufox.com/fingerprint](https://camoufox.com/fingerprint/) |
| Stealth details | [camoufox.com/stealth](https://camoufox.com/stealth) |
| Building from source | [camoufox/README.md](camoufox/README.md#build-system) |

---

## License

[Mozilla Public License 2.0](LICENSE)
