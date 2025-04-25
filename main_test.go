package main

import (
	"bytes"
	"context"
	"encoding/json"
	"k8s.io/client-go/rest"
	"log/slog"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"

	"github.com/cloudogu/ces-importer/configuration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudogu/ces-importer/api/exporter"
)

var testCtx = context.Background()

const testFqdn = "server.fqdn"

var testConfig = configuration.Configuration{
	ExporterHost:              testFqdn,
	ExporterSSHUser:           "root",
	ExporterApiKey:            "my-key",
	ImporterPrivateSSHKeyPath: "/something",
	ImporterNamespace:         "ecosystem",
	LogLevel:                  "INFO",
	MigrationRegularCron:      "0,30 * * * * *",
	MigrationFinalTimestamp:   "2025-something",
}

func Test_isApiExportReady(t *testing.T) {
	t.Run("should be ready", func(t *testing.T) {
		// given
		exportApiClient := newMockExporterApiClient(t)
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/export/mode").Return([]byte(`{"isActive": true}`), nil)

		// when
		ready, err := isApiExportReady(testCtx, "server.fqdn", exportApiClient)

		// then
		require.NoError(t, err)
		assert.True(t, ready)
	})
	t.Run("should not be ready", func(t *testing.T) {
		// given
		exportApiClient := newMockExporterApiClient(t)
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/export/mode").Return([]byte(`{"isActive": false}`), nil)

		// when
		ready, err := isApiExportReady(testCtx, "server.fqdn", exportApiClient)

		// then
		require.NoError(t, err)
		assert.False(t, ready)
	})
	t.Run("should return error for upstream error", func(t *testing.T) {
		// given
		exportApiClient := newMockExporterApiClient(t)
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/export/mode").Return(nil, assert.AnError)

		// when
		_, err := isApiExportReady(testCtx, "server.fqdn", exportApiClient)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to check whether exporter is export ready")
	})
	t.Run("should return error json parsing error", func(t *testing.T) {
		// given
		exportApiClient := newMockExporterApiClient(t)
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/export/mode").Return([]byte(`{banane`), nil)

		// when
		_, err := isApiExportReady(testCtx, "server.fqdn", exportApiClient)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse export mode response")
	})
}

func Test_fetchExporterSystemInfo(t *testing.T) {
	t.Run("should return system infos", func(t *testing.T) {
		// given
		exportApiClient := newMockExporterApiClient(t)
		responseJson := `{"fqdn":"server.fqdn","isMultinode":false,"dogus":[{"name":"official/jenkins","version":"2.492.3-4","volume":{"sizeInBytes":1234}}],"components":[{"name":"k8s/k8s-dogu-operator","version":"3.5.0"}]}`
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/system-info").Return([]byte(responseJson), nil)

		// when
		actual, err := fetchExporterSystemInfo(testCtx, testFqdn, exportApiClient)

		// then
		require.NoError(t, err)
		expectedDogus := []exporter.Dogu{{
			Name:    "official/jenkins",
			Version: "2.492.3-4",
			Volume:  exporter.DoguVolume{SizeInBytes: 1234},
		}}
		expectedComps := []exporter.Component{{
			Name:    "k8s/k8s-dogu-operator",
			Version: "3.5.0",
		}}

		expected := &exporter.SystemInfo{
			FQDN:        testFqdn,
			IsMultinode: false,
			Dogus:       expectedDogus,
			Components:  expectedComps,
		}
		assert.Equal(t, expected, actual)
	})
	t.Run("should return error for upstream error", func(t *testing.T) {
		// given
		exportApiClient := newMockExporterApiClient(t)
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/system-info").Return(nil, assert.AnError)

		// when
		_, err := fetchExporterSystemInfo(testCtx, testFqdn, exportApiClient)

		// then
		require.Error(t, err)
	})
}

