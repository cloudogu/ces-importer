package configuration

import (
	"context"
	"fmt"
	doguCommons "github.com/cloudogu/ces-commons-lib/dogu"
	regConfig "github.com/cloudogu/k8s-registry-lib/config"
	"log/slog"
	"os"
)

const localConfigPathTemplate = "/data/%s/localConfig/local.yaml"

var getLocalConfigFileForDogu = func(dogu string) string {
	return fmt.Sprintf(localConfigPathTemplate, dogu)
}

func (ci *ConfigImporter) importDoguConfigs(ctx context.Context, config []doguConfig) error {
	slog.Info("Importing dogu config...")

	for _, dc := range config {
		if dc.Name == nginxDoguName {
			nginxStaticConfig := createDoguConfigForNginxStatic(dc)
			if err := ci.importDoguConfig(ctx, nginxStaticConfig); err != nil {
				return fmt.Errorf("failed to import dogu config for dogu '%s': %w", nginxStaticConfig.Name, err)
			}

			nginxIngressConfig := createDoguConfigForNginxIngress(dc)
			if err := ci.importDoguConfig(ctx, nginxIngressConfig); err != nil {
				return fmt.Errorf("failed to import dogu config for dogu '%s': %w", nginxIngressConfig.Name, err)
			}
		} else {
			if err := ci.importDoguConfig(ctx, dc); err != nil {
				return fmt.Errorf("failed to import dogu config for dogu '%s': %w", dc.Name, err)
			}
		}
	}

	slog.Info("...Successfully imported dogu config.")
	return nil
}

func (ci *ConfigImporter) importDoguConfig(ctx context.Context, dc doguConfig) error {
	if err := importDoguConfigWithRepo(ctx, dc.Name, dc.NormalConfig, ci.doguConfigRepo); err != nil {
		return fmt.Errorf("failed to import dogu config for dogu '%s': %w", dc.Name, err)
	}

	if err := importDoguConfigWithRepo(ctx, dc.Name, dc.SensitiveConfig, ci.sensitiveDoguConfigRepo); err != nil {
		return fmt.Errorf("failed to import sensitive dogu config for dogu '%s': %w", dc.Name, err)
	}

	if err := importLocalConfig(dc.Name, dc.LocalConfig); err != nil {
		return fmt.Errorf("failed to import local config for dogu '%s': %w", dc.Name, err)
	}

	return nil
}

func importDoguConfigWithRepo(ctx context.Context, dogu string, dc []keyValue, repo doguConfigRepo) error {
	doguName := doguCommons.SimpleName(dogu)

	err := repo.Delete(ctx, doguName)
	if err != nil {
		return fmt.Errorf("failed to delete original dogu config: %w", err)
	}

	registryDoguConfig, err := repo.Create(ctx, regConfig.CreateDoguConfig(doguName, map[regConfig.Key]regConfig.Value{}))
	if err != nil {
		return fmt.Errorf("failed to create new dogu config: %w", err)
	}

	for _, kv := range dc {
		slog.Debug("Setting dogu config", "key", kv.Key, "value", kv.Value)
		newDoguConfig, err := registryDoguConfig.Set(regConfig.Key(kv.Key), regConfig.Value(kv.Value))
		registryDoguConfig = regConfig.DoguConfig{
			DoguName: doguName,
			Config:   newDoguConfig,
		}

		if err != nil {
			return fmt.Errorf("failed to set key %s: %w\n", kv.Key, err)
		}
	}

	_, err = repo.SaveOrMerge(ctx, registryDoguConfig)
	if err != nil {
		return fmt.Errorf("failed to save dogu config: %w", err)
	}

	return nil
}

func importLocalConfig(dogu string, dc []keyValue) error {
	localConfigFile := getLocalConfigFileForDogu(dogu)

	file, err := os.OpenFile(localConfigFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open local config file at '%s': %w", localConfigFile, err)
	}
	defer func(file *os.File) {
		if cErr := file.Close(); cErr != nil {
			slog.Warn("Failed to close local config file", "err", cErr)
		}
	}(file)

	cfgData := regConfig.Entries{}
	for _, kv := range dc {
		cfgData[regConfig.Key(kv.Key)] = regConfig.Value(kv.Value)
	}

	converter := &regConfig.YamlConverter{}
	if err := converter.Write(file, cfgData); err != nil {
		return fmt.Errorf("failed to write local config file to '%s': %w", localConfigFile, err)
	}

	return nil
}
