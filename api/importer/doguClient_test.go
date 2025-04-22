package importer

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/cloudogu/ces-importer/api/exporter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/ptr"
)

var testCtx, _ = context.WithTimeout(context.Background(), 1*time.Second)
var gibiByte int64 = 1024 * 1024 * 1024
var jenkinsK8sLabels = map[string]string{"app": "ces", "dogu.name": "jenkins"}

const (
	scaleUp   int32 = 1
	scaleDown int32 = 0
)

func TestNewDoguDeploymentClient(t *testing.T) {
	client := NewDoguDeploymentClient(nil, "ecosystem")

	require.NotNil(t, client)
	assert.Implements(t, (*DoguStarter)(nil), client)
	assert.Implements(t, (*DoguStopper)(nil), client)
}

func Test_doguClient_StopDogu(t *testing.T) {
	t.Run("should scale down the given dogu deployment to zero", func(t *testing.T) {
		// given
		clientSetMock := fake.NewClientset()
		inputDeploy := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "jenkins",
				Namespace: "ecosystem",
				Labels:    jenkinsK8sLabels,
			},
			Spec: appsv1.DeploymentSpec{Replicas: ptr.To(scaleUp)},
			Status: appsv1.DeploymentStatus{
				Replicas:      scaleUp,
				ReadyReplicas: 0,
			},
		}

		_, err := clientSetMock.AppsV1().Deployments("ecosystem").Create(testCtx, &inputDeploy, metav1.CreateOptions{})
		require.NoError(t, err)

		dogu := exporter.Dogu{
			Name:    "official/jenkins",
			Version: "2.492.3-4",
			Volume: exporter.DoguVolume{
				SizeInBytes: 2 * gibiByte,
			},
		}

		sut := &doguClient{clientSetMock, "ecosystem"}

		// when
		err = sut.StopDogu(testCtx, dogu)

		// then
		require.NoError(t, err)
	})
	t.Run("should return without error when dogu is already stopped", func(t *testing.T) {
		// given
		clientSetMock := fake.NewClientset()
		inputDeploy := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "jenkins",
				Namespace: "ecosystem",
				Labels:    jenkinsK8sLabels,
			},
			Spec: appsv1.DeploymentSpec{Replicas: ptr.To(scaleDown)},
			Status: appsv1.DeploymentStatus{
				Replicas:      scaleDown,
				ReadyReplicas: 1,
			},
		}

		_, err := clientSetMock.AppsV1().Deployments("ecosystem").Create(testCtx, &inputDeploy, metav1.CreateOptions{})
		require.NoError(t, err)

		dogu := exporter.Dogu{
			Name:    "official/jenkins",
			Version: "2.492.3-4",
			Volume: exporter.DoguVolume{
				SizeInBytes: 2 * gibiByte,
			},
		}

		sut := &doguClient{clientSetMock, "ecosystem"}

		// when
		err = sut.StopDogu(testCtx, dogu)

		// then
		require.NoError(t, err)
	})
	t.Run("should return without error but log warning when dogu was removed in the meantime", func(t *testing.T) {
		// given
		clientSetMock := fake.NewClientset()

		dogu := exporter.Dogu{
			Name:    "official/jenkins",
			Version: "2.492.3-4",
			Volume: exporter.DoguVolume{
				SizeInBytes: 2 * gibiByte,
			},
		}

		sut := &doguClient{clientSetMock, "ecosystem"}

		opts := &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}
		var mockStdout bytes.Buffer
		logHandler := slog.NewTextHandler(&mockStdout, opts)

		logger := slog.New(logHandler)
		slog.SetDefault(logger)

		// when
		err := sut.StopDogu(testCtx, dogu)

		// then
		require.NoError(t, err)
		logOutput := mockStdout.String()
		assert.Contains(t, logOutput, "WARN")
		assert.Contains(t, logOutput, "Cannot scale down dogu deployment because it does not exist")
		assert.Contains(t, logOutput, "jenkins")
	})
	t.Run("should return with error on misconfigured dogu name", func(t *testing.T) {
		// given
		dogu := exporter.Dogu{
			Name: "missingnamespacedoguname",
		}

		sut := &doguClient{nil, "ecosystem"}

		// when
		err := sut.StopDogu(testCtx, dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to stop dogu: dogu name needs to be in the form 'namespace/dogu' but is 'missingnamespacedoguname'")
	})
}