func Test_deactivateImporterDogus(t *testing.T) {
	t.Run("should stop dogu", func(t *testing.T) {
		// given
		jenkinsDogu := exporter.Dogu{
			Name:    "official/jenkins",
			Version: "2.492.3-4",
			Volume:  exporter.DoguVolume{SizeInBytes: 1234},
		}

		expectedDogus := []exporter.Dogu{jenkinsDogu}
		expectedComps := []exporter.Component{{
			Name:    "k8s/k8s-dogu-operator",
			Version: "3.5.0",
		}}

		stopper := newMockDoguStopper(t)
		stopper.EXPECT().StopDogu(testCtx, jenkinsDogu).Return(nil)

		inputInfo := &exporter.SystemInfo{
			FQDN:        "server.fqdn",
			IsMultinode: false,
			Dogus:       expectedDogus,
			Components:  expectedComps,
		}

		// when
		err := deactivateImporterDogus(testCtx, inputInfo, stopper)

		// then
		require.NoError(t, err)
	})
	t.Run("should return with error", func(t *testing.T) {
		// given
		jenkinsDogu := exporter.Dogu{
			Name:    "official/jenkins",
			Version: "2.492.3-4",
			Volume:  exporter.DoguVolume{SizeInBytes: 1234},
		}

		expectedDogus := []exporter.Dogu{jenkinsDogu}
		expectedComps := []exporter.Component{{
			Name:    "k8s/k8s-dogu-operator",
			Version: "3.5.0",
		}}

		stopper := newMockDoguStopper(t)
		stopper.EXPECT().StopDogu(testCtx, jenkinsDogu).Return(assert.AnError)

		inputInfo := &exporter.SystemInfo{
			FQDN:        "server.fqdn",
			IsMultinode: false,
			Dogus:       expectedDogus,
			Components:  expectedComps,
		}

		// when
		err := deactivateImporterDogus(testCtx, inputInfo, stopper)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to deactivate dogu official/jenkins in the importer")
	})
}

func Test_activateImporterDogus(t *testing.T) {
	t.Run("should stop dogu", func(t *testing.T) {
		// given
		jenkinsDogu := exporter.Dogu{
			Name:    "official/jenkins",
			Version: "2.492.3-4",
			Volume:  exporter.DoguVolume{SizeInBytes: 1234},
		}

		expectedDogus := []exporter.Dogu{jenkinsDogu}
		expectedComps := []exporter.Component{{
			Name:    "k8s/k8s-dogu-operator",
			Version: "3.5.0",
		}}

		starter := newMockDoguStarter(t)
		starter.EXPECT().StartDogu(testCtx, jenkinsDogu).Return(nil)

		inputInfo := &exporter.SystemInfo{
			FQDN:        "server.fqdn",
			IsMultinode: false,
			Dogus:       expectedDogus,
			Components:  expectedComps,
		}

		// when
		err := activateImporterDogus(testCtx, inputInfo, starter)

		// then
		require.NoError(t, err)
	})
	t.Run("starting dogu should fail with error", func(t *testing.T) {
		// given
		jenkinsDogu := exporter.Dogu{
			Name:    "official/jenkins",
			Version: "2.492.3-4",
			Volume:  exporter.DoguVolume{SizeInBytes: 1234},
		}

		expectedDogus := []exporter.Dogu{jenkinsDogu}
		expectedComps := []exporter.Component{{
			Name:    "k8s/k8s-dogu-operator",
			Version: "3.5.0",
		}}

		starter := newMockDoguStarter(t)
		starter.EXPECT().StartDogu(testCtx, jenkinsDogu).Return(assert.AnError)

		inputInfo := &exporter.SystemInfo{
			FQDN:        "server.fqdn",
			IsMultinode: false,
			Dogus:       expectedDogus,
			Components:  expectedComps,
		}

		// when
		err := activateImporterDogus(testCtx, inputInfo, starter)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to activate dogu official/jenkins in the importer")
	})
}

