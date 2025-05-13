package importer

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/cloudogu/ces-importer/api/exporter"
)

var testCtx, _ = context.WithTimeout(context.Background(), 1*time.Second)
var gibiByte int64 = 1024 * 1024 * 1024
var jenkinsDoguNotFoundErr = errors.NewNotFound(schema.GroupResource{Group: "k8s.cloudogu.com", Resource: "dogu/v2"}, "jenkins")

func TestNewDoguDeploymentClient(t *testing.T) {
	client := NewDoguClient(nil)

	require.NotNil(t, client)
}

func Test_doguClient_StopAll(t *testing.T) {
	t.Run("should stop all dogus", func(t *testing.T) {
		// given
		v2DoguJenkins := v2.Dogu{Spec: v2.DoguSpec{
			Name:    "official/jenkins",
			Stopped: false,
		}}
		v2DoguRedmine := v2.Dogu{Spec: v2.DoguSpec{
			Name:    "official/redmine",
			Stopped: false,
		}}

		doguCli := NewMockDoguInterface(t)
		doguCli.EXPECT().List(testCtx, mock.Anything).Return(&v2.DoguList{Items: []v2.Dogu{v2DoguJenkins, v2DoguRedmine}}, nil)
		doguCli.EXPECT().Get(testCtx, "jenkins", mock.Anything).Return(&v2DoguJenkins, nil)
		doguCli.EXPECT().Get(testCtx, "redmine", mock.Anything).Return(&v2DoguRedmine, nil)
		doguCli.EXPECT().UpdateSpecWithRetry(testCtx, &v2DoguJenkins, mock.Anything, mock.Anything).Return(&v2DoguJenkins, nil)
		doguCli.EXPECT().UpdateSpecWithRetry(testCtx, &v2DoguRedmine, mock.Anything, mock.Anything).Return(&v2DoguRedmine, nil)

		sut := &doguClient{doguCli: doguCli}

		// when
		err := sut.StopAll(testCtx)

		// then
		require.NoError(t, err)
	})

	t.Run("should fail to stop all dogus for error in list", func(t *testing.T) {
		// given
		doguCli := NewMockDoguInterface(t)
		doguCli.EXPECT().List(testCtx, mock.Anything).Return(nil, assert.AnError)

		sut := &doguClient{doguCli: doguCli}

		// when
		err := sut.StopAll(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to list all dogus:")
	})

	t.Run("should fail to stop all dogus for error in startStop", func(t *testing.T) {
		// given
		v2DoguJenkins := v2.Dogu{Spec: v2.DoguSpec{
			Name:    "official/jenkins",
			Stopped: false,
		}}
		v2DoguRedmine := v2.Dogu{Spec: v2.DoguSpec{
			Name:    "official/redmine",
			Stopped: false,
		}}

		doguCli := NewMockDoguInterface(t)
		doguCli.EXPECT().List(testCtx, mock.Anything).Return(&v2.DoguList{Items: []v2.Dogu{v2DoguJenkins, v2DoguRedmine}}, nil)
		doguCli.EXPECT().Get(testCtx, "jenkins", mock.Anything).Return(&v2DoguJenkins, assert.AnError)

		sut := &doguClient{doguCli: doguCli}

		// when
		err := sut.StopAll(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to stop dogu: failed to get dogu official/jenkins:")
	})
}

func Test_doguClient_StopDogu(t *testing.T) {
	t.Run("should stop the given dogu", func(t *testing.T) {
		// given
		v2DoguJenkins := &v2.Dogu{Spec: v2.DoguSpec{
			Name:    "jenkins",
			Stopped: false,
		}}

		dogu := exporter.Dogu{
			Name:    "official/jenkins",
			Version: "2.492.3-4",
			Volume: exporter.DoguVolume{
				SizeInBytes: 2 * gibiByte,
			},
		}
		doguCli := NewMockDoguInterface(t)
		doguCli.EXPECT().Get(testCtx, "jenkins", mock.Anything).Return(v2DoguJenkins, nil)
		doguCli.EXPECT().UpdateSpecWithRetry(testCtx, v2DoguJenkins, mock.Anything, mock.Anything).Return(v2DoguJenkins, nil)

		sut := &doguClient{doguCli: doguCli}

		// when
		err := sut.StopDogu(testCtx, dogu.Name)

		// then
		require.NoError(t, err)
	})
	t.Run("should return without error when dogu is already stopped", func(t *testing.T) {
		// given
		v2DoguJenkins := &v2.Dogu{Spec: v2.DoguSpec{
			Name:    "jenkins",
			Stopped: true,
		}}

		doguCli := NewMockDoguInterface(t)
		doguCli.EXPECT().Get(testCtx, "jenkins", mock.Anything).Return(v2DoguJenkins, nil)

		dogu := exporter.Dogu{
			Name:    "official/jenkins",
			Version: "2.492.3-4",
			Volume: exporter.DoguVolume{
				SizeInBytes: 2 * gibiByte,
			},
		}

		sut := &doguClient{doguCli}

		// when
		err := sut.StopDogu(testCtx, dogu.Name)

		// then
		require.NoError(t, err)
	})
	t.Run("should return without error but log warning when dogu was removed in the meantime", func(t *testing.T) {
		// given
		doguCli := NewMockDoguInterface(t)
		doguCli.EXPECT().Get(testCtx, "jenkins", mock.Anything).Return(nil, jenkinsDoguNotFoundErr)

		dogu := exporter.Dogu{
			Name:    "official/jenkins",
			Version: "2.492.3-4",
			Volume: exporter.DoguVolume{
				SizeInBytes: 2 * gibiByte,
			},
		}

		sut := &doguClient{doguCli}

		opts := &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}
		var mockStdout bytes.Buffer
		logHandler := slog.NewTextHandler(&mockStdout, opts)

		logger := slog.New(logHandler)
		slog.SetDefault(logger)

		// when
		err := sut.StopDogu(testCtx, dogu.Name)

		// then
		require.NoError(t, err)
		logOutput := mockStdout.String()
		assert.Contains(t, logOutput, "WARN")
		assert.Contains(t, logOutput, "Cannot start/stop dogu because it does not exist")
		assert.Contains(t, logOutput, "jenkins")
	})
	t.Run("should return with error on misconfigured dogu name", func(t *testing.T) {
		// given
		dogu := exporter.Dogu{
			Name: "missingnamespacedoguname",
		}

		sut := &doguClient{nil}

		// when
		err := sut.StopDogu(testCtx, dogu.Name)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "dogu name needs to be in the form 'namespace/dogu' but is 'missingnamespacedoguname'")
	})
}

