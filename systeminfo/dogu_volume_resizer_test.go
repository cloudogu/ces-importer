package systeminfo

import (
	"context"
	"errors"
	"fmt"
	"testing"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/ces-importer/migration"
	"github.com/cloudogu/cesapp-lib/core"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_defaultDoguVolumeResizer_resize(t *testing.T) {
	testCtx := context.Background()
	t.Run("should fail to resize pvc if doguName can not be parsed", func(t *testing.T) {
		resizer := &DoguVolumeResizer{}

		err := resizer.resize(testCtx, "simpleDoguName", 1)

		require.Error(t, err)
		require.ErrorContains(t, err, "dogu simpleDoguName name is not a qualified dogu name: dogu name needs to be in the form 'namespace/dogu' but is 'simpleDoguName'")
	})

	t.Run("should fail to resize pvc if dogu can not be found", func(t *testing.T) {
		mDoguClient := newMockDoguClient(t)
		mDoguClient.EXPECT().Get(testCtx, "ldap", metav1.GetOptions{}).Return(nil, assert.AnError)

		resizer := &DoguVolumeResizer{
			doguClient: mDoguClient,
		}

		err := resizer.resize(testCtx, "official/ldap", 1)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		require.ErrorContains(t, err, "dogu \"ldap\" could not be found:")
	})

	t.Run("should fail to resize pvc if dogu can not be updated", func(t *testing.T) {
		mDoguClient := newMockDoguClient(t)
		dogu := &doguv2.Dogu{
			Spec: doguv2.DoguSpec{
				Resources: doguv2.DoguResources{
					MinDataVolumeSize: resource.MustParse("1Gi"),
				},
			},
		}
		mDoguClient.EXPECT().Get(testCtx, "ldap", metav1.GetOptions{}).Return(dogu, nil)
		mDoguClient.EXPECT().Update(testCtx, dogu, metav1.UpdateOptions{}).Return(nil, assert.AnError)

		resizer := &DoguVolumeResizer{
			doguClient: mDoguClient,
		}

		err := resizer.resize(testCtx, "official/ldap", 2*1024*1024*1024)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		require.ErrorContains(t, err, "dogu \"ldap\" does not have enough volume capacity and the volume could not be resized:")
	})

	t.Run("should fail to resize pvc if wait for pvc has error", func(t *testing.T) {
		mDoguClient := newMockDoguClient(t)
		dogu := &doguv2.Dogu{
			Spec: doguv2.DoguSpec{
				Resources: doguv2.DoguResources{
					MinDataVolumeSize: resource.MustParse("1Gi"),
				},
			},
		}
		mDoguClient.EXPECT().Get(testCtx, "ldap", metav1.GetOptions{}).Return(dogu, nil)
		mDoguClient.EXPECT().Update(testCtx, dogu, metav1.UpdateOptions{}).RunAndReturn(func(ctx context.Context, dogu *doguv2.Dogu, options metav1.UpdateOptions) (*doguv2.Dogu, error) {
			assert.Equal(t, "6Gi", dogu.Spec.Resources.MinDataVolumeSize.String())

			return dogu, nil
		})

		mPvcClient := newMockPvcClient(t)
		i := 0
		mPvcClient.EXPECT().Get(testCtx, "ldap", metav1.GetOptions{}).RunAndReturn(func(ctx context.Context, doguName string, options metav1.GetOptions) (*corev1.PersistentVolumeClaim, error) {
			// increase with each iteration
			i++
			assert.Less(t, i, 4)

			pvc := &corev1.PersistentVolumeClaim{
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("6Gi"),
						},
					},
				},
				Status: corev1.PersistentVolumeClaimStatus{
					Capacity: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse(fmt.Sprintf("%dGi", i)),
					},
				},
			}

			return pvc, nil
		})
		resizer := &DoguVolumeResizer{
			doguClient: mDoguClient,
			pvcClient:  mPvcClient,
		}

		waitSecondsBetweenRetries = 0
		defaultMaxRetries := maxRetries
		maxRetries = 3
		defer func() {
			waitSecondsBetweenRetries = defaultWaitSecondsBetweenRetries
			maxRetries = defaultMaxRetries
		}()

		err := resizer.resize(testCtx, "official/ldap", 6*1024*1024*1024)

		require.Error(t, err)
		require.ErrorContains(t, err, "error waiting for pvc of dogu ldap to be resized: maximum amount of retries reached for the resize of dogu \"ldap\" volume")
	})

	t.Run("should successfully resize pvc for dogu", func(t *testing.T) {
		mDoguClient := newMockDoguClient(t)
		dogu := &doguv2.Dogu{
			Spec: doguv2.DoguSpec{
				Resources: doguv2.DoguResources{
					MinDataVolumeSize: resource.MustParse("1Gi"),
				},
			},
		}
		mDoguClient.EXPECT().Get(testCtx, "ldap", metav1.GetOptions{}).Return(dogu, nil)
		mDoguClient.EXPECT().Update(testCtx, dogu, metav1.UpdateOptions{}).RunAndReturn(func(ctx context.Context, dogu *doguv2.Dogu, options metav1.UpdateOptions) (*doguv2.Dogu, error) {
			assert.Equal(t, "2Gi", dogu.Spec.Resources.MinDataVolumeSize.String())

			return dogu, nil
		})

		mPvcClient := newMockPvcClient(t)
		mPvcClient.EXPECT().Get(testCtx, "ldap", metav1.GetOptions{}).Return(&corev1.PersistentVolumeClaim{
			Spec: corev1.PersistentVolumeClaimSpec{
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("2Gi"),
					},
				},
			},
			Status: corev1.PersistentVolumeClaimStatus{
				Capacity: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("2Gi"),
				},
			},
		}, nil)

		resizer := &DoguVolumeResizer{
			doguClient: mDoguClient,
			pvcClient:  mPvcClient,
		}

		waitSecondsBetweenRetries = 0
		defer func() {
			waitSecondsBetweenRetries = defaultWaitSecondsBetweenRetries
		}()

		err := resizer.resize(testCtx, "official/ldap", 2*1024*1024*1024)

		require.NoError(t, err)
	})
}

