package configuration

import (
	"github.com/cloudogu/ces-importer/api/exporter"
	"log/slog"
	"strings"
)

const (
	nginxDoguName           = "nginx"
	nginxStaticDoguName     = "nginx-static"
	nginxIngressDoguName    = "nginx-ingress"
	nginxGlobalConfigPrefix = "/externals"
)

var (
	nginxStaticExcludeConfigKeys = []string{
		"google_tracking_id",
		"disable_access_log",
		"buffering/*",
		"externals/*",
	}

	nginxIngressExcludeConfigKeys = []string{
		"buffering/*",
		"externals/*",
		"html_content_url",
	}
)

func mergeNginxExternalsConfigIntoGlobalConfig(config *exporter.Configuration) {
	for _, dc := range config.DoguConfigs {
		if dc.Name != nginxDoguName {
			continue
		}

		slog.Debug("Found nginx configuration. Merge 'externals'-config into global config...")
		for _, kv := range dc.NormalConfig {
			if strings.HasPrefix(kv.Key, nginxGlobalConfigPrefix) {
				config.GlobalConfig = append(config.GlobalConfig, kv)
			}
		}

		for _, kv := range dc.LocalConfig {
			if strings.HasPrefix(kv.Key, nginxGlobalConfigPrefix) {
				config.GlobalConfig = append(config.GlobalConfig, kv)
			}
		}
	}
}

func createDoguConfigForNginxStatic(config exporter.DoguConfig) exporter.DoguConfig {
	return createDoguConfigForNginxDogu(config, nginxStaticDoguName, nginxStaticExcludeConfigKeys)
}

func createDoguConfigForNginxIngress(config exporter.DoguConfig) exporter.DoguConfig {
	return createDoguConfigForNginxDogu(config, nginxIngressDoguName, nginxIngressExcludeConfigKeys)
}

func createDoguConfigForNginxDogu(config exporter.DoguConfig, doguName string, excludeWithPrefix []string) exporter.DoguConfig {
	var newNormalConfig []exporter.KeyValue
	var newSensitiveConfig []exporter.KeyValue
	var newLocalConfig []exporter.KeyValue

	for _, configKey := range config.NormalConfig {
		if matchesAnyKeyByPattern(configKey.Key, excludeWithPrefix) {
			continue
		}

		newNormalConfig = append(newNormalConfig, configKey)
	}

	for _, configKey := range config.SensitiveConfig {
		if matchesAnyKeyByPattern(configKey.Key, excludeWithPrefix) {
			continue
		}

		newSensitiveConfig = append(newSensitiveConfig, configKey)
	}

	for _, configKey := range config.LocalConfig {
		if matchesAnyKeyByPattern(configKey.Key, excludeWithPrefix) {
			continue
		}

		newLocalConfig = append(newLocalConfig, configKey)
	}

	return exporter.DoguConfig{
		Name:            doguName,
		NormalConfig:    newNormalConfig,
		SensitiveConfig: newSensitiveConfig,
		LocalConfig:     newLocalConfig,
	}
}
