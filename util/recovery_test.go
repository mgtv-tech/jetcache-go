package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithRecover(t *testing.T) {
	t.Run("test not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			WithRecover(func() {
				panic("panic")
			})
		})
	})
}
