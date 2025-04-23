package main

import (
	"bytes"
	"context"
	"log/slog"
	"os"
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
		exportApiClient := NewMockexporterApiClient(t)
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/export/mode").Return([]byte(`{"isActive": true}`), nil)

		// when
		ready, err := isApiExportReady(testCtx, "server.fqdn", exportApiClient)

		// then
		require.NoError(t, err)
		assert.True(t, ready)
	})
	t.Run("should not be ready", func(t *testing.T) {
		// given
		exportApiClient := NewMockexporterApiClient(t)
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/export/mode").Return([]byte(`{"isActive": false}`), nil)

		// when
		ready, err := isApiExportReady(testCtx, "server.fqdn", exportApiClient)

		// then
		require.NoError(t, err)
		assert.False(t, ready)
	})
	t.Run("should return error for upstream error", func(t *testing.T) {
		// given
		exportApiClient := NewMockexporterApiClient(t)
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/export/mode").Return(nil, assert.AnError)

		// when
		_, err := isApiExportReady(testCtx, "server.fqdn", exportApiClient)

		// then
		require.Error(t, err)
	})
}

func Test_fetchExporterSystemInfo(t *testing.T) {
	t.Run("should return system infos", func(t *testing.T) {
		// given
		exportApiClient := NewMockexporterApiClient(t)
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
		exportApiClient := NewMockexporterApiClient(t)
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

		stopper := NewMockdoguStopper(t)
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

		stopper := NewMockdoguStopper(t)
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

		starter := NewMockdoguStarter(t)
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

		starter := NewMockdoguStarter(t)
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
		exportApiClient := NewMockexporterApiClient(t)
		responseJson := `{"fqdn":"server.fqdn","isMultinode":false,"dogus":[{"name":"official/jenkins","version":"2.492.3-4","volume":{"sizeInBytes":1234}}],"components":[{"name":"k8s/k8s-dogu-operator","version":"3.5.0"}]}`
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/system-info").Return([]byte(responseJson), nil)
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/export/mode").Return([]byte(`{"isActive": true}`), nil)

		jenkinsDogu := exporter.Dogu{
			Name:    "official/jenkins",
			Version: "2.492.3-4",
			Volume:  exporter.DoguVolume{SizeInBytes: 1234},
		}

		stopper := NewMockdoguStopper(t)
		stopper.EXPECT().StopDogu(testCtx, jenkinsDogu).Return(nil)

		starter := NewMockdoguStarter(t)
		starter.EXPECT().StartDogu(testCtx, jenkinsDogu).Return(nil)

		// when
		err := createMainLoop(testConfig, exportApiClient, starter, stopper)(testCtx)

		// then
		require.NoError(t, err)
	})
	t.Run("should error on the exporter export mode API call but return no error to recover for the next run", func(t *testing.T) {
		// given
		exportApiClient := NewMockexporterApiClient(t)
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/export/mode").Return(nil, assert.AnError)

		stopper := NewMockdoguStopper(t)
		starter := NewMockdoguStarter(t)

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
		err := createMainLoop(testConfig, exportApiClient, starter, stopper)(testCtx)

		// then
		require.NoError(t, err)

		logOutput := mockStdout.String()
		assert.Contains(t, logOutput, "level=ERROR msg=\"Error while checking export sync readiness")
		assert.Contains(t, logOutput, "level=INFO msg=\"Waiting for the next run...")
	})
	t.Run("should recover for the the next run when the exporter is not ready to export", func(t *testing.T) {
		// given
		exportApiClient := NewMockexporterApiClient(t)
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/export/mode").Return([]byte(`{"isActive": false}`), nil)

		stopper := NewMockdoguStopper(t)
		starter := NewMockdoguStarter(t)

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
		err := createMainLoop(testConfig, exportApiClient, starter, stopper)(testCtx)

		// then
		require.NoError(t, err)

		logOutput := mockStdout.String()
		assert.Contains(t, logOutput, "level=INFO msg=\"Exporter does not seem to be ready. Waiting for the next run...")
	})
	t.Run("should error on the exporter system info API call but return nil to recover for the next run", func(t *testing.T) {
		// given
		exportApiClient := NewMockexporterApiClient(t)
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/export/mode").Return([]byte(`{"isActive": true}`), nil)
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/system-info").Return(nil, assert.AnError)

		stopper := NewMockdoguStopper(t)
		starter := NewMockdoguStarter(t)

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
		err := createMainLoop(testConfig, exportApiClient, starter, stopper)(testCtx)

		// then
		require.NoError(t, err)

		logOutput := mockStdout.String()
		assert.Contains(t, logOutput, "level=ERROR msg=\"Failed to fetch the system info from the exporter")
		assert.Contains(t, logOutput, "level=INFO msg=\"Waiting for the next run...")
	})
	t.Run("should fail on stopping a dogu", func(t *testing.T) {
		// given
		exportApiClient := NewMockexporterApiClient(t)
		responseJson := `{"fqdn":"server.fqdn","isMultinode":false,"dogus":[{"name":"official/jenkins","version":"2.492.3-4","volume":{"sizeInBytes":1234}}],"components":[{"name":"k8s/k8s-dogu-operator","version":"3.5.0"}]}`
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/system-info").Return([]byte(responseJson), nil)
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/export/mode").Return([]byte(`{"isActive": true}`), nil)

		jenkinsDogu := exporter.Dogu{
			Name:    "official/jenkins",
			Version: "2.492.3-4",
			Volume:  exporter.DoguVolume{SizeInBytes: 1234},
		}

		stopper := NewMockdoguStopper(t)
		stopper.EXPECT().StopDogu(testCtx, jenkinsDogu).Return(assert.AnError)

		starter := NewMockdoguStarter(t)

		// when
		err := createMainLoop(testConfig, exportApiClient, starter, stopper)(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to deactivate dogu official/jenkins in the importer")
	})
	t.Run("should fail on starting a dogu", func(t *testing.T) {
		// given
		exportApiClient := NewMockexporterApiClient(t)
		responseJson := `{"fqdn":"server.fqdn","isMultinode":false,"dogus":[{"name":"official/jenkins","version":"2.492.3-4","volume":{"sizeInBytes":1234}}],"components":[{"name":"k8s/k8s-dogu-operator","version":"3.5.0"}]}`
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/system-info").Return([]byte(responseJson), nil)
		exportApiClient.EXPECT().DoGetRequest(testCtx, "https://server.fqdn/export/mode").Return([]byte(`{"isActive": true}`), nil)

		jenkinsDogu := exporter.Dogu{
			Name:    "official/jenkins",
			Version: "2.492.3-4",
			Volume:  exporter.DoguVolume{SizeInBytes: 1234},
		}

		stopper := NewMockdoguStopper(t)
		stopper.EXPECT().StopDogu(testCtx, jenkinsDogu).Return(nil)

		starter := NewMockdoguStarter(t)
		starter.EXPECT().StartDogu(testCtx, jenkinsDogu).Return(assert.AnError)

		// when
		err := createMainLoop(testConfig, exportApiClient, starter, stopper)(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to activate dogu official/jenkins in the importer")
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
				assert.ErrorContains(t, castedErr, "failed to read kube config: invalid configuration")
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

func writeKubeConfig(t *testing.T) *os.File {
	t.Helper()

	kubeconfig, err := os.CreateTemp(os.TempDir(), "test.kubeconfig.delete.me")
	require.NoError(t, err)

	fakeKubeConfigData := `
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJkekNDQVIyZ0F3SUJBZ0lCQURBS0JnZ3Foa2pPUFFRREFqQWpNU0V3SHdZRFZRUUREQmhyTTNNdGMyVnkKZG1WeUxXTmhRREUyTmpBM01qWTJNREF3SGhjTk1qSXdPREUzTURnMU5qUXdXaGNOTXpJd09ERTBNRGcxTmpRdwpXakFqTVNFd0h3WURWUVFEREJock0zTXRjMlZ5ZG1WeUxXTmhRREUyTmpBM01qWTJNREF3V1RBVEJnY3Foa2pPClBRSUJCZ2dxaGtqT1BRTUJCd05DQUFTRnNnekdMYTkxelZVSmgybmRxdUFhbDYzUTRQTGQzMFc1dGY1S0JoQUcKYnhVY2xrOVhPeTlmVDZTMEd5YUNJbk5HU00relV3OUNDTEhRV0JQWVpMZTFvMEl3UURBT0JnTlZIUThCQWY4RQpCQU1DQXFRd0R3WURWUjBUQVFIL0JBVXdBd0VCL3pBZEJnTlZIUTRFRmdRVTQ3T1J0SDdaRDdwYkphc084dW9UCnBtUjl1TDR3Q2dZSUtvWkl6ajBFQXdJRFNBQXdSUUlnREtNRmk1L0ViRW94dmZmVDZhNTNnd1diM0Z1NTNIakcKOUlzT0d2MjhmRzhDSVFEVldmZGpBcy92Vy9XdnNoZi9QUENDNkowRnlEK1lwdFl2WFl5b2k1S2ZlUT09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
    server: https://127.0.0.1:6443
  name: default
contexts:
- context:
    cluster: default
    user: default
  name: default
current-context: default
kind: Config
preferences: {}
users:
- name: default
  user:
    client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJrRENDQVRlZ0F3SUJBZ0lJYlEvTEFBSURDTTR3Q2dZSUtvWkl6ajBFQXdJd0l6RWhNQjhHQTFVRUF3d1kKYXpOekxXTnNhV1Z1ZEMxallVQXhOall3TnpJMk5qQXdNQjRYRFRJeU1EZ3hOekE0TlRZME1Gb1hEVEl6TURneApOekE0TlRZME1Gb3dNREVYTUJVR0ExVUVDaE1PYzNsemRHVnRPbTFoYzNSbGNuTXhGVEFUQmdOVkJBTVRESE41CmMzUmxiVHBoWkcxcGJqQlpNQk1HQnlxR1NNNDlBZ0VHQ0NxR1NNNDlBd0VIQTBJQUJLKzE3VHlFVEl5YUludEsKQWM0TFl2L1FLVWhVbVpXeWFRWGhUU1FrcTZxWUlEZlFrMjV6OEE3YXJCd2YyQ2tsZDZ0NzJFc2J0WXEvTzlPNgp0VU5Nb01DalNEQkdNQTRHQTFVZER3RUIvd1FFQXdJRm9EQVRCZ05WSFNVRUREQUtCZ2dyQmdFRkJRY0RBakFmCkJnTlZIU01FR0RBV2dCU2JmMHdXL0NsMTNVRFJHOU93anE3WWFpUXZqREFLQmdncWhrak9QUVFEQWdOSEFEQkUKQWlBN0cvUVYvc0ZpYUhmbUlRZitBVVQ2SS9nRWlkdlgzbVd0ZGFYb1FmSjUyZ0lnUVNCVFdrb2RZZHR4b3F0bgpnMkszYkxjUEdXbnBGTFdUREdVQU1WZFpDR3M9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0KLS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJlRENDQVIyZ0F3SUJBZ0lCQURBS0JnZ3Foa2pPUFFRREFqQWpNU0V3SHdZRFZRUUREQmhyTTNNdFkyeHAKWlc1MExXTmhRREUyTmpBM01qWTJNREF3SGhjTk1qSXdPREUzTURnMU5qUXdXaGNOTXpJd09ERTBNRGcxTmpRdwpXakFqTVNFd0h3WURWUVFEREJock0zTXRZMnhwWlc1MExXTmhRREUyTmpBM01qWTJNREF3V1RBVEJnY3Foa2pPClBRSUJCZ2dxaGtqT1BRTUJCd05DQUFUb01XT2R6Qnd1a3pidWJ3aVFLWVljYW1rL2lpQnYzSmRIUlpxWUxhWWEKOG9sUmhVNkRYMVRDMiswVHBNOGVZK0dlVnlFZUo2b3lwYXE1SUZEU2RtbjdvMEl3UURBT0JnTlZIUThCQWY4RQpCQU1DQXFRd0R3WURWUjBUQVFIL0JBVXdBd0VCL3pBZEJnTlZIUTRFRmdRVW0zOU1GdndwZGQxQTBSdlRzSTZ1CjJHb2tMNHd3Q2dZSUtvWkl6ajBFQXdJRFNRQXdSZ0loQUp3UlRZdVdZZGlqNUpKVlRkSi96cjh0QXd3NkIwR3IKK1NGUmk1bDIvLzEvQWlFQXlvMVNUUEtyai9FcXQ4MWtyMEdpdnhwbjZMQnExaG9rZkpLbWY2OHQ4MFU9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
    client-key-data: LS0tLS1CRUdJTiBFQyBQUklWQVRFIEtFWS0tLS0tCk1IY0NBUUVFSU1OcTVvWUU4c09BSEwxaWIwYTBVak1rTUl1SXZIem14L1NUTGc5ZTA0SGFvQW9HQ0NxR1NNNDkKQXdFSG9VUURRZ0FFcjdYdFBJUk1qSm9pZTBvQnpndGkvOUFwU0ZTWmxiSnBCZUZOSkNTcnFwZ2dOOUNUYm5QdwpEdHFzSEIvWUtTVjNxM3ZZU3h1MWlyODcwN3ExUTB5Z3dBPT0KLS0tLS1FTkQgRUMgUFJJVkFURSBLRVktLS0tLQo=
`
	err = os.WriteFile(kubeconfig.Name(), []byte(fakeKubeConfigData), 0755)
	require.NoError(t, err)

	return kubeconfig
}
