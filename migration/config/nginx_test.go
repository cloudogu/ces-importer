package configuration

import (
	"github.com/cloudogu/ces-importer/migration"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMergeNginxExternalsConfigIntoGlobalConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    *migration.Configuration
		expected []migration.KeyValue
	}{
		{
			name: "No nginx configuration present",
			input: &migration.Configuration{
				DoguConfigs: []migration.DoguConfig{
					{
						Name:         "other-dogu",
						NormalConfig: []migration.KeyValue{{Key: "key1", Value: "value1"}},
					},
				},
			},
			expected: nil,
		},
		{
			name: "Nginx present but no externals keys",
			input: &migration.Configuration{
				DoguConfigs: []migration.DoguConfig{
					{
						Name:         nginxDoguName,
						NormalConfig: []migration.KeyValue{{Key: "otherKey", Value: "value1"}},
					},
				},
			},
			expected: nil,
		},
		{
			name: "Merge externals keys from NormalConfig",
			input: &migration.Configuration{
				DoguConfigs: []migration.DoguConfig{
					{
						Name: nginxDoguName,
						NormalConfig: []migration.KeyValue{
							{Key: "/externals/someKey1", Value: "value1"},
							{Key: "otherKey", Value: "value2"},
						},
					},
				},
			},
			expected: []migration.KeyValue{
				{Key: "/externals/someKey1", Value: "value1"},
			},
		},
		{
			name: "Merge externals keys from LocalConfig",
			input: &migration.Configuration{
				DoguConfigs: []migration.DoguConfig{
					{
						Name: nginxDoguName,
						LocalConfig: []migration.KeyValue{
							{Key: "/externals/someKey2", Value: "value2"},
							{Key: "otherKey", Value: "value1"},
						},
					},
				},
			},
			expected: []migration.KeyValue{
				{Key: "/externals/someKey2", Value: "value2"},
			},
		},
		{
			name: "Merge keys from both NormalConfig and LocalConfig",
			input: &migration.Configuration{
				DoguConfigs: []migration.DoguConfig{
					{
						Name: nginxDoguName,
						NormalConfig: []migration.KeyValue{
							{Key: "/externals/someKey1", Value: "value1"},
						},
						LocalConfig: []migration.KeyValue{
							{Key: "/externals/someKey2", Value: "value2"},
						},
					},
				},
			},
			expected: []migration.KeyValue{
				{Key: "/externals/someKey1", Value: "value1"},
				{Key: "/externals/someKey2", Value: "value2"},
			},
		},
		{
			name: "Multiple nginx configs add all externals",
			input: &migration.Configuration{
				GlobalConfig: []migration.KeyValue{
					{Key: "/something/different", Value: "fooBar"},
				},
				DoguConfigs: []migration.DoguConfig{
					{
						Name: nginxDoguName,
						NormalConfig: []migration.KeyValue{
							{Key: "/externals/someKey1", Value: "value1"},
						},
						LocalConfig: []migration.KeyValue{
							{Key: "/externals/someKey2", Value: "value2"},
						},
					},
				},
			},
			expected: []migration.KeyValue{
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

func Test_createDoguConfigForNginxIngress(t *testing.T) {
	t.Run("should return a dogu config for the nginx-ingress dogu", func(t *testing.T) {
		cfg := migration.DoguConfig{
			Name: "nginx",
			NormalConfig: []migration.KeyValue{
				{Key: "key1", Value: "value1"},
				{Key: "/buffering/test", Value: "buf_test"},
				{Key: "/externals/test", Value: "ext_test"},
				{Key: "/html_content_url", Value: "content_url"},
			},
			SensitiveConfig: []migration.KeyValue{
				{Key: "key1", Value: "value1"},
				{Key: "/buffering/test", Value: "buf_test"},
				{Key: "/externals/test", Value: "ext_test"},
				{Key: "/html_content_url", Value: "content_url"},
			},
			LocalConfig: []migration.KeyValue{
				{Key: "key1", Value: "value1"},
				{Key: "/buffering/test", Value: "buf_test"},
				{Key: "/externals/test", Value: "ext_test"},
				{Key: "/html_content_url", Value: "content_url"},
			},
		}

		newCfg := createDoguConfigForNginxIngress(cfg)

		assert.Equal(t, "nginx-ingress", newCfg.Name)

		assert.Len(t, newCfg.NormalConfig, 1)
		assert.Equal(t, migration.KeyValue{Key: "key1", Value: "value1"}, newCfg.NormalConfig[0])

		assert.Len(t, newCfg.SensitiveConfig, 1)
		assert.Equal(t, migration.KeyValue{Key: "key1", Value: "value1"}, newCfg.SensitiveConfig[0])

		assert.Len(t, newCfg.LocalConfig, 1)
		assert.Equal(t, migration.KeyValue{Key: "key1", Value: "value1"}, newCfg.LocalConfig[0])
	})
}

func Test_createDoguConfigForNginxStatic(t *testing.T) {
	t.Run("should return a dogu config for the nginx-static dogu", func(t *testing.T) {
		cfg := migration.DoguConfig{
			Name: "nginx",
			NormalConfig: []migration.KeyValue{
				{Key: "key1", Value: "value1"},
				{Key: "/buffering/test", Value: "buf_test"},
				{Key: "/externals/test", Value: "ext_test"},
				{Key: "/google_tracking_id", Value: "tracking_id"},
				{Key: "/disable_access_log", Value: "test"},
			},
			SensitiveConfig: []migration.KeyValue{
				{Key: "key1", Value: "value1"},
				{Key: "/buffering/test", Value: "buf_test"},
				{Key: "/externals/test", Value: "ext_test"},
				{Key: "/google_tracking_id", Value: "tracking_id"},
				{Key: "/disable_access_log", Value: "test"},
			},
			LocalConfig: []migration.KeyValue{
				{Key: "key1", Value: "value1"},
				{Key: "/buffering/test", Value: "buf_test"},
				{Key: "/externals/test", Value: "ext_test"},
				{Key: "/google_tracking_id", Value: "tracking_id"},
				{Key: "/disable_access_log", Value: "test"},
			},
		}

		newCfg := createDoguConfigForNginxStatic(cfg)

		assert.Equal(t, "nginx-static", newCfg.Name)

		assert.Len(t, newCfg.NormalConfig, 1)
		assert.Equal(t, migration.KeyValue{Key: "key1", Value: "value1"}, newCfg.NormalConfig[0])

		assert.Len(t, newCfg.SensitiveConfig, 1)
		assert.Equal(t, migration.KeyValue{Key: "key1", Value: "value1"}, newCfg.SensitiveConfig[0])

		assert.Len(t, newCfg.LocalConfig, 1)
		assert.Equal(t, migration.KeyValue{Key: "key1", Value: "value1"}, newCfg.LocalConfig[0])
	})
}
