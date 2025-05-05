package configuration

import (
	"testing"
)

func TestMergeNginxExternalsConfigIntoGlobalConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    *configuration
		expected []keyValue
	}{
		{
			name: "No nginx configuration present",
			input: &configuration{
				DoguConfigs: []doguConfig{
					{
						Name:         "other-dogu",
						NormalConfig: []keyValue{{Key: "key1", Value: "value1"}},
					},
				},
			},
			expected: nil,
		},
		{
			name: "Nginx present but no externals keys",
			input: &configuration{
				DoguConfigs: []doguConfig{
					{
						Name:         nginxDoguName,
						NormalConfig: []keyValue{{Key: "otherKey", Value: "value1"}},
					},
				},
			},
			expected: nil,
		},
		{
			name: "Merge externals keys from NormalConfig",
			input: &configuration{
				DoguConfigs: []doguConfig{
					{
						Name: nginxDoguName,
						NormalConfig: []keyValue{
							{Key: "/externals/someKey1", Value: "value1"},
							{Key: "otherKey", Value: "value2"},
						},
					},
				},
			},
			expected: []keyValue{
				{Key: "/externals/someKey1", Value: "value1"},
			},
		},
		{
			name: "Merge externals keys from LocalConfig",
			input: &configuration{
				DoguConfigs: []doguConfig{
					{
						Name: nginxDoguName,
						LocalConfig: []keyValue{
							{Key: "/externals/someKey2", Value: "value2"},
							{Key: "otherKey", Value: "value1"},
						},
					},
				},
			},
			expected: []keyValue{
				{Key: "/externals/someKey2", Value: "value2"},
			},
		},
		{
			name: "Merge keys from both NormalConfig and LocalConfig",
			input: &configuration{
				DoguConfigs: []doguConfig{
					{
						Name: nginxDoguName,
						NormalConfig: []keyValue{
							{Key: "/externals/someKey1", Value: "value1"},
						},
						LocalConfig: []keyValue{
							{Key: "/externals/someKey2", Value: "value2"},
						},
					},
				},
			},
			expected: []keyValue{
				{Key: "/externals/someKey1", Value: "value1"},
				{Key: "/externals/someKey2", Value: "value2"},
			},
		},
		{
			name: "Multiple nginx configs add all externals",
			input: &configuration{
				GlobalConfig: []keyValue{
					{Key: "/something/different", Value: "fooBar"},
				},
				DoguConfigs: []doguConfig{
					{
						Name: nginxDoguName,
						NormalConfig: []keyValue{
							{Key: "/externals/someKey1", Value: "value1"},
						},
						LocalConfig: []keyValue{
							{Key: "/externals/someKey2", Value: "value2"},
						},
					},
				},
			},
			expected: []keyValue{
				{Key: "/something/different", Value: "fooBar"},
				{Key: "/externals/someKey1", Value: "value1"},
				{Key: "/externals/someKey2", Value: "value2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mergeNginxExternalsConfigIntoGlobalConfig(tt.input)
			if len(tt.input.GlobalConfig) != len(tt.expected) {
				t.Fatalf("expected %d global config entries, got %d", len(tt.expected), len(tt.input.GlobalConfig))
			}

			for i, expectedKV := range tt.expected {
				if tt.input.GlobalConfig[i] != expectedKV {
					t.Errorf("unexpected global config at index %d: expected %+v, got %+v", i, expectedKV, tt.input.GlobalConfig[i])
				}
			}
		})
	}
}