func Test_defaultDoguVolumeResizer_waitForPVCResize(t *testing.T) {
	testCtx := context.Background()
	t.Run("should not fail to wait for pvc resize if pvc can not be found", func(t *testing.T) {
		requestedDataVolumeSize := resource.MustParse("2Gi")
		actualDataVolumeSize := resource.MustParse("2Gi")

		counter := 0
		mPvcClient := newMockPvcClient(t)
		mPvcClient.EXPECT().Get(testCtx, "ldap", metav1.GetOptions{}).RunAndReturn(func(ctx context.Context, doguName string, options metav1.GetOptions) (*corev1.PersistentVolumeClaim, error) {
			counter++

			if counter <= 1 {
				return nil, assert.AnError
			}

			pvc := &corev1.PersistentVolumeClaim{
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: requestedDataVolumeSize,
						},
					},
				},
				Status: corev1.PersistentVolumeClaimStatus{
					Capacity: corev1.ResourceList{
						corev1.ResourceStorage: actualDataVolumeSize,
					},
				},
			}

			return pvc, nil
		})

		resizer := &DoguVolumeResizer{
			pvcClient: mPvcClient,
		}

		waitSecondsBetweenRetries = 0
		defer func() {
			waitSecondsBetweenRetries = defaultWaitSecondsBetweenRetries
		}()

		err := resizer.waitForPVCResize(testCtx, "ldap", &requestedDataVolumeSize)

		require.NoError(t, err)
	})

	t.Run("should fail to wait for pvc resize if max retries is reached", func(t *testing.T) {
		requestedDataVolumeSize := resource.MustParse("6Gi")

		mPvcClient := newMockPvcClient(t)

		i := 0
		mPvcClient.EXPECT().Get(testCtx, "ldap", metav1.GetOptions{}).RunAndReturn(func(ctx context.Context, doguName string, options metav1.GetOptions) (*corev1.PersistentVolumeClaim, error) {
			// increase with each iteration
			i++
			assert.Less(t, i, 4)

			pvc := &corev1.PersistentVolumeClaim{
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: requestedDataVolumeSize,
						},
					},
				},
				Status: corev1.PersistentVolumeClaimStatus{
					Capacity: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse(fmt.Sprintf("%dGi", i)),
					},
				},
			}

			return pvc, nil
		})

		resizer := &DoguVolumeResizer{
			pvcClient: mPvcClient,
		}

		waitSecondsBetweenRetries = 0
		defaultMaxRetries := maxRetries
		maxRetries = 3
		defer func() {
			waitSecondsBetweenRetries = defaultWaitSecondsBetweenRetries
			maxRetries = defaultMaxRetries
		}()

		err := resizer.waitForPVCResize(testCtx, "ldap", &requestedDataVolumeSize)

		require.Error(t, err)
		assert.ErrorContains(t, err, "maximum amount of retries reached for the resize of dogu \"ldap\" volume")
	})

	t.Run("should succeed to wait for pvc resize with retries", func(t *testing.T) {
		requestedDataVolumeSize := resource.MustParse("3Gi")

		mPvcClient := newMockPvcClient(t)

		i := 0
		mPvcClient.EXPECT().Get(testCtx, "ldap", metav1.GetOptions{}).RunAndReturn(func(ctx context.Context, doguName string, options metav1.GetOptions) (*corev1.PersistentVolumeClaim, error) {
			// increase with each iteration
			i++
			assert.Less(t, i, 4)

			pvc := &corev1.PersistentVolumeClaim{
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: requestedDataVolumeSize,
						},
					},
				},
				Status: corev1.PersistentVolumeClaimStatus{
					Capacity: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse(fmt.Sprintf("%dGi", i)),
					},
				},
			}

			return pvc, nil
		})

		resizer := &DoguVolumeResizer{
			pvcClient: mPvcClient,
		}

		waitSecondsBetweenRetries = 0
		defer func() {
			waitSecondsBetweenRetries = defaultWaitSecondsBetweenRetries
		}()

		err := resizer.waitForPVCResize(testCtx, "ldap", &requestedDataVolumeSize)

		require.NoError(t, err)
	})

	t.Run("should succeed to wait for pvc resize if actual size is bigger than requested size", func(t *testing.T) {
		requestedDataVolumeSize := resource.MustParse("3Gi")

		mPvcClient := newMockPvcClient(t)

		i := 0
		mPvcClient.EXPECT().Get(testCtx, "ldap", metav1.GetOptions{}).RunAndReturn(func(ctx context.Context, doguName string, options metav1.GetOptions) (*corev1.PersistentVolumeClaim, error) {
			// increase with each iteration
			i++
			assert.Less(t, i, 3)

			actulaDataVolumeSize := resource.MustParse("2Gi")
			if i == 2 {
				actulaDataVolumeSize = resource.MustParse("8Gi")
			}

			pvc := &corev1.PersistentVolumeClaim{
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: requestedDataVolumeSize,
						},
					},
				},
				Status: corev1.PersistentVolumeClaimStatus{
					Capacity: corev1.ResourceList{
						corev1.ResourceStorage: actulaDataVolumeSize,
					},
				},
			}

			return pvc, nil
		})

		resizer := &DoguVolumeResizer{
			pvcClient: mPvcClient,
		}

		waitSecondsBetweenRetries = 0
		defer func() {
			waitSecondsBetweenRetries = defaultWaitSecondsBetweenRetries
		}()

		err := resizer.waitForPVCResize(testCtx, "ldap", &requestedDataVolumeSize)

		require.NoError(t, err)
	})
}

