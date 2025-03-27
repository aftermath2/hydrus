package logger

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockWriter is a simple writer that captures written data for testing purposes.
type mockWriter struct {
	buf *bytes.Buffer
}

func (mw *mockWriter) Write(p []byte) (int, error) {
	return mw.buf.Write(p)
}

func TestLoggerLog(t *testing.T) {
	tests := []struct {
		name        string
		loggerLevel Level
		logLevel    Level
		message     string
		expectLog   bool
	}{
		{
			name:        "Same log level",
			loggerLevel: trace,
			logLevel:    trace,
			message:     "Test message",
			expectLog:   true,
		},
		{
			name:        "Log level above logger level",
			loggerLevel: trace,
			logLevel:    info,
			message:     "Test message",
			expectLog:   true,
		},
		{
			name:        "Log level below logger level",
			loggerLevel: warning,
			logLevel:    info,
			message:     "Info message",
			expectLog:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockWriter := &mockWriter{buf: &bytes.Buffer{}}
			logger := &logger{
				out:   mockWriter,
				tag:   "test",
				level: tt.loggerLevel,
			}

			logger.log(tt.logLevel, tt.message)

			if tt.expectLog {
				if mockWriter.buf.Len() == 0 {
					assert.Fail(t, "Expected log message to be written but nothing was written.")
				}

				line, err := mockWriter.buf.ReadString('\n')
				assert.NoError(t, err)

				assert.True(t, strings.Contains(line, levelName(tt.logLevel)))
				assert.True(t, strings.Contains(line, logger.tag))
				assert.True(t, strings.HasSuffix(line, tt.message+"\n"))
			} else {
				if mockWriter.buf.Len() > 0 {
					assert.Fail(t, "Did not expect log message to be written, but it was.")
				}
			}
		})
	}
}
