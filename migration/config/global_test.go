package configuration

import (
	"testing"
)

func TestShouldKeepGlobalConfigKey(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		keysToKeep []string
		want       bool
	}{
		{
			name:       "Exact match",
			key:        "my-key",
			keysToKeep: []string{"my-key"},
			want:       true,
		},
		{
			name:       "Pattern match",
			key:        "config1/key",
			keysToKeep: []string{"config1/*"},
			want:       true,
		},
		{
			name:       "No pattern match",
			key:        "config2/other",
			keysToKeep: []string{"config1/*"},
			want:       false,
		},
		{
			name:       "Empty keysToKeep",
			key:        "my-key",
			keysToKeep: []string{},
			want:       false,
		},
		{
			name:       "Invalid pattern",
			key:        "key-to-test",
			keysToKeep: []string{"["},
			want:       false,
		},
		{
			name:       "Multiple patterns, one match",
			key:        "key1",
			keysToKeep: []string{"key2", "key1"},
			want:       true,
		},
		{
			name:       "No match in multiple patterns",
			key:        "key3",
			keysToKeep: []string{"key1", "key2"},
			want:       false,
		},
		{
			name:       "Wildcard pattern",
			key:        "any-key",
			keysToKeep: []string{"*"},
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldKeepGlobalConfigKey(tt.key, tt.keysToKeep)

			if got != tt.want {
				t.Errorf("shouldKeepGlobalConfigKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
