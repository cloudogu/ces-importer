package configuration

import "testing"

func TestMatchesAnyKeyByPattern(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		patterns []string
		want     bool
	}{
		{
			name:     "Exact match",
			key:      "my-key",
			patterns: []string{"my-key"},
			want:     true,
		},
		{
			name:     "Pattern match",
			key:      "config1/key",
			patterns: []string{"config1/*"},
			want:     true,
		},
		{
			name:     "No pattern match",
			key:      "config2/other",
			patterns: []string{"config1/*"},
			want:     false,
		},
		{
			name:     "Empty patterns",
			key:      "my-key",
			patterns: []string{},
			want:     false,
		},
		{
			name:     "Invalid pattern",
			key:      "key-to-test",
			patterns: []string{"["},
			want:     false,
		},
		{
			name:     "Multiple patterns, one match",
			key:      "key1",
			patterns: []string{"key2", "key1"},
			want:     true,
		},
		{
			name:     "No match in multiple patterns",
			key:      "key3",
			patterns: []string{"key1", "key2"},
			want:     false,
		},
		{
			name:     "Wildcard pattern",
			key:      "any-key",
			patterns: []string{"*"},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesAnyKeyByPattern(tt.key, tt.patterns)

			if got != tt.want {
				t.Errorf("matchesAnyKeyByPattern() = %v, want %v", got, tt.want)
			}
		})
	}
}
