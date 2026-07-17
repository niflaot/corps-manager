// Package logger creates the process logger.
package logger

import "go.uber.org/zap"

// New creates a development or production Zap logger.
func New(development bool) (*zap.Logger, error) {
	if development {
		return zap.NewDevelopment()
	}
	return zap.NewProduction()
}
