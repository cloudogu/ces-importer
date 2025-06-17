package configuration

import (
	"context"
	"errors"
	"fmt"
	doguCommons "github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/ces-importer/migration"
	regConfig "github.com/cloudogu/k8s-registry-lib/config"
	"log/slog"
	"os"
	"path"
)

const localConfigPathTemplate = "%s/localConfig/local.yaml"

var getLocalConfigFileForDogu = func(dataBasePath string, dogu string) string {
	doguPath := fmt.Sprintf(localConfigPathTemplate, dogu)

	return path.Join(dataBasePath, doguPath)
}

type cesDoguConfigImporter struct {
	dataBasePath            string
	doguConfigRepo          doguConfigRepo
	sensitiveDoguConfigRepo doguConfigRepo
}

func (dci *cesDoguConfigImporter) importDoguConfigs(ctx context.Context, config []migration.DoguConfig) error {
	slog.Info("Importing dogu config...")

	for _, dc := range config {
		if dc.Name == nginxDoguName {
			nginxStaticConfig := createDoguConfigForNginxStatic(dc)
			if err := dci.importDoguConfig(ctx, nginxStaticConfig); err != nil {
				return fmt.Errorf("failed to import dogu config for dogu '%s': %w", nginxStaticConfig.Name, err)
			}

			nginxIngressConfig := createDoguConfigForNginxIngress(dc)
			if err := dci.importDoguConfig(ctx, nginxIngressConfig); err != nil {
				return fmt.Errorf("failed to import dogu config for dogu '%s': %w", nginxIngressConfig.Name, err)
			}
		} else {
			if err := dci.importDoguConfig(ctx, dc); err != nil {
				return fmt.Errorf("failed to import dogu config for dogu '%s': %w", dc.Name, err)
			}
		}
	}

	slog.Info("...Successfully imported dogu config.")
	return nil
}

func (dci *cesDoguConfigImporter) importDoguConfig(ctx context.Context, dc migration.DoguConfig) error {
	if err := importDoguConfigWithRepo(ctx, dc.Name, dc.NormalConfig, dci.doguConfigRepo); err != nil {
		return fmt.Errorf("failed to import dogu config for dogu '%s': %w", dc.Name, err)
	}

	if err := importDoguConfigWithRepo(ctx, dc.Name, dc.SensitiveConfig, dci.sensitiveDoguConfigRepo); err != nil {
		return fmt.Errorf("failed to import sensitive dogu config for dogu '%s': %w", dc.Name, err)
	}

	if err := importLocalConfig(dci.dataBasePath, dc.Name, dc.LocalConfig); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			slog.Debug("no local config found for dogu", "dogu", dc.Name)
			return nil
		}

		return fmt.Errorf("failed to import local config for dogu '%s': %w", dc.Name, err)
	}

	return nil
}

func importDoguConfigWithRepo(ctx context.Context, dogu string, dc []migration.KeyValue, repo doguConfigRepo) error {
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
		slog.Debug("Setting dogu config", "key", kv.Key)
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

func importLocalConfig(dataBasePath string, dogu string, dc []migration.KeyValue) error {
	if len(dc) == 0 {
		return nil
	}

	localConfigFile := getLocalConfigFileForDogu(dataBasePath, dogu)

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
