<h1 align="center">
    BrowserForge
</h1>

<p align="center">
    <a href="https://github.com/MUHVUGGAH/zorroz/blob/main/browserforge/LICENSE">
        <img src="https://img.shields.io/github/license/MUHVUGGAH/zorroz.svg?color=yellow">
    </a>
    <a href="https://go.dev/">
        <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white">
    </a>
</p>

<h4 align="center">
    Intelligent browser header & fingerprint generator
</h4>

---

## What is it?

BrowserForge is a browser header and fingerprint generator that mimics the frequency of different browsers, operating systems, and devices found in the wild.

It uses a Bayesian generative network to produce realistic browser configurations matching real-world traffic distributions.

Originally a reimplementation of [Apify's fingerprint-suite](https://github.com/apify/fingerprint-suite), now rewritten in Go.

---

## Features

- Uses a Bayesian generative network to mimic actual web traffic
- Extremely fast runtime
- Extensive customization options for browsers, operating systems, devices, locales, and HTTP version
- Written with type safety

---

## Installation

```bash
go get github.com/MUHVUGGAH/zorroz/browserforge/browserforge
```

---

## Generating Headers

### Simple usage

```go
package main

import (
    "fmt"
    "github.com/MUHVUGGAH/zorroz/browserforge/browserforge/headers"
)

func main() {
    hg, err := headers.NewHeaderGenerator(
        headers.HeaderGeneratorConfig{
            InputNetworkPath:      "path/to/input-network.json",
            HeaderNetworkPath:     "path/to/header-network.json",
            HeadersOrderPath:      "path/to/headers-order.json",
            BrowserHelperFilePath: "path/to/browser-helper.json",
        },
        nil,  // browsers (nil = all supported)
        nil,  // os (nil = all supported)
        nil,  // device (nil = all supported)
        nil,  // locale (nil = ["en-US"])
        2,    // httpVersion
        false, // strict
    )
    if err != nil {
        panic(err)
    }

    result, err := hg.Generate(nil)
    if err != nil {
        panic(err)
    }
    fmt.Println(result)
}
```

### Constraining headers

#### Browser specifications

Set specifications for browsers, including version ranges and HTTP version:

```go
browsers := []headers.Browser{
    {Name: "chrome", MinVersion: 140, MaxVersion: 145, HTTPVersion: "2"},
    {Name: "firefox", MinVersion: 144, HTTPVersion: "2"},
    {Name: "edge", MaxVersion: 140, HTTPVersion: "1"},
}

hg, err := headers.NewHeaderGenerator(config, browsers, nil, nil, nil, 2, false)
```

#### Per-call overrides

Override defaults on a per-call basis using `GenerateOptions`:

```go
result, err := hg.Generate(&headers.GenerateOptions{
    OS:      []string{"windows"},
    Devices: []string{"desktop"},
    Locales: []string{"en-US", "en", "de"},
})
```

Note that all constraints passed into `NewHeaderGenerator` can be overridden by passing `GenerateOptions` to `Generate`.

#### Generate headers given User-Agent

Headers can be generated given an existing user agent:

```go
result, err := hg.Generate(&headers.GenerateOptions{
    UserAgent: []string{
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36",
    },
})
```

Select from multiple User-Agents based on their frequency in the wild:

```go
result, err := hg.Generate(&headers.GenerateOptions{
    UserAgent: []string{
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:144.0) Gecko/20100101 Firefox/144.0",
        "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36",
    },
})
```

<hr width=50>

## Generating Fingerprints

### Simple usage

```go
package main

import (
    "fmt"
    "github.com/MUHVUGGAH/zorroz/browserforge/browserforge/fingerprints"
    "github.com/MUHVUGGAH/zorroz/browserforge/browserforge/headers"
)

func main() {
    fg, err := fingerprints.NewFingerprintGenerator(
        fingerprints.FingerprintGeneratorConfig{
            FingerprintNetworkPath: "path/to/fingerprint-network.json",
            HeaderGeneratorConfig: headers.HeaderGeneratorConfig{
                InputNetworkPath:      "path/to/input-network.json",
                HeaderNetworkPath:     "path/to/header-network.json",
                HeadersOrderPath:      "path/to/headers-order.json",
                BrowserHelperFilePath: "path/to/browser-helper.json",
            },
        },
        nil,   // screen constraints
        false, // strict
        false, // mockWebRTC
        false, // slim
        nil,   // browsers
        nil,   // os
        nil,   // device
        nil,   // locale
        2,     // httpVersion
        false, // headerStrict
    )
    if err != nil {
        panic(err)
    }

    fp, err := fg.Generate(nil)
    if err != nil {
        panic(err)
    }
    fmt.Println(fp)
}
```

### Constraining fingerprints

#### Screen width/height

Constrain the minimum/maximum screen width and height:

```go
minW, maxW := 100, 1280
minH, maxH := 400, 720

screen, err := fingerprints.NewScreen(&minW, &maxW, &minH, &maxH)
if err != nil {
    panic(err)
}

fg, err := fingerprints.NewFingerprintGenerator(config, screen, false, false, false, nil, nil, nil, nil, 2, false)
```

Note: Not all bounds need to be defined. Pass `nil` for any unconstrained dimension.

#### Per-call overrides

`FingerprintGenerator.Generate` accepts `GenerateOptions` to override defaults:

```go
fp, err := fg.Generate(&fingerprints.GenerateOptions{
    Screen:     screen,
    MockWebRTC: boolPtr(true),
    HeaderOptions: &headers.GenerateOptions{
        OS:      []string{"windows"},
        Devices: []string{"desktop"},
    },
})
```

#### Serializing fingerprints

Serialize a fingerprint to JSON:

```go
jsonStr, err := fp.Dumps()
```

<hr width=50>

## Supported Constraints

| Parameter | Supported Values |
|-----------|-----------------|
| `browser` | `chrome`, `firefox`, `safari`, `edge` |
| `os` | `windows`, `macos`, `linux`, `android`, `ios` |
| `device` | `desktop`, `mobile` |
| `httpVersion` | `1`, `2` |

---
