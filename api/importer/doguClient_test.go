package importer

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var testCtx, _ = context.WithTimeout(context.Background(), 1*time.Second)
var gibiByte int64 = 1024 * 1024 * 1024
var jenkinsDoguNotFoundErr = errors.NewNotFound(schema.GroupResource{Group: "k8s.cloudogu.com", Resource: "dogu/v2"}, "jenkins")

func TestNewDoguDeploymentClient(t *testing.T) {
	client := NewDoguControl(nil, nil)

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

		podCli := &podInterfaceStub{}

		sut := &DoguControl{doguCli: doguCli, podCli: podCli}

		// when
		err := sut.StopAll(testCtx)

		// then
		require.NoError(t, err)
		assert.Equal(t, []string{"dogu.name=jenkins", "dogu.name=redmine"}, podCli.selectors)
	})

	t.Run("should fail to stop all dogus for error in list", func(t *testing.T) {
		// given
		doguCli := NewMockDoguInterface(t)
		doguCli.EXPECT().List(testCtx, mock.Anything).Return(nil, assert.AnError)

		sut := &DoguControl{doguCli: doguCli}

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

		sut := &DoguControl{doguCli: doguCli}

		// when
		err := sut.StopAll(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to stop dogu: failed to get dogu official/jenkins:")
	})
}