func Test_defaultDoguVolumeResizer_ResizeDogusIfNeeded(t *testing.T) {
	testCtx := context.Background()
	t.Run("Should resize dogus if needed", func(t *testing.T) {
		mDoguClient := newMockDoguClient(t)
		dogu := &doguv2.Dogu{
			Spec: doguv2.DoguSpec{
				Resources: doguv2.DoguResources{
					MinDataVolumeSize: resource.MustParse("1Gi"),
				},
			},
		}
		mDoguClient.EXPECT().Get(testCtx, "ldap", metav1.GetOptions{}).Return(dogu, nil)
		mDoguClient.EXPECT().Update(testCtx, dogu, metav1.UpdateOptions{}).RunAndReturn(func(ctx context.Context, dogu *doguv2.Dogu, options metav1.UpdateOptions) (*doguv2.Dogu, error) {
			assert.Equal(t, "2Gi", dogu.Spec.Resources.MinDataVolumeSize.String())

			return dogu, nil
		})

		mPvcClient := newMockPvcClient(t)
		mPvcClient.EXPECT().Get(testCtx, "ldap", metav1.GetOptions{}).Return(&corev1.PersistentVolumeClaim{
			Spec: corev1.PersistentVolumeClaimSpec{
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("2Gi"),
					},
				},
			},
			Status: corev1.PersistentVolumeClaimStatus{
				Capacity: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("2Gi"),
				},
			},
		}, nil)

		resizer := &DoguVolumeResizer{
			doguClient:    mDoguClient,
			pvcClient:     mPvcClient,
			excludedDogus: []string{"excluded/dogu"},
		}

		waitSecondsBetweenRetries = 0
		defer func() {
			waitSecondsBetweenRetries = defaultWaitSecondsBetweenRetries
		}()

		exporterDogus := []migration.Dogu{
			{
				Name:    "official/ldap",
				Version: "1.2.3",
				Volume:  migration.DoguVolume{SizeInBytes: 2 * 1024 * 1024 * 1024},
			},
			{
				Name:    "official/otherDogu",
				Version: "1.2.3",
				Volume:  migration.DoguVolume{SizeInBytes: 2 * 1024 * 1024 * 1024},
			},
			{
				Name:    "official/cas",
				Version: "1.2.3",
				Volume:  migration.DoguVolume{SizeInBytes: 2 * 1024 * 1024 * 1024},
			},
			{
				Name:    "simpleName",
				Version: "1.2.3",
				Volume:  migration.DoguVolume{SizeInBytes: 2 * 1024 * 1024 * 1024},
			},
			{
				Name:    "excluded/dogu",
				Version: "1.2.3",
				Volume:  migration.DoguVolume{SizeInBytes: 2 * 1024 * 1024 * 1024},
			},
		}

		importerDogus := []migration.Dogu{
			{
				Name:    "official/ldap",
				Version: "1.2.3",
				Volume:  migration.DoguVolume{SizeInBytes: 1 * 1024 * 1024 * 1024},
			},
			{
				Name:    "official/cas",
				Version: "1.2.3",
				Volume:  migration.DoguVolume{SizeInBytes: 2 * 1024 * 1024 * 1024},
			},
			{
				Name:    "simpleName",
				Version: "1.2.3",
				Volume:  migration.DoguVolume{SizeInBytes: 1 * 1024 * 1024 * 1024},
			},
		}

		err := resizer.ResizeDogusIfNeeded(testCtx, exporterDogus, importerDogus)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to find dogu official/otherDogu in the importing system")
		assert.ErrorContains(t, err, "failed to resize dogu simpleName: dogu simpleName name is not a qualified dogu name: dogu name needs to be in the form 'namespace/dogu' but is 'simpleName'")
	})
}

