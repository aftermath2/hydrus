package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLevelFromString(t *testing.T) {
	tests := []struct {
		str           string
		expectedLevel Level
	}{
		{
			str:           "disabled",
			expectedLevel: disabled,
		},
		{
			str:           "fatal",
			expectedLevel: fatal,
		},
		{
			str:           "error",
			expectedLevel: errorl,
		},
		{
			str:           "warning",
			expectedLevel: warning,
		},
		{
			str:           "info",
			expectedLevel: info,
		},
		{
			str:           "debug",
			expectedLevel: debug,
		},
		{
			str:           "trace",
			expectedLevel: trace,
		},
	}

	for _, tt := range tests {
		t.Run(tt.str, func(t *testing.T) {
			level, err := LevelFromString(tt.str)
			assert.NoError(t, err)

			assert.Equal(t, tt.expectedLevel, level)
		})
	}
}

func TestLevelFromStringError(t *testing.T) {
	_, err := LevelFromString("test")
	assert.Error(t, err)
}

func TestLevelName(t *testing.T) {
	tests := []struct {
		name     string
		level    Level
		expected string
	}{
		{
			name:     "fatal",
			level:    fatal,
			expected: "FTL",
		},
		{
			name:     "error",
			level:    errorl,
			expected: "ERR",
		},
		{
			name:     "warning",
			level:    warning,
			expected: "WRN",
		},
		{
			name:     "info",
			level:    info,
			expected: "INF",
		},
		{
			name:     "debug",
			level:    debug,
			expected: "DBG",
		},
		{
			name:     "trace",
			level:    trace,
			expected: "TRC",
		},
		{
			name:     "unknown level",
			level:    7,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := levelName(tt.level)
			assert.Equal(t, tt.expected, got)
		})
	}
}