func Test_doguClient_StopAll_waitForPods(t *testing.T) {
	originalTimeout, originalInterval := stopTimeout, stopPollInterval
	stopPollInterval = 1 * time.Millisecond
	t.Cleanup(func() { stopTimeout, stopPollInterval = originalTimeout, originalInterval })

	v2DoguLdap := v2.Dogu{Spec: v2.DoguSpec{Name: "official/ldap", Stopped: true}}

	t.Run("should wait until terminating pods are gone", func(t *testing.T) {
		// given
		stopTimeout = 1 * time.Second
		doguCli := NewMockDoguInterface(t)
		doguCli.EXPECT().List(testCtx, mock.Anything).Return(&v2.DoguList{Items: []v2.Dogu{v2DoguLdap}}, nil)
		doguCli.EXPECT().Get(testCtx, "ldap", mock.Anything).Return(&v2DoguLdap, nil)
		// the first two lists still return the terminating pod
		podCli := &podInterfaceStub{remainingPods: []int{1, 1, 0}}

		sut := &DoguControl{doguCli: doguCli, podCli: podCli}

		// when
		err := sut.StopAll(testCtx)

		// then
		require.NoError(t, err)
		assert.Len(t, podCli.selectors, 3)
	})

	t.Run("should ignore pods in phase Succeeded or Failed", func(t *testing.T) {
		// given
		stopTimeout = 1 * time.Second
		doguCli := NewMockDoguInterface(t)
		doguCli.EXPECT().List(testCtx, mock.Anything).Return(&v2.DoguList{Items: []v2.Dogu{v2DoguLdap}}, nil)
		doguCli.EXPECT().Get(testCtx, "ldap", mock.Anything).Return(&v2DoguLdap, nil)
		podCli := &podInterfaceStub{fixedPods: []corev1.Pod{
			{Status: corev1.PodStatus{Phase: corev1.PodSucceeded}},
			{Status: corev1.PodStatus{Phase: corev1.PodFailed}},
		}}

		sut := &DoguControl{doguCli: doguCli, podCli: podCli}

		// when
		err := sut.StopAll(testCtx)

		// then
		require.NoError(t, err)
		assert.Len(t, podCli.selectors, 1)
	})

	t.Run("should fail if pods do not terminate in time", func(t *testing.T) {
		// given
		stopTimeout = 10 * time.Millisecond
		doguCli := NewMockDoguInterface(t)
		doguCli.EXPECT().List(testCtx, mock.Anything).Return(&v2.DoguList{Items: []v2.Dogu{v2DoguLdap}}, nil)
		doguCli.EXPECT().Get(testCtx, "ldap", mock.Anything).Return(&v2DoguLdap, nil)
		podCli := &podInterfaceStub{alwaysPods: 1}

		sut := &DoguControl{doguCli: doguCli, podCli: podCli}

		// when
		err := sut.StopAll(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.ErrorContains(t, err, "failed to wait for dogu to stop: pods of dogu ldap did not terminate in time")
	})

	t.Run("should fail if pods cannot be listed", func(t *testing.T) {
		// given
		stopTimeout = 1 * time.Second
		doguCli := NewMockDoguInterface(t)
		doguCli.EXPECT().List(testCtx, mock.Anything).Return(&v2.DoguList{Items: []v2.Dogu{v2DoguLdap}}, nil)
		doguCli.EXPECT().Get(testCtx, "ldap", mock.Anything).Return(&v2DoguLdap, nil)
		podCli := &podInterfaceStub{err: assert.AnError}

		sut := &DoguControl{doguCli: doguCli, podCli: podCli}

		// when
		err := sut.StopAll(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to wait for dogu to stop: failed to list pods of dogu ldap:")
	})
}

// podInterfaceStub returns fixedPods if set, otherwise remainingPods[i] running pods on the i-th call
// (0 after the list is exhausted), or alwaysPods running pods on every call if set.
type podInterfaceStub struct {
	fixedPods     []corev1.Pod
	remainingPods []int
	alwaysPods    int
	err           error
	selectors     []string
}

func (s *podInterfaceStub) List(_ context.Context, opts metav1.ListOptions) (*corev1.PodList, error) {
	s.selectors = append(s.selectors, opts.LabelSelector)
	if s.err != nil {
		return nil, s.err
	}
	if s.fixedPods != nil {
		return &corev1.PodList{Items: s.fixedPods}, nil
	}
	count := s.alwaysPods
	if len(s.remainingPods) > 0 {
		count, s.remainingPods = s.remainingPods[0], s.remainingPods[1:]
	}
	pods := make([]corev1.Pod, count)
	for i := range pods {
		pods[i].Status.Phase = corev1.PodRunning
	}
	return &corev1.PodList{Items: pods}, nil
}

func Test_doguClient_StopDogu(t *testing.T) {
	t.Run("should stop the given dogu", func(t *testing.T) {
		// given
		v2DoguJenkins := &v2.Dogu{Spec: v2.DoguSpec{
			Name:    "jenkins",
			Stopped: false,
		}}

		doguCli := NewMockDoguInterface(t)
		doguCli.EXPECT().Get(testCtx, "jenkins", mock.Anything).Return(v2DoguJenkins, nil)
		doguCli.EXPECT().UpdateSpecWithRetry(testCtx, v2DoguJenkins, mock.Anything, mock.Anything).Return(v2DoguJenkins, nil)

		sut := &DoguControl{doguCli: doguCli}

		// when
		err := sut.StopDogu(testCtx, "official/jenkins")

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

		sut := &DoguControl{doguCli: doguCli}

		// when
		err := sut.StopDogu(testCtx, "official/jenkins")

		// then
		require.NoError(t, err)
	})
	t.Run("should return without error but log warning when dogu was removed in the meantime", func(t *testing.T) {
		// given
		doguCli := NewMockDoguInterface(t)
		doguCli.EXPECT().Get(testCtx, "jenkins", mock.Anything).Return(nil, jenkinsDoguNotFoundErr)

		sut := &DoguControl{doguCli: doguCli}

		opts := &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}
		var mockStdout bytes.Buffer
		logHandler := slog.NewTextHandler(&mockStdout, opts)

		logger := slog.New(logHandler)
		slog.SetDefault(logger)

		// when
		err := sut.StopDogu(testCtx, "official/jenkins")

		// then
		require.NoError(t, err)
		logOutput := mockStdout.String()
		assert.Contains(t, logOutput, "WARN")
		assert.Contains(t, logOutput, "Cannot start/stop dogu because it does not exist")
		assert.Contains(t, logOutput, "jenkins")
	})
	t.Run("should return with error on misconfigured dogu name", func(t *testing.T) {
		// given
		sut := &DoguControl{}

		// when
		err := sut.StopDogu(testCtx, "missingnamespacedoguname")

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

		sut := &DoguControl{doguCli: doguCli}

		// when
		err := sut.StartDogu(testCtx, "official/jenkins")

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

		sut := &DoguControl{doguCli: doguCli}

		// when
		err := sut.StartDogu(testCtx, "official/jenkins")

		// then
		require.NoError(t, err)
	})
	t.Run("should return without error but log warning when dogu was removed in the meantime", func(t *testing.T) {
		// given
		doguCli := NewMockDoguInterface(t)
		doguCli.EXPECT().Get(testCtx, "jenkins", mock.Anything).Return(nil, jenkinsDoguNotFoundErr)

		sut := &DoguControl{doguCli: doguCli}

		opts := &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}
		var mockStdout bytes.Buffer
		logHandler := slog.NewTextHandler(&mockStdout, opts)

		logger := slog.New(logHandler)
		slog.SetDefault(logger)

		// when
		err := sut.StartDogu(testCtx, "official/jenkins")

		// then
		require.NoError(t, err)
		logOutput := mockStdout.String()
		assert.Contains(t, logOutput, "WARN")
		assert.Contains(t, logOutput, "Cannot start/stop dogu because it does not exist")
		assert.Contains(t, logOutput, "jenkins")
	})
	t.Run("should return with error on misconfigured dogu name", func(t *testing.T) {
		// given
		sut := &DoguControl{}

		// when
		err := sut.StartDogu(testCtx, "missingnamespacedoguname")

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

		sut := &DoguControl{doguCli: doguCli}

		// when
		err := sut.StartAll(testCtx)

		// then
		require.NoError(t, err)
	})

	t.Run("should fail to start all dogus for error in list", func(t *testing.T) {
		// given
		doguCli := NewMockDoguInterface(t)
		doguCli.EXPECT().List(testCtx, mock.Anything).Return(nil, assert.AnError)

		sut := &DoguControl{doguCli: doguCli}

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

		sut := &DoguControl{doguCli: doguCli}

		// when
		err := sut.StartAll(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to start dogu: failed to get dogu official/jenkins:")
	})
}