func Test_doguClient_StartDogu(t *testing.T) {
	t.Run("should start the given dogu", func(t *testing.T) {
		// given
		v2DoguJenkins := &v2.Dogu{Spec: v2.DoguSpec{
			Name:    "jenkins",
			Stopped: true,
		}}

		doguCli := NewMockDoguInterface(t)
		doguCli.EXPECT().Get(testCtx, "jenkins", mock.Anything).Return(v2DoguJenkins, nil)
		doguCli.EXPECT().UpdateSpecWithRetry(testCtx, v2DoguJenkins, mock.Anything, mock.Anything).Return(v2DoguJenkins, nil)

		dogu := exporter.Dogu{
			Name:    "official/jenkins",
			Version: "2.492.3-4",
			Volume: exporter.DoguVolume{
				SizeInBytes: 2 * gibiByte,
			},
		}

		sut := &doguClient{doguCli}

		// when
		err := sut.StartDogu(testCtx, dogu.Name)

		// then
		require.NoError(t, err)
	})
	t.Run("should return without error when dogu is already started", func(t *testing.T) {
		// given
		v2DoguJenkins := &v2.Dogu{Spec: v2.DoguSpec{
			Name:    "jenkins",
			Stopped: false,
		}}

		doguCli := NewMockDoguInterface(t)
		doguCli.EXPECT().Get(testCtx, "jenkins", mock.Anything).Return(v2DoguJenkins, nil)

		dogu := exporter.Dogu{
			Name:    "official/jenkins",
			Version: "2.492.3-4",
			Volume: exporter.DoguVolume{
				SizeInBytes: 2 * gibiByte,
			},
		}

		sut := &doguClient{doguCli}

		// when
		err := sut.StartDogu(testCtx, dogu.Name)

		// then
		require.NoError(t, err)
	})
	t.Run("should return without error but log warning when dogu was removed in the meantime", func(t *testing.T) {
		// given

		dogu := exporter.Dogu{
			Name:    "official/jenkins",
			Version: "2.492.3-4",
			Volume: exporter.DoguVolume{
				SizeInBytes: 2 * gibiByte,
			},
		}

		doguCli := NewMockDoguInterface(t)
		doguCli.EXPECT().Get(testCtx, "jenkins", mock.Anything).Return(nil, jenkinsDoguNotFoundErr)

		sut := &doguClient{doguCli}

		opts := &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}
		var mockStdout bytes.Buffer
		logHandler := slog.NewTextHandler(&mockStdout, opts)

		logger := slog.New(logHandler)
		slog.SetDefault(logger)

		// when
		err := sut.StartDogu(testCtx, dogu.Name)

		// then
		require.NoError(t, err)
		logOutput := mockStdout.String()
		assert.Contains(t, logOutput, "WARN")
		assert.Contains(t, logOutput, "Cannot start/stop dogu because it does not exist")
		assert.Contains(t, logOutput, "jenkins")
	})
	t.Run("should return with error on misconfigured dogu name", func(t *testing.T) {
		// given
		dogu := exporter.Dogu{
			Name: "missingnamespacedoguname",
		}

		sut := &doguClient{nil}

		// when
		err := sut.StartDogu(testCtx, dogu.Name)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to start dogu: dogu name needs to be in the form 'namespace/dogu' but is 'missingnamespacedoguname'")
	})
}