func Test_doguClient_StartDogu(t *testing.T) {
	t.Run("should scale up the given dogu deployment to one", func(t *testing.T) {
		// given
		clientSetMock := fake.NewClientset()
		inputDeploy := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "jenkins",
				Namespace: "ecosystem",
				Labels:    jenkinsK8sLabels,
			},
			Spec: appsv1.DeploymentSpec{Replicas: ptr.To(scaleDown)},
			Status: appsv1.DeploymentStatus{
				Replicas:      scaleDown,
				ReadyReplicas: 1,
			},
		}

		_, err := clientSetMock.AppsV1().Deployments("ecosystem").Create(testCtx, &inputDeploy, metav1.CreateOptions{})
		require.NoError(t, err)

		dogu := exporter.Dogu{
			Name:    "official/jenkins",
			Version: "2.492.3-4",
			Volume: exporter.DoguVolume{
				SizeInBytes: 2 * gibiByte,
			},
		}

		sut := &doguClient{clientSetMock, "ecosystem"}

		// when
		err = sut.StartDogu(testCtx, dogu)

		// then
		require.NoError(t, err)
	})
	t.Run("should return without error when dogu is already started", func(t *testing.T) {
		// given
		clientSetMock := fake.NewClientset()
		inputDeploy := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "jenkins",
				Namespace: "ecosystem",
				Labels:    jenkinsK8sLabels,
			},
			Spec: appsv1.DeploymentSpec{Replicas: ptr.To(scaleUp)},
			Status: appsv1.DeploymentStatus{
				Replicas:      scaleUp,
				ReadyReplicas: 0,
			},
		}

		_, err := clientSetMock.AppsV1().Deployments("ecosystem").Create(testCtx, &inputDeploy, metav1.CreateOptions{})
		require.NoError(t, err)

		dogu := exporter.Dogu{
			Name:    "official/jenkins",
			Version: "2.492.3-4",
			Volume: exporter.DoguVolume{
				SizeInBytes: 2 * gibiByte,
			},
		}

		sut := &doguClient{clientSetMock, "ecosystem"}

		// when
		err = sut.StartDogu(testCtx, dogu)

		// then
		require.NoError(t, err)
	})
	t.Run("should return without error but log warning when dogu was removed in the meantime", func(t *testing.T) {
		// given
		clientSetMock := fake.NewClientset()

		dogu := exporter.Dogu{
			Name:    "official/jenkins",
			Version: "2.492.3-4",
			Volume: exporter.DoguVolume{
				SizeInBytes: 2 * gibiByte,
			},
		}

		sut := &doguClient{clientSetMock, "ecosystem"}

		opts := &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}
		var mockStdout bytes.Buffer
		logHandler := slog.NewTextHandler(&mockStdout, opts)

		logger := slog.New(logHandler)
		slog.SetDefault(logger)

		// when
		err := sut.StartDogu(testCtx, dogu)

		// then
		require.NoError(t, err)
		logOutput := mockStdout.String()
		assert.Contains(t, logOutput, "WARN")
		assert.Contains(t, logOutput, "Cannot scale down dogu deployment because it does not exist")
		assert.Contains(t, logOutput, "jenkins")
	})
	t.Run("should return with error on misconfigured dogu name", func(t *testing.T) {
		// given
		dogu := exporter.Dogu{
			Name: "missingnamespacedoguname",
		}

		sut := &doguClient{nil, "ecosystem"}

		// when
		err := sut.StartDogu(testCtx, dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to start dogu: dogu name needs to be in the form 'namespace/dogu' but is 'missingnamespacedoguname'")
	})
}
