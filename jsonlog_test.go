package jsonlog

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
)

// TestWithLogLevel tests creating new loggers with given log levels.
func TestWithLogLevel(t *testing.T) {
	for l := range logLevelNames {
		logger := DefaultLogger.WithLogLevel(l)
		if logger.logLevel != l {
			t.Errorf("Log level %v should be %v.", logger.logLevel, l)
		}
	}
}

// TestWithContext tests creating new loggers with given contexts.
func TestWithContext(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "key", "value")
	logger := DefaultLogger.WithContext(ctx)
	if logger.context != ctx {
		t.Error("Logger context was not set.")
	}
}

type testSimpleLogsExample struct {
	message  string
	logLevel LogLevel
}

// TestSimpleLogs tests simple logs with no data from the `data' argument or
// from the context.
func TestSimpleLogs(t *testing.T) {
	examples := []testSimpleLogsExample{
		{
			"Some log message.",
			LogLevelDebug,
		},
		{
			"A very long log message. It really shouldn't change anything as the code doesn't really care about that. I don't think I will add any sort of protection for this. Instead I will just consider it is the responsibility of the caller to ensure they won't log huge amounts of data.",
			LogLevelDebug,
		},
		{
			"",
			LogLevelError,
		},
		{
			"Let us add some \x00\x00 null bytes.",
			LogLevelDebug,
		},
		{
			"Template strings and escapes? %v %c %% \\\\\\",
			LogLevelInfo,
		},
	}
	testLogger := DefaultLogger.WithLogLevel(0)
	for _, example := range examples {
		buffer := bytes.NewBuffer(make([]byte, 2048))
		buffer.Reset()
		exampleLogger := testLogger.WithWriter(buffer)
		err := exampleLogger.Log(example.logLevel, example.message, nil)
		if err != nil {
			t.Errorf("Logging errored with '%s'.", err.Error())
		} else {
			output := message{}
			err := json.Unmarshal(buffer.Bytes(), &output)
			if err != nil {
				t.Errorf("Parsing output JSON errored with '%s'.", err.Error())
			} else {
				if output.Message != example.message {
					t.Errorf("Output message '%s' but input was '%s'.", output.Message, example.message)
				}
				if output.Level != logLevelNames[example.logLevel] {
					t.Errorf("Output log level '%s' but input was '%s'.", output.Level, example.logLevel)
				}
			}
		}
	}
}

// TestLogsWithDataMapStringString tests logging with a simple
// map[string]string `data' field.
func TestLogsWithDataMapStringString(t *testing.T) {
	example := map[string]string{
		"type": "map of string to string",
		"foo":  "bar",
	}
	buffer := bytes.NewBuffer(make([]byte, 2048))
	buffer.Reset()
	logger := DefaultLogger.WithWriter(buffer)
	err := logger.Warning("log", example)
	if err != nil {
		t.Errorf("Logging errored with '%s'.", err.Error())
	} else {
		output := struct {
			Data map[string]string `json:"data"`
		}{}
		err := json.Unmarshal(buffer.Bytes(), &output)
		if err != nil {
			t.Errorf("Parsing output JSON errored with '%s'.", err.Error())
		} else {
			for k := range example {
				if example[k] != output.Data[k] {
					t.Errorf("Output '%s' should be '%s'.", output.Data[k], example[k])
				}
			}
		}
	}
}

type testLogsWithContextData struct {
	contextKey interface{}
	outputKey  string
	value      interface{}
	used       bool
}

func TestLogsWithContextData(t *testing.T) {
	examples := []testLogsWithContextData{
		{
			"stringContextKey",
			"string",
			"foobar",
			true,
		},
		{
			424242,
			"integer",
			float64(42),
			true,
		},
		{
			24,
			"unused",
			"nothere",
			false,
		},
	}
	ctx := context.Background()
	for _, e := range examples {
		ctx = context.WithValue(ctx, e.contextKey, e.value)
	}
	buffer := bytes.NewBuffer(make([]byte, 2048))
	buffer.Reset()
	logger := DefaultLogger
	logger = logger.WithContext(ctx)
	logger = logger.WithWriter(buffer)
	for _, e := range examples {
		if e.used {
			logger = logger.WithContextKey(e.contextKey, e.outputKey)
		}
	}
	err := logger.Info("log", nil)
	if err != nil {
		t.Errorf("Logging errored with '%s'.", err.Error())
	} else {
		output := struct {
			Context map[string]interface{} `json:"context"`
		}{}
		err := json.Unmarshal(buffer.Bytes(), &output)
		if err != nil {
			t.Errorf("Parsing output JSON errored with '%s'.", err.Error())
			t.Log(buffer.String())
		} else {
			for _, e := range examples {
				if e.used {
					if output.Context[e.outputKey] != e.value {
						t.Errorf("Context data '%s' is %v but should be %v.", e.outputKey, output.Context[e.outputKey], e.value)
					}
				} else {
					if output.Context[e.outputKey] != nil {
						t.Errorf("Context data '%s' is present and should not be.", e.outputKey)
					}
				}
			}
		}
	}
}