func Test_doguClient_StartAll(t *testing.T) {
	t.Run("should start all dogus", func(t *testing.T) {
		// given
		v2DoguJenkins := v2.Dogu{Spec: v2.DoguSpec{
			Name:    "official/jenkins",
			Stopped: true,
		}}
		v2DoguRedmine := v2.Dogu{Spec: v2.DoguSpec{
			Name:    "official/redmine",
			Stopped: true,
		}}

		doguCli := NewMockDoguInterface(t)
		doguCli.EXPECT().List(testCtx, mock.Anything).Return(&v2.DoguList{Items: []v2.Dogu{v2DoguJenkins, v2DoguRedmine}}, nil)
		doguCli.EXPECT().Get(testCtx, "jenkins", mock.Anything).Return(&v2DoguJenkins, nil)
		doguCli.EXPECT().Get(testCtx, "redmine", mock.Anything).Return(&v2DoguRedmine, nil)
		doguCli.EXPECT().UpdateSpecWithRetry(testCtx, &v2DoguJenkins, mock.Anything, mock.Anything).Return(&v2DoguJenkins, nil)
		doguCli.EXPECT().UpdateSpecWithRetry(testCtx, &v2DoguRedmine, mock.Anything, mock.Anything).Return(&v2DoguRedmine, nil)

		sut := &doguClient{doguCli: doguCli}

		// when
		err := sut.StartAll(testCtx)

		// then
		require.NoError(t, err)
	})

	t.Run("should fail to start all dogus for error in list", func(t *testing.T) {
		// given
		doguCli := NewMockDoguInterface(t)
		doguCli.EXPECT().List(testCtx, mock.Anything).Return(nil, assert.AnError)

		sut := &doguClient{doguCli: doguCli}

		// when
		err := sut.StartAll(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to list all dogus:")
	})

	t.Run("should fail to start all dogus for error in startStop", func(t *testing.T) {
		// given
		v2DoguJenkins := v2.Dogu{Spec: v2.DoguSpec{
			Name:    "official/jenkins",
			Stopped: true,
		}}
		v2DoguRedmine := v2.Dogu{Spec: v2.DoguSpec{
			Name:    "official/redmine",
			Stopped: true,
		}}

		doguCli := NewMockDoguInterface(t)
		doguCli.EXPECT().List(testCtx, mock.Anything).Return(&v2.DoguList{Items: []v2.Dogu{v2DoguJenkins, v2DoguRedmine}}, nil)
		doguCli.EXPECT().Get(testCtx, "jenkins", mock.Anything).Return(&v2DoguJenkins, assert.AnError)

		sut := &doguClient{doguCli: doguCli}

		// when
		err := sut.StartAll(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to start dogu: failed to get dogu official/jenkins:")
	})
}
