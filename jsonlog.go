// Package jsonlog provides structured JSON logging with arbitrary data. It
// supports contexts and can extract values from these.
package jsonlog

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"time"
)

// LogLevel represents a level of logging: the Logger is setup with one and
// will only log messages with a log level superior or equal to its own.
type LogLevel uint

// contextKey lets the jsonlog module define keys to Context.Value which will
// not collide with anything else.
type contextKey uint

// Logger logs messages to an io.Writer in JSON format, possibly extracting
// values from its Context.
type Logger struct {
	encoder     *json.Encoder
	logLevel    LogLevel
	contextKeys map[interface{}]string
	context     context.Context
}

// message represents a single messaged logged by a Logger.
type message struct {
	Level   string                 `json:"level"`
	Time    time.Time              `json:"time"`
	Message string                 `json:"message"`
	Data    interface{}            `json:"data,omitempty"`
	Context map[string]interface{} `json:"context,omitempty"`
}

const (
	LogLevelDebug = LogLevel(iota)
	LogLevelInfo
	LogLevelWarning
	LogLevelError

	contextKeyLogger = contextKey(iota)
)

var (
	// logLevelNames maps predefined log levels to their string
	// representations.
	logLevelNames = map[LogLevel]string{
		LogLevelDebug:   "debug",
		LogLevelInfo:    "info",
		LogLevelWarning: "warning",
		LogLevelError:   "error",
	}

	// DefaultLogger logs to the standard output, filtering out debug
	// messages, and uses the background context.
	DefaultLogger = Logger{
		encoder:     json.NewEncoder(os.Stdout),
		logLevel:    LogLevelInfo,
		contextKeys: nil,
		context:     context.Background(),
	}
)

// Debug is a shorthand for logging with level Debug.
func (l Logger) Debug(str string, data interface{}) error { return l.Log(LogLevelDebug, str, data) }

// Info is a shorthand for logging with level Info.
func (l Logger) Info(str string, data interface{}) error { return l.Log(LogLevelInfo, str, data) }

// Warning is a shorthand for logging with level Warning.
func (l Logger) Warning(str string, data interface{}) error { return l.Log(LogLevelWarning, str, data) }

// Error is a shorthand for logging with level Error.
func (l Logger) Error(str string, data interface{}) error { return l.Log(LogLevelError, str, data) }

// Debug is a shorthand for debug logging on default logger.
func Debug(str string, data interface{}) error { return DefaultLogger.Debug(str, data) }

// Info is a shorthand for info logging on default logger.
func Info(str string, data interface{}) error { return DefaultLogger.Info(str, data) }

// Warning is a shorthand for warning logging on default logger.
func Warning(str string, data interface{}) error { return DefaultLogger.Warning(str, data) }

// Error is a shorthand for error logging on default logger.
func Error(str string, data interface{}) error { return DefaultLogger.Error(str, data) }

// Log is a shorthand for logging on default logger.
func Log(logLevel LogLevel, str string, data interface{}) error {
	return DefaultLogger.Log(logLevel, str, data)
}

// Log logs a message as specified by the Logger. Each message is output as a
// JSON object with `str' in the "message" field, `data' in the "data" field
// (if not nil) and values from the context in "context".
func (l Logger) Log(logLevel LogLevel, str string, data interface{}) error {
	if l.shouldLog(logLevel) {
		return l.doLog(logLevel, str, data)
	} else {
		return nil
	}
}

// shouldLog determines whether the logger should log a given log level.
func (l Logger) shouldLog(logLevel LogLevel) bool {
	return logLevel >= l.logLevel
}

// doLog performs the logging operation with no additional checks.
func (l Logger) doLog(logLevel LogLevel, str string, data interface{}) error {
	m := message{
		Message: str,
		Level:   logLevelNames[logLevel],
		Time:    time.Now(),
		Context: getMessageValuesFromContext(l),
		Data:    data,
	}
	return l.encoder.Encode(m)
}

// getMessageValuesFromContext builds the map of values taken from the context.
// The Logger has a mapping of context keys to JSON keys which is used here.
// For example, if the Logger has a mapping ContextKey(42)->"life", then it
// will look for context value ContextKey(42) and if it exists, output it under
// "life".
func getMessageValuesFromContext(l Logger) map[string]interface{} {
	output := map[string]interface{}{}
	for contextKey, messageKey := range l.contextKeys {
		contextValue := l.context.Value(contextKey)
		if contextValue != nil {
			output[messageKey] = contextValue
		}
	}
	return output
}

// WithWriter returns a new Logger writing to the given Writer.
func (l Logger) WithWriter(w io.Writer) Logger {
	return Logger{
		encoder:     json.NewEncoder(w),
		logLevel:    l.logLevel,
		contextKeys: l.contextKeys,
		context:     l.context,
	}
}

// WithLogLevel returns a new Logger with the given log level.
func (l Logger) WithLogLevel(logLevel LogLevel) Logger {
	return Logger{
		encoder:     l.encoder,
		logLevel:    logLevel,
		contextKeys: l.contextKeys,
		context:     l.context,
	}
}

// WithContext returns a new Logger with the given context.
func (l Logger) WithContext(ctx context.Context) Logger {
	return Logger{
		encoder:     l.encoder,
		logLevel:    l.logLevel,
		contextKeys: l.contextKeys,
		context:     ctx,
	}
}

// WithContextKey returns a new Logger which will extract from the context the
// value at `contextKey' and output it under `messageKey' in the JSON message.
func (l Logger) WithContextKey(contextKey interface{}, messageKey string) Logger {
	newLogger := Logger{
		encoder:     l.encoder,
		logLevel:    l.logLevel,
		contextKeys: nil,
		context:     l.context,
	}
	if l.contextKeys == nil {
		newLogger.contextKeys = map[interface{}]string{
			contextKey: messageKey,
		}
	} else {
		newLogger.contextKeys = shallowCopyMap(l.contextKeys)
		newLogger.contextKeys[contextKey] = messageKey
	}
	return newLogger
}

// ContextWithLogger creates a new context holding a given logger.
// The logger can be retrieved with LoggerFromContextOrDefault.
func ContextWithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, contextKeyLogger, logger)
}

// LoggerFromContextOrDefault gets a Logger from the current context if there
// is one. Otherwise it returns the default logger.
func LoggerFromContextOrDefault(ctx context.Context) Logger {
	value := ctx.Value(contextKeyLogger)
	logger, ok := value.(Logger)
	if ok {
		return logger
	} else {
		return DefaultLogger
	}
}

// shallowCopyMap makes a shallow copy of a map[interface{}]string.
func shallowCopyMap(source map[interface{}]string) map[interface{}]string {
	destination := map[interface{}]string{}
	for k, v := range source {
		destination[k] = v
	}
	return destination
}
