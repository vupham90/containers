package main

import (
	"reflect"
	"testing"
)

func TestSanitizeDockerArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		env      map[string]EnvVar
		expected []string
	}{
		{
			name:     "no environment variables",
			args:     []string{"run", "--rm", "-v", "/path:/workspace", "image:latest"},
			env:      map[string]EnvVar{},
			expected: []string{"run", "--rm", "-v", "/path:/workspace", "image:latest"},
		},
		{
			name: "non-sensitive environment variable",
			args: []string{"run", "--rm", "-e", "LOG_LEVEL=debug", "image:latest"},
			env: map[string]EnvVar{
				"LOG_LEVEL": {Value: "debug", Sensitive: false},
			},
			expected: []string{"run", "--rm", "-e", "LOG_LEVEL=debug", "image:latest"},
		},
		{
			name: "sensitive environment variable",
			args: []string{"run", "--rm", "-e", "API_KEY=secret123", "image:latest"},
			env: map[string]EnvVar{
				"API_KEY": {Value: "secret123", Sensitive: true},
			},
			expected: []string{"run", "--rm", "-e", "API_KEY=***REDACTED***", "image:latest"},
		},
		{
			name: "mixed sensitive and non-sensitive variables",
			args: []string{"run", "--rm", "-e", "LOG_LEVEL=debug", "-e", "API_KEY=secret123", "-e", "PORT=8080", "image:latest"},
			env: map[string]EnvVar{
				"LOG_LEVEL": {Value: "debug", Sensitive: false},
				"API_KEY":   {Value: "secret123", Sensitive: true},
				"PORT":      {Value: "8080", Sensitive: false},
			},
			expected: []string{"run", "--rm", "-e", "LOG_LEVEL=debug", "-e", "API_KEY=***REDACTED***", "-e", "PORT=8080", "image:latest"},
		},
		{
			name: "multiple sensitive variables",
			args: []string{"run", "--rm", "-e", "API_KEY=secret123", "-e", "DB_PASSWORD=pass456", "image:latest"},
			env: map[string]EnvVar{
				"API_KEY":     {Value: "secret123", Sensitive: true},
				"DB_PASSWORD": {Value: "pass456", Sensitive: true},
			},
			expected: []string{"run", "--rm", "-e", "API_KEY=***REDACTED***", "-e", "DB_PASSWORD=***REDACTED***", "image:latest"},
		},
		{
			name: "sensitive variable with special characters in value",
			args: []string{"run", "--rm", "-e", "TOKEN=abc$123!@#", "image:latest"},
			env: map[string]EnvVar{
				"TOKEN": {Value: "abc$123!@#", Sensitive: true},
			},
			expected: []string{"run", "--rm", "-e", "TOKEN=***REDACTED***", "image:latest"},
		},
		{
			name: "empty environment value",
			args: []string{"run", "--rm", "-e", "EMPTY=", "image:latest"},
			env: map[string]EnvVar{
				"EMPTY": {Value: "", Sensitive: true},
			},
			expected: []string{"run", "--rm", "-e", "EMPTY=***REDACTED***", "image:latest"},
		},
		{
			name: "environment variable not in map",
			args: []string{"run", "--rm", "-e", "UNKNOWN=value", "image:latest"},
			env: map[string]EnvVar{
				"API_KEY": {Value: "secret", Sensitive: true},
			},
			expected: []string{"run", "--rm", "-e", "UNKNOWN=value", "image:latest"},
		},
		{
			name: "-e flag at end without value (edge case)",
			args: []string{"run", "--rm", "image:latest", "-e"},
			env: map[string]EnvVar{
				"API_KEY": {Value: "secret", Sensitive: true},
			},
			expected: []string{"run", "--rm", "image:latest", "-e"},
		},
		{
			name: "no -e flags",
			args: []string{"run", "--rm", "-v", "/path:/workspace", "image:latest", "command"},
			env: map[string]EnvVar{
				"API_KEY": {Value: "secret", Sensitive: true},
			},
			expected: []string{"run", "--rm", "-v", "/path:/workspace", "image:latest", "command"},
		},
		{
			name: "environment variable with equals sign in value",
			args: []string{"run", "--rm", "-e", "COMPLEX=key=value", "image:latest"},
			env: map[string]EnvVar{
				"COMPLEX": {Value: "key=value", Sensitive: true},
			},
			expected: []string{"run", "--rm", "-e", "COMPLEX=***REDACTED***", "image:latest"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeDockerArgs(tt.args, tt.env)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("sanitizeDockerArgs() =\n%v\nexpected\n%v", result, tt.expected)
			}
		})
	}
}

func TestSanitizeDockerArgsDoesNotModifyOriginal(t *testing.T) {
	original := []string{"run", "--rm", "-e", "API_KEY=secret123", "image:latest"}
	originalCopy := make([]string, len(original))
	copy(originalCopy, original)

	env := map[string]EnvVar{
		"API_KEY": {Value: "secret123", Sensitive: true},
	}

	sanitizeDockerArgs(original, env)

	if !reflect.DeepEqual(original, originalCopy) {
		t.Errorf("sanitizeDockerArgs modified the original slice.\nOriginal: %v\nAfter: %v", originalCopy, original)
	}
}
