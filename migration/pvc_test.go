package migration

import (
	"context"
	"github.com/cloudogu/ces-importer/migration/mocks"
	v2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_newPVCGetter(t *testing.T) {
	pGetter := newPVCGetter(mocks.NewPersistentVolumeClaimInterface(t))

	assert.NotNil(t, pGetter)
	assert.NotNil(t, pGetter.client)
	assert.Contains(t, pGetter.doguSelector, v2.DoguLabelName)
}

func Test_pvcGetter_GetDoguVolumes(t *testing.T) {
	t.Run("get pvc volumes", func(t *testing.T) {
		pvcClientMock := mocks.NewPersistentVolumeClaimInterface(t)
		pvcClientMock.EXPECT().List(mock.Anything, mock.Anything).Return(&v1.PersistentVolumeClaimList{
			Items: []v1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "jenkins-data",
						Labels: map[string]string{
							v2.DoguLabelName: "jenkins",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cas-data",
						Labels: map[string]string{
							v2.DoguLabelName: "cas",
						},
					},
				},
			},
		}, nil)

		pGetter := pvcGetter{
			client:       pvcClientMock,
			doguSelector: "",
		}

		pvcList, err := pGetter.GetDoguVolumes(context.TODO())
		assert.NoError(t, err)
		assert.Equal(t, 2, len(pvcList))

		assert.Equal(t, "jenkins", pvcList[0].doguName)
		assert.Equal(t, "jenkins-data", pvcList[0].pvcName)

		assert.Equal(t, "cas", pvcList[1].doguName)
		assert.Equal(t, "cas-data", pvcList[1].pvcName)
	})

	t.Run("pvc client returns error", func(t *testing.T) {
		pvcClientMock := mocks.NewPersistentVolumeClaimInterface(t)
		pvcClientMock.EXPECT().List(mock.Anything, mock.Anything).Return(nil, assert.AnError)

		pGetter := pvcGetter{
			client:       pvcClientMock,
			doguSelector: "",
		}

		pvcList, err := pGetter.GetDoguVolumes(context.TODO())
		assert.ErrorIs(t, err, assert.AnError)
		assert.Nil(t, pvcList)
	})

	t.Run("client returns pvc list with value nil", func(t *testing.T) {
		pvcClientMock := mocks.NewPersistentVolumeClaimInterface(t)
		pvcClientMock.EXPECT().List(mock.Anything, mock.Anything).Return(nil, nil)

		pGetter := pvcGetter{
			client:       pvcClientMock,
			doguSelector: "",
		}

		pvcList, err := pGetter.GetDoguVolumes(context.TODO())
		assert.Error(t, err)
		assert.Nil(t, pvcList)
	})

}
