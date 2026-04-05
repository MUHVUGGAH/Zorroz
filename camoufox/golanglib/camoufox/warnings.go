package camoufox

import (
	"fmt"
	"log"
	"sync"
)

// warningsData holds the loaded warnings.
var (
	warningsOnce sync.Once
	warningsData map[string]string
)

func loadWarnings() map[string]string {
	warningsOnce.Do(func() {
		var raw map[string]interface{}
		if err := LoadYAML("warnings.yml", &raw); err != nil {
			warningsData = map[string]string{}
			return
		}
		warningsData = make(map[string]string, len(raw))
		for k, v := range raw {
			if s, ok := v.(string); ok {
				warningsData[k] = s
			}
		}
	})
	return warningsData
}

// LeakWarning prints a warning about a setting that can cause detection.
func LeakWarning(warningKey string, iKnowWhatImDoing *bool) {
	if iKnowWhatImDoing != nil && *iKnowWhatImDoing {
		return
	}
	data := loadWarnings()
	msg, ok := data[warningKey]
	if !ok {
		return
	}
	if iKnowWhatImDoing != nil {
		msg += "\nIf this is intentional, pass IKnowWhatImDoing: true."
	}
	log.Printf("[LeakWarning] %s", msg)
}

// LeakWarnf prints a formatted leak warning.
func LeakWarnf(format string, args ...interface{}) {
	log.Printf("[LeakWarning] %s", fmt.Sprintf(format, args...))
}
