package ui

import (
	"os"
	"sync"

	"github.com/jedib0t/go-pretty/v6/text"
)

var initColors sync.Once

var colorDisabled = sync.OnceValue(func() bool {
	_, ok := os.LookupEnv("NO_COLOR")
	return ok
})

func ensureColors() {
	initColors.Do(func() {
		if colorDisabled() {
			text.DisableColors()
		}
	})
}

// SetNoColor overrides the color-disabled flag for testing.
func SetNoColor(disabled bool) {
	colorDisabled = func() bool { return disabled }
	if disabled {
		text.DisableColors()
	} else {
		text.EnableColors()
	}
}

// Green formats text in green.
func Green(s string) string {
	ensureColors()
	if colorDisabled() {
		return s
	}
	return text.FgGreen.Sprint(s)
}

// Yellow formats text in yellow.
func Yellow(s string) string {
	ensureColors()
	if colorDisabled() {
		return s
	}
	return text.FgYellow.Sprint(s)
}
