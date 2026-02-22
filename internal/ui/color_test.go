package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGreen(t *testing.T) {
	out := Green("ok")
	assert.Contains(t, out, "ok")
}

func TestYellow(t *testing.T) {
	out := Yellow("warn")
	assert.Contains(t, out, "warn")
}

func TestSetNoColor(t *testing.T) {
	t.Run("disabled returns plain text", func(t *testing.T) {
		SetNoColor(true)
		t.Cleanup(func() { SetNoColor(false) })

		assert.Equal(t, "plain", Green("plain"))
		assert.Equal(t, "warn", Yellow("warn"))
	})

	t.Run("enabled returns colored text", func(t *testing.T) {
		SetNoColor(false)

		out := Green("ok")
		assert.Contains(t, out, "ok")
		assert.NotEqual(t, "ok", out)
	})
}
