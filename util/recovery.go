package util

import (
	"runtime/debug"
	"strings"

	"github.com/daoshenzzg/jetcache-go/logger"
)

func WithRecover(fn func()) {
	defer func() {
		if err := recover(); err != nil {
			logger.Error("%+v\n\n%s", err, strings.TrimSpace(string(debug.Stack())))
		}
	}()

	fn()
}
