// Copyright (c) The go-conntrack Authors.
// Licensed under the Apache License 2.0.

// Package slogadapter provides a logging adapter for [logging.Logger].
package slogadapter

import (
	"log/slog"

	"github.com/marefr/go-conntrack/v2/logging"
)

func Logger(logger *slog.Logger) logging.LoggerFunc {
	if logger == nil {
		return nil
	}

	return logger.LogAttrs
}
