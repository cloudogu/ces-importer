package main

import (
	"context"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewPreflightExecuter(t *testing.T) {
	t.Run("should create PreflightExecuter without errors", func(t *testing.T) {
		hc := newMockHealthClient(t)
		sig := newMockSystemInfoGetter(t)
		sc := newMockSecretClient(t)

		pe := newPreflightExecuter(hc, sig, sc)

		require.Equal(t, hc, pe.healthClient)
		require.Equal(t, sig, pe.systemInfoGetter)
		require.Equal(t, sc, pe.secretClient)
	})
}

func TestRunPreflightCheck(t *testing.T) {
	t.Run("should return no errors", func(t *testing.T) {
		hc := newMockHealthClient(t)
		hc.EXPECT().GetIsHealthy(mock.Anything).Return(true, nil)

		sig := newMockSystemInfoGetter(t)
		sig.EXPECT().GetImporterSystemInfo(mock.Anything).Return(nil, nil)

		sshct := newMockTestSSHConnection(t)
		sshct.EXPECT().Execute(mock.Anything, mock.Anything, mock.Anything).Return(nil)

		pe := PreflightExecuter{
			healthClient:     hc,
			systemInfoGetter: sig,
			secretClient:     nil,

			testSSHConnection: sshct.Execute,
		}
		cfg := configuration.Coordinator{
			Logging:      configuration.Logging{},
			API:          configuration.API{},
			Migration:    configuration.Migration{},
			SSH:          configuration.SSH{},
			JobConfig:    configuration.JobConfig{},
			JobContainer: configuration.JobContainer{},
			Smtp:         configuration.Smtp{},
			General:      configuration.General{},
		}
		err := pe.runPreflightCheck(context.Background(), cfg)
		require.NoError(t, err)
	})
	t.Run("should error on getting health status", func(t *testing.T) {
		hc := newMockHealthClient(t)
		hc.EXPECT().GetIsHealthy(mock.Anything).Return(false, assert.AnError)

		sig := newMockSystemInfoGetter(t)
		sig.EXPECT().GetImporterSystemInfo(mock.Anything).Return(nil, nil)

		sshct := newMockTestSSHConnection(t)
		sshct.EXPECT().Execute(mock.Anything, mock.Anything, mock.Anything).Return(nil)

		pe := PreflightExecuter{
			healthClient:     hc,
			systemInfoGetter: sig,
			secretClient:     nil,

			testSSHConnection: sshct.Execute,
		}
		cfg := configuration.Coordinator{
			Logging:      configuration.Logging{},
			API:          configuration.API{},
			Migration:    configuration.Migration{},
			SSH:          configuration.SSH{},
			JobConfig:    configuration.JobConfig{},
			JobContainer: configuration.JobContainer{},
			Smtp:         configuration.Smtp{},
			General:      configuration.General{},
		}
		err := pe.runPreflightCheck(context.Background(), cfg)
		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "unable to determine exporter health status")
	})

	t.Run("should error on getting system info", func(t *testing.T) {
		hc := newMockHealthClient(t)
		hc.EXPECT().GetIsHealthy(mock.Anything).Return(true, nil)

		sig := newMockSystemInfoGetter(t)
		sig.EXPECT().GetImporterSystemInfo(mock.Anything).Return(nil, assert.AnError)

		sshct := newMockTestSSHConnection(t)
		sshct.EXPECT().Execute(mock.Anything, mock.Anything, mock.Anything).Return(nil)

		pe := PreflightExecuter{
			healthClient:      hc,
			systemInfoGetter:  sig,
			secretClient:      nil,
			testSSHConnection: sshct.Execute,
		}
		cfg := configuration.Coordinator{
			Logging:      configuration.Logging{},
			API:          configuration.API{},
			Migration:    configuration.Migration{},
			SSH:          configuration.SSH{},
			JobConfig:    configuration.JobConfig{},
			JobContainer: configuration.JobContainer{},
			Smtp:         configuration.Smtp{},
			General:      configuration.General{},
		}
		err := pe.runPreflightCheck(context.Background(), cfg)
		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "unable to retrieve current systems system info")
	})

	t.Run("should error on testing ssh connection", func(t *testing.T) {
		hc := newMockHealthClient(t)
		hc.EXPECT().GetIsHealthy(mock.Anything).Return(true, nil)

		sig := newMockSystemInfoGetter(t)
		sig.EXPECT().GetImporterSystemInfo(mock.Anything).Return(nil, nil)

		sshct := newMockTestSSHConnection(t)
		sshct.EXPECT().Execute(mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)

		pe := PreflightExecuter{
			healthClient:      hc,
			systemInfoGetter:  sig,
			secretClient:      nil,
			testSSHConnection: sshct.Execute,
		}
		cfg := configuration.Coordinator{
			Logging:      configuration.Logging{},
			API:          configuration.API{},
			Migration:    configuration.Migration{},
			SSH:          configuration.SSH{},
			JobConfig:    configuration.JobConfig{},
			JobContainer: configuration.JobContainer{},
			Smtp:         configuration.Smtp{},
			General:      configuration.General{},
		}
		err := pe.runPreflightCheck(context.Background(), cfg)
		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "unable to test ssh connection")
	})
}

//func TestSshConnectionTest(t *testing.T) {
//t.Run("should return no errors", func(t *testing.T) {
//	sc := newMockSecretClient(t)
//	secret := &v1.Secret{
//		TypeMeta:   metav1.TypeMeta{},
//		ObjectMeta: metav1.ObjectMeta{},
//		Immutable:  nil,
//		Data: map[string][]byte{
//			"secret": []byte("123"),
//		},
//		StringData: nil,
//		Type:       "",
//	}
//	sc.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(secret, nil)
//	cfg := configuration.Coordinator{
//		Logging:   configuration.Logging{},
//		API:       configuration.API{},
//		Migration: configuration.Migration{},
//		SSH: configuration.SSH{
//			User:              "",
//			PrivateSSHKeyPath: "",
//			SecretName:        "secret",
//			SecretDataKey:     "secret",
//		},
//		JobConfig:    configuration.JobConfig{},
//		JobContainer: configuration.JobContainer{},
//		Smtp:         configuration.Smtp{},
//		General:      configuration.General{},
//	}
//	ppk := newMockParsePrivateKey(t)
//	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
//	if err != nil {
//		t.Fatalf("Failed to generate RSA key: %v", err)
//	}
//
//	signer, err := ssh.NewSignerFromKey(privateKey)
//	if err != nil {
//		t.Fatalf("Failed to create signer: %v", err)
//	}
//	ppk.EXPECT().Execute(mock.Anything).Return(signer, nil)
//	ds := newMockDial(t)
//	client := newMockSshClient(t)
//	client.EXPECT().NewSession().Return()
//	ds.EXPECT().Execute(mock.Anything, mock.Anything, mock.Anything).Return(client, nil)
//
//	sshConnector{
//		parsePrivateKey:   ppk.Execute,
//		dial:              ds.Execute,
//		testSSHConnection: nil,
//		newSSHSession:     nil,
//	}
//	err = sshConnectionTest(context.Background(), cfg, sc, ppk.Execute, ds.Execute)
//	require.NoError(t, err)
//})
//}