func Test_createMainLoop_int(t *testing.T) {
	t.Run("should run the function successfully", func(t *testing.T) {
		// given
		exportApiClient := newMockExporterApiClient(t)
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/export/mode").Return([]byte(`{"isActive": true}`), nil)
		responseJson := `{"fqdn":"server.fqdn","isMultinode":false,"dogus":[{"name":"official/jenkins","version":"2.492.3-4","volume":{"sizeInBytes":1234}}],"components":[{"name":"k8s/k8s-dogu-operator","version":"3.5.0"}]}`
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/system-info").Return([]byte(responseJson), nil)

		jenkinsDogu := exporter.Dogu{
			Name:    "official/jenkins",
			Version: "2.492.3-4",
			Volume:  exporter.DoguVolume{SizeInBytes: 1234},
		}

		stopper := newMockDoguStopper(t)
		stopper.EXPECT().StopDogu(testCtx, jenkinsDogu).Return(nil)

		var systemInfo exporter.SystemInfo
		err := json.Unmarshal([]byte(responseJson), &systemInfo)
		require.NoError(t, err)
		doguSyncer := newMockDoguVolumeSyncer(t)
		doguSyncer.EXPECT().SyncDogu(testCtx, "and here", "call your your exporterApiClient for data here", "and here").Return(nil)

		starter := newMockDoguStarter(t)
		starter.EXPECT().StartDogu(testCtx, jenkinsDogu).Return(nil)

		validator := newMockSystemInfoValidator(t)
		validator.EXPECT().ValidateSystemInfo().Return(nil)

		// when
		sut := createMainLoop(testConfig, exportApiClient, starter, stopper, doguSyncer, validator)
		code, err := sut(testCtx)

		// then
		require.NoError(t, err)
		require.Equal(t, 0, code)
	})
	t.Run("should error on the exporter export mode API call but return no error to recover for the next run", func(t *testing.T) {
		// given
		exportApiClient := newMockExporterApiClient(t)
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/export/mode").Return(nil, assert.AnError)

		stopper := newMockDoguStopper(t)
		doguSyncer := newMockDoguVolumeSyncer(t)
		starter := newMockDoguStarter(t)

		opts := &slog.HandlerOptions{Level: slog.LevelDebug}
		var mockStdout bytes.Buffer
		logHandler := slog.NewTextHandler(&mockStdout, opts)

		logger := slog.New(logHandler)
		defer func() {
			orig := slog.Default()
			slog.SetDefault(orig)
		}()
		slog.SetDefault(logger)

		validator := newMockSystemInfoValidator(t)

		// when
		exitCode, err := createMainLoop(testConfig, exportApiClient, starter, stopper, doguSyncer, validator)(testCtx)

		// then
		require.NoError(t, err)
		assert.Equal(t, 0, exitCode)

		logOutput := mockStdout.String()
		assert.Contains(t, logOutput, "level=ERROR msg=\"Error while checking export sync readiness")
		assert.Contains(t, logOutput, "level=INFO msg=\"Waiting for the next run...")
	})
	t.Run("should recover for the the next run when the exporter is not ready to export", func(t *testing.T) {
		// given
		exportApiClient := newMockExporterApiClient(t)
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/export/mode").Return([]byte(`{"isActive": false}`), nil)

		stopper := newMockDoguStopper(t)
		doguSyncer := newMockDoguVolumeSyncer(t)
		starter := newMockDoguStarter(t)

		opts := &slog.HandlerOptions{Level: slog.LevelDebug}
		var mockStdout bytes.Buffer
		logHandler := slog.NewTextHandler(&mockStdout, opts)

		logger := slog.New(logHandler)
		defer func() {
			orig := slog.Default()
			slog.SetDefault(orig)
		}()
		slog.SetDefault(logger)

		validator := newMockSystemInfoValidator(t)

		// when
		exitCode, err := createMainLoop(testConfig, exportApiClient, starter, stopper, doguSyncer, validator)(testCtx)

		// then
		require.NoError(t, err)
		assert.Equal(t, 0, exitCode)

		logOutput := mockStdout.String()
		assert.Contains(t, logOutput, "level=INFO msg=\"Exporter does not seem to be ready. Waiting for the next run...")
	})
	t.Run("should error on the exporter system info API call but return nil to recover for the next run", func(t *testing.T) {
		// given
		exportApiClient := newMockExporterApiClient(t)
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/export/mode").Return([]byte(`{"isActive": true}`), nil)
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/system-info").Return(nil, assert.AnError)

		stopper := newMockDoguStopper(t)
		doguSyncer := newMockDoguVolumeSyncer(t)
		starter := newMockDoguStarter(t)

		opts := &slog.HandlerOptions{Level: slog.LevelDebug}
		var mockStdout bytes.Buffer
		logHandler := slog.NewTextHandler(&mockStdout, opts)

		logger := slog.New(logHandler)
		defer func() {
			orig := slog.Default()
			slog.SetDefault(orig)
		}()
		slog.SetDefault(logger)

		validator := newMockSystemInfoValidator(t)

		// when
		exitCode, err := createMainLoop(testConfig, exportApiClient, starter, stopper, doguSyncer, validator)(testCtx)

		// then
		require.NoError(t, err)
		assert.Equal(t, 0, exitCode)

		logOutput := mockStdout.String()
		assert.Contains(t, logOutput, "level=ERROR msg=\"Failed to fetch the system info from the exporter")
		assert.Contains(t, logOutput, "level=INFO msg=\"Waiting for the next run...")
	})
	t.Run("should fail on stopping a dogu", func(t *testing.T) {
		// given
		exportApiClient := newMockExporterApiClient(t)
		responseJson := `{"fqdn":"server.fqdn","isMultinode":false,"dogus":[{"name":"official/jenkins","version":"2.492.3-4","volume":{"sizeInBytes":1234}}],"components":[{"name":"k8s/k8s-dogu-operator","version":"3.5.0"}]}`
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/system-info").Return([]byte(responseJson), nil)
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/export/mode").Return([]byte(`{"isActive": true}`), nil)

		jenkinsDogu := exporter.Dogu{
			Name:    "official/jenkins",
			Version: "2.492.3-4",
			Volume:  exporter.DoguVolume{SizeInBytes: 1234},
		}

		stopper := newMockDoguStopper(t)
		stopper.EXPECT().StopDogu(testCtx, jenkinsDogu).Return(assert.AnError)

		doguSyncer := newMockDoguVolumeSyncer(t)

		starter := newMockDoguStarter(t)

		validator := newMockSystemInfoValidator(t)
		validator.EXPECT().ValidateSystemInfo().Return(nil)

		// when
		exitCode, err := createMainLoop(testConfig, exportApiClient, starter, stopper, doguSyncer, validator)(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to deactivate dogu official/jenkins in the importer")
		assert.NotEqual(t, 0, exitCode)
	})
	t.Run("should fail on starting a dogu", func(t *testing.T) {
		// given
		exportApiClient := newMockExporterApiClient(t)
		responseJson := `{"fqdn":"server.fqdn","isMultinode":false,"dogus":[{"name":"official/jenkins","version":"2.492.3-4","volume":{"sizeInBytes":1234}}],"components":[{"name":"k8s/k8s-dogu-operator","version":"3.5.0"}]}`
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/system-info").Return([]byte(responseJson), nil)
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/export/mode").Return([]byte(`{"isActive": true}`), nil)

		jenkinsDogu := exporter.Dogu{
			Name:    "official/jenkins",
			Version: "2.492.3-4",
			Volume:  exporter.DoguVolume{SizeInBytes: 1234},
		}

		stopper := newMockDoguStopper(t)
		stopper.EXPECT().StopDogu(testCtx, jenkinsDogu).Return(nil)

		var systemInfo exporter.SystemInfo
		err := json.Unmarshal([]byte(responseJson), &systemInfo)
		require.NoError(t, err)
		doguSyncer := newMockDoguVolumeSyncer(t)
		doguSyncer.EXPECT().SyncDogu(testCtx, "and here", "call your your exporterApiClient for data here", "and here").Return(nil)

		starter := newMockDoguStarter(t)
		starter.EXPECT().StartDogu(testCtx, jenkinsDogu).Return(assert.AnError)

		validator := newMockSystemInfoValidator(t)
		validator.EXPECT().ValidateSystemInfo().Return(nil)

		// when
		exitCode, err := createMainLoop(testConfig, exportApiClient, starter, stopper, doguSyncer, validator)(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to activate dogu official/jenkins in the importer")
		assert.NotEqual(t, 0, exitCode)
	})
	t.Run("should fail on syncing dogu data", func(t *testing.T) {
		// given
		exportApiClient := newMockExporterApiClient(t)
		responseJson := `{"fqdn":"server.fqdn","isMultinode":false,"dogus":[{"name":"official/jenkins","version":"2.492.3-4","volume":{"sizeInBytes":1234}}],"components":[{"name":"k8s/k8s-dogu-operator","version":"3.5.0"}]}`
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/system-info").Return([]byte(responseJson), nil)
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/export/mode").Return([]byte(`{"isActive": true}`), nil)

		jenkinsDogu := exporter.Dogu{
			Name:    "official/jenkins",
			Version: "2.492.3-4",
			Volume:  exporter.DoguVolume{SizeInBytes: 1234},
		}

		stopper := newMockDoguStopper(t)
		stopper.EXPECT().StopDogu(testCtx, jenkinsDogu).Return(nil)

		var systemInfo exporter.SystemInfo
		err := json.Unmarshal([]byte(responseJson), &systemInfo)
		require.NoError(t, err)
		doguSyncer := newMockDoguVolumeSyncer(t)
		doguSyncer.EXPECT().SyncDogu(testCtx, "and here", "call your your exporterApiClient for data here", "and here").Return(assert.AnError)

		starter := newMockDoguStarter(t)

		validator := newMockSystemInfoValidator(t)
		validator.EXPECT().ValidateSystemInfo().Return(nil)

		// when
		exitCode, err := createMainLoop(testConfig, exportApiClient, starter, stopper, doguSyncer, validator)(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		// TODO: test for a better error message once the sync package is fully implemented
		assert.ErrorContains(t, err, "failed to sync source")
		assert.NotEqual(t, 0, exitCode)
	})
}

func Test_logUsedConfig(t *testing.T) {
	// given
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	var mockStdout bytes.Buffer
	logHandler := slog.NewTextHandler(&mockStdout, opts)

	logger := slog.New(logHandler)
	defer func() {
		orig := slog.Default()
		slog.SetDefault(orig)
	}()
	slog.SetDefault(logger)

	// when
	logUsedConfig(testCtx, testConfig)

	// then
	logOutput := mockStdout.String()
	assert.Contains(t, logOutput, "                     ./////,                    ")
	assert.Contains(t, logOutput, "                 ./////==//////,                ")
	assert.Contains(t, logOutput, "                ////.  ___   ////.              ")
	assert.Contains(t, logOutput, "         ,OO,. ////  ,////A,  */// ,OO,.        ")
	assert.Contains(t, logOutput, "    ,/////////////*  */////*  *////////////A    ")
	assert.Contains(t, logOutput, "   ////'        `VA.   '|'   .///'       '///*  ")
	assert.Contains(t, logOutput, "  *///  .*///*,         |         .*//*,   ///* ")
	assert.Contains(t, logOutput, "  (///  (//////)**--_./////_----*//////)   ///) ")
	assert.Contains(t, logOutput, "   V///   '°°°°      (/////)      °°°°'   ////  ")
	assert.Contains(t, logOutput, "    V/////(////////o. '°°°' ./////////(///(/'   ")
	assert.Contains(t, logOutput, "       'V/(/////////////////////////////V'      ")
	assert.Contains(t, logOutput, "ces-importer started using this configuration:")
	assert.Contains(t, logOutput, `config="configuration.Configuration{ExporterHost:\"server.fqdn\", ExporterSSHUser:\"root\", ExporterApiKey:\"my-key\", ImporterPrivateSSHKeyPath:\"/something\", ImporterNamespace:\"ecosystem\", LogLevel:\"INFO\", MigrationRegularCron:\"0,30 * * * * *\", MigrationFinalTimestamp:\"2025-something\"}"`)
}

func Test_configureLogger(t *testing.T) {
	t.Run("should fallback to INFO on config error", func(t *testing.T) {
		// given
		brokenConfig := configuration.Configuration{LogLevel: "banana"}

		// when
		configureLogger(brokenConfig)

		// then
		assert.True(t, slog.Default().Enabled(testCtx, slog.LevelError))
		assert.True(t, slog.Default().Enabled(testCtx, slog.LevelWarn))
		assert.True(t, slog.Default().Enabled(testCtx, slog.LevelInfo))
		assert.False(t, slog.Default().Enabled(testCtx, slog.LevelDebug))
	})
	t.Run("should set loglevel to ERROR", func(t *testing.T) {
		// given
		brokenConfig := configuration.Configuration{LogLevel: "ERROR"}

		// when
		configureLogger(brokenConfig)

		// then
		assert.True(t, slog.Default().Enabled(testCtx, slog.LevelError))
		assert.False(t, slog.Default().Enabled(testCtx, slog.LevelWarn))
		assert.False(t, slog.Default().Enabled(testCtx, slog.LevelInfo))
		assert.False(t, slog.Default().Enabled(testCtx, slog.LevelDebug))
	})
	t.Run("should set loglevel to WARN", func(t *testing.T) {
		// given
		config := configuration.Configuration{LogLevel: "WARN"}

		// when
		configureLogger(config)

		// then
		assert.True(t, slog.Default().Enabled(testCtx, slog.LevelError))
		assert.True(t, slog.Default().Enabled(testCtx, slog.LevelWarn))
		assert.False(t, slog.Default().Enabled(testCtx, slog.LevelInfo))
		assert.False(t, slog.Default().Enabled(testCtx, slog.LevelDebug))
	})
}

// NOTE: Be careful with testing main() because the app may get stuck in the main loop indefinitely.
func Test_main(t *testing.T) {
	t.Run("should panic on missing kube config", func(t *testing.T) {
		// override default controller method to retrieve a kube config
		oldGetConfigDelegate := ctrl.GetConfig
		defer func() {
			ctrl.GetConfig = oldGetConfigDelegate
		}()
		ctrl.GetConfig = func() (*rest.Config, error) {
			return &rest.Config{}, assert.AnError
		}

		// given
		t.Setenv("LOG_LEVEL", "DEBUG")
		t.Setenv("EXPORTER_HOST", "source.net")
		t.Setenv("EXPORTER_SSH_USER", "root")
		t.Setenv("EXPORTER_API_KEY", "example1-1234-5678-102938475")
		t.Setenv("MIGRATION_REGULAR_SCHEDULE", "0 4 * * *")
		t.Setenv("MIGRATION_FINAL_SCHEDULE", "2025-04-03 12:34:56Z")
		t.Setenv("IMPORTER_NAMESPACE", "ecosystem")

		defer func() {
			if r := recover(); r != nil {
				// then
				castedErr := r.(error)
				assert.ErrorContains(t, castedErr, "failed to read kube config")
				assert.ErrorContains(t, castedErr, assert.AnError.Error())
			}
		}()

		// when
		main()

		// we should never arrive here
		t.FailNow()

	})
	t.Run("should panic on missing config variables", func(t *testing.T) {
		// given
		// no env vars go here

		defer func() {
			if r := recover(); r != nil {
				// then
				castedErr := r.(error)
				assert.ErrorContains(t, castedErr, "failed to read config:")
			}
		}()

		// when
		main()

		// we should never arrive here
		t.FailNow()
	})
}
