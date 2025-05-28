package configuration

import (
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"testing"
)

// Helper function to create valid config files
func createValidConfig(t *testing.T, dir string, filename string) {
	var content string
	switch filename {
	case fileLoggingConfig:
		content = "level: INFO"
	case fileAPIConfig:
		content = "host: test-host\napiKey: test-key\nskipTLSVerify: true"
	case fileMigrationConfig:
		content = "regularSchedule: \"0 * * * *\"\nchangeFQDN: true\nmaintenanceModeMessage:\n  title: \"Migration completed.\"\n  text: \"The migration of your instance has been completed.\""
	case fileSSHConfig:
		content = "user: test-user\nprivateKeyPath: /test/path"
	case fileJobContainerConfig:
		content = "image:\n  registry: test-registry\n  repository: test-repo\n  tag: latest"
	case fileJobConfig:
		content = "doguVolumeBasePath: /data\nexclude:\n- dogu: jenkins\n  pattern: JENKINS_PATTERN\n- dogu: redmine\n  pattern: REDMINE_PATTERN"
	case fileSMTPConfig:
		content = "server: test-host\nport: 25\nusername: test-user\npassword: test\nfrom: importer@ces.com\nto: []"
	}

	err := os.WriteFile(path.Join(dir, filename), []byte(content), 0600)
	require.NoError(t, err)
}

// Helper function to write invalid YAML
func writeInvalidYaml(t *testing.T, dir string, filename string) {
	err := os.WriteFile(path.Join(dir, filename), []byte("invalid: yaml: }{"), 0600)
	require.NoError(t, err)
}
