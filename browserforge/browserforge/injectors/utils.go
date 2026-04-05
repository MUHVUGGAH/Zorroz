package injectors

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"

	"browserforge/fingerprints"

	"github.com/ulikunitz/xz"
)

// RequestHeaders are HTTP headers that depend on the request and should be filtered out.
var RequestHeaders = map[string]bool{
	"accept-encoding":           true,
	"accept":                    true,
	"cache-control":             true,
	"pragma":                    true,
	"sec-fetch-dest":            true,
	"sec-fetch-mode":            true,
	"sec-fetch-site":            true,
	"sec-fetch-user":            true,
	"upgrade-insecure-requests": true,
}

// OnlyInjectableHeaders filters out request-dependent headers and leaves only the browser-wide ones.
func OnlyInjectableHeaders(headers map[string]string, browserName string) map[string]string {
	filtered := make(map[string]string)
	for k, v := range headers {
		if !RequestHeaders[strings.ToLower(k)] {
			filtered[k] = v
		}
	}

	// Chromium-based controlled browsers do not support `te` header.
	// Remove the `te` header if the browser is not Firefox.
	if browserName == "" || !strings.Contains(strings.ToLower(browserName), "firefox") {
		delete(filtered, "te")
		delete(filtered, "Te")
	}

	return filtered
}

// InjectFunction returns the JavaScript injection code for the given fingerprint.
func InjectFunction(fp *fingerprints.Fingerprint, utilsJSContent string) (string, error) {
	fpJSON, err := fp.Dumps()
	if err != nil {
		return "", fmt.Errorf("failed to serialize fingerprint: %w", err)
	}

	historyLength := rand.Intn(5) + 1

	return fmt.Sprintf(`
    (()=>{
        %s

        const fp = %s;
        const {
            battery,
            navigator: {
                userAgentData,
                webdriver,
                ...navigatorProps
            },
            screen: allScreenProps,
            videoCard,
            audioCodecs,
            videoCodecs,
            mockWebRTC,
        } = fp;
        
        slim = fp.slim;
        
        const historyLength = %d;
        
        const {
            outerHeight,
            outerWidth,
            devicePixelRatio,
            innerWidth,
            innerHeight,
            screenX,
            pageXOffset,
            pageYOffset,
            clientWidth,
            clientHeight,
            hasHDR,
            ...newScreen
        } = allScreenProps;

        const windowScreenProps = {
            innerHeight,
            outerHeight,
            outerWidth,
            innerWidth,
            screenX,
            pageXOffset,
            pageYOffset,
            devicePixelRatio,
        };

        const documentScreenProps = {
            clientHeight,
            clientWidth,
        };

        runHeadlessFixes();
        if (mockWebRTC) blockWebRTC();
        if (slim) {
            window['slim'] = true;
        }
        overrideIntlAPI(navigatorProps.language);
        overrideStatic();
        if (userAgentData) {
            overrideUserAgentData(userAgentData);
        }
        if (window.navigator.webdriver) {
            navigatorProps.webdriver = false;
        }
        overrideInstancePrototype(window.navigator, navigatorProps);
        overrideInstancePrototype(window.screen, newScreen);
        overrideWindowDimensionsProps(windowScreenProps);
        overrideDocumentDimensionsProps(documentScreenProps);
        overrideInstancePrototype(window.history, { length: historyLength });
        overrideWebGl(videoCard);
        overrideCodecs(audioCodecs, videoCodecs);
        overrideBattery(battery);
    })()
    `, utilsJSContent, fpJSON, historyLength), nil
}

// UtilsJS opens and decompresses the utils.js.xz file and returns it as a string.
func UtilsJS(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	r, err := xz.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("failed to create xz reader: %w", err)
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("failed to decompress utils.js: %w", err)
	}

	return string(data), nil
}

// GenerateFingerprint generates a fingerprint if one doesn't exist.
func GenerateFingerprint(fp *fingerprints.Fingerprint, generator *fingerprints.FingerprintGenerator, opts *fingerprints.GenerateOptions) (*fingerprints.Fingerprint, error) {
	if fp != nil {
		return fp, nil
	}
	return generator.Generate(opts)
}