func TestNewDoguVolumeResizer(t *testing.T) {
	t.Run("should create new DoguVolumeResizer", func(t *testing.T) {
		mDoguClient := newMockDoguClient(t)
		mPvcClient := newMockPvcClient(t)
		mDoguDescriptorDoguRepo := newMockDoguDescriptorRepo(t)

		dvr := NewDoguVolumeResizer(mDoguClient, mPvcClient, mDoguDescriptorDoguRepo, []string{"test1", "test2"})

		assert.NotNil(t, dvr)
		assert.Equal(t, mDoguClient, dvr.doguClient)
		assert.Equal(t, mPvcClient, dvr.pvcClient)
		assert.Equal(t, mDoguDescriptorDoguRepo, dvr.doguDescriptorRepo)
		assert.Equal(t, append([]string{"test1", "test2"}, doguNginx), dvr.excludedDogus)
	})
}

func TestDoguVolumeResizer_hasVolumeWithBackup(t *testing.T) {
	testCtx := context.Background()
	t.Run("should check if volume needs backup, return true if one volume needs backup", func(t *testing.T) {
		importerDogu := migration.Dogu{
			Name:    "official/ldap",
			Version: "1.2.3",
			Volume:  migration.DoguVolume{SizeInBytes: 1 * 1024 * 1024 * 1024},
		}
		mDoguDescriptorDoguRepo := newMockDoguDescriptorRepo(t)
		version, err := core.ParseVersion("1.2.3")
		require.NoError(t, err)
		expectedDogu := &core.Dogu{Volumes: []core.Volume{
			{
				NeedsBackup: false,
			},
			{
				NeedsBackup: true,
			},
		}}

		mDoguDescriptorDoguRepo.EXPECT().Get(testCtx, cescommons.NewSimpleNameVersion("ldap", version)).Return(expectedDogu, nil)

		dvr := &DoguVolumeResizer{
			doguDescriptorRepo: mDoguDescriptorDoguRepo,
		}
		result, err := dvr.hasVolumeWithBackup(testCtx, importerDogu)

		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("should check if volume needs backup, return false if no volume needs backup", func(t *testing.T) {
		importerDogu := migration.Dogu{
			Name:    "official/ldap",
			Version: "1.2.3",
			Volume:  migration.DoguVolume{SizeInBytes: 1 * 1024 * 1024 * 1024},
		}
		mDoguDescriptorDoguRepo := newMockDoguDescriptorRepo(t)
		version, err := core.ParseVersion("1.2.3")
		require.NoError(t, err)
		expectedDogu := &core.Dogu{Volumes: []core.Volume{
			{
				NeedsBackup: false,
			},
			{
				NeedsBackup: false,
			},
		}}

		mDoguDescriptorDoguRepo.EXPECT().Get(testCtx, cescommons.NewSimpleNameVersion("ldap", version)).Return(expectedDogu, nil)

		dvr := &DoguVolumeResizer{
			doguDescriptorRepo: mDoguDescriptorDoguRepo,
		}
		result, err := dvr.hasVolumeWithBackup(testCtx, importerDogu)

		require.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("should check if volume needs backup, throw error if dogu version can't be parsed", func(t *testing.T) {
		importerDogu := migration.Dogu{
			Name:    "official/ldap",
			Version: "abc",
			Volume:  migration.DoguVolume{SizeInBytes: 1 * 1024 * 1024 * 1024},
		}
		mDoguDescriptorDoguRepo := newMockDoguDescriptorRepo(t)

		dvr := &DoguVolumeResizer{
			doguDescriptorRepo: mDoguDescriptorDoguRepo,
		}
		result, err := dvr.hasVolumeWithBackup(testCtx, importerDogu)

		require.Error(t, err)
		assert.False(t, result)
		assert.ErrorContains(t, err, "failed to parse importer dogu version:")
	})

	t.Run("should check if volume needs backup, throw error if dogu can't be returned from repo", func(t *testing.T) {
		importerDogu := migration.Dogu{
			Name:    "ldap",
			Version: "1.2.3",
			Volume:  migration.DoguVolume{SizeInBytes: 1 * 1024 * 1024 * 1024},
		}
		mDoguDescriptorDoguRepo := newMockDoguDescriptorRepo(t)

		dvr := &DoguVolumeResizer{
			doguDescriptorRepo: mDoguDescriptorDoguRepo,
		}
		result, err := dvr.hasVolumeWithBackup(testCtx, importerDogu)

		require.Error(t, err)
		assert.False(t, result)
		assert.ErrorContains(t, err, "failed to get qualified dogu name")
	})

	t.Run("should check if volume needs backup, return false if no volume needs backup", func(t *testing.T) {
		importerDogu := migration.Dogu{
			Name:    "official/ldap",
			Version: "1.2.3",
			Volume:  migration.DoguVolume{SizeInBytes: 1 * 1024 * 1024 * 1024},
		}
		mDoguDescriptorDoguRepo := newMockDoguDescriptorRepo(t)
		version, err := core.ParseVersion("1.2.3")
		require.NoError(t, err)

		mDoguDescriptorDoguRepo.EXPECT().Get(testCtx, cescommons.NewSimpleNameVersion("ldap", version)).Return(nil, errors.New("error retrieving dogu"))

		dvr := &DoguVolumeResizer{
			doguDescriptorRepo: mDoguDescriptorDoguRepo,
		}
		result, err := dvr.hasVolumeWithBackup(testCtx, importerDogu)

		require.Error(t, err)
		assert.False(t, result)
		assert.ErrorContains(t, err, "failed to get dogu desciptor")
	})
}
