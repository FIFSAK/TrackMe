package log

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
)

var (
	defaultLogger zerolog.Logger
)

func init() {
	defaultLogger = New()
}

type loggerKey struct{}

// ContextWithLogger adds logger to context
func ContextWithLogger(ctx context.Context, l zerolog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, l)
}

// LoggerFromContext returns logger from context
func LoggerFromContext(ctx context.Context) zerolog.Logger {
	if l, ok := ctx.Value(loggerKey{}).(zerolog.Logger); ok {
		return l
	}
	return defaultLogger
}

func New() zerolog.Logger {
	// Configure zerolog
	zerolog.TimeFieldFormat = time.RFC3339

	// Enable asynchronous logging
	fileWriter, err := os.OpenFile("service.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)

	// Create an asynchronous writer with a buffer
	asyncWriter := diode.NewWriter(fileWriter, 1000, 10*time.Millisecond, func(missed int) {
		fmt.Printf("Logger dropped %d messages", missed)
	})

	// Use multi-writer for console and async file output
	multi := io.MultiWriter(os.Stdout, asyncWriter)

	// Set log level based on DEBUG environment variable
	level := zerolog.InfoLevel
	if os.Getenv("DEBUG") == "true" {
		level = zerolog.DebugLevel
	}

	zerolog.SetGlobalLevel(level)

	// Create logger with timestamp
	logger := zerolog.New(multi).With().Timestamp().Logger()

	// Handle file writer error
	if err != nil {
		logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
		logger.Warn().Err(err).Msg("Unable to set up file logging. Using stdout only.")
	}

	return logger
}
