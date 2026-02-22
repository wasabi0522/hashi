package ui

import (
	"os"
	"sync/atomic"

	"github.com/jedib0t/go-pretty/v6/text"
)

// colorState tracks whether colors are disabled.
// 0 = unresolved, 1 = disabled, 2 = enabled.
var colorState atomic.Int32

func isColorDisabled() bool {
	if v := colorState.Load(); v != 0 {
		return v == 1
	}
	_, ok := os.LookupEnv("NO_COLOR")
	if ok {
		if colorState.CompareAndSwap(0, 1) {
			text.DisableColors()
		}
		return true
	}
	colorState.CompareAndSwap(0, 2)
	return false
}

// SetNoColor overrides the color-disabled flag for testing.
func SetNoColor(disabled bool) {
	if disabled {
		colorState.Store(1)
		text.DisableColors()
	} else {
		colorState.Store(2)
		text.EnableColors()
	}
}

// Green formats text in green.
func Green(s string) string {
	if isColorDisabled() {
		return s
	}
	return text.FgGreen.Sprint(s)
}

// Yellow formats text in yellow.
func Yellow(s string) string {
	if isColorDisabled() {
		return s
	}
	return text.FgYellow.Sprint(s)
}
