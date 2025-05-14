package migration

import (
	"context"
	"fmt"
	v2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type PVCGetter struct {
	client       v1.PersistentVolumeClaimInterface
	doguSelector string
}

type doguPVC struct {
	doguName string
	pvcName  string
}

func NewPVCGetter(client v1.PersistentVolumeClaimInterface) *PVCGetter {
	doguSelector := metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      v2.DoguLabelName,
				Operator: metav1.LabelSelectorOpExists,
				Values:   nil,
			},
		},
	}

	// Convert the selector to a string
	selector := metav1.FormatLabelSelector(&doguSelector)

	return &PVCGetter{
		client:       client,
		doguSelector: selector,
	}
}

func (p PVCGetter) GetDoguVolumes(ctx context.Context) ([]doguPVC, error) {
	pvcList, err := p.client.List(ctx, metav1.ListOptions{LabelSelector: p.doguSelector})
	if err != nil {
		return nil, fmt.Errorf("failed to list PVCs: %w", err)
	}

	if pvcList == nil {
		return nil, fmt.Errorf("PCV list is nil")
	}

	doguVolumes := make([]doguPVC, 0, pvcList.Size())

	for _, pvc := range pvcList.Items {
		doguVolumes = append(doguVolumes, doguPVC{
			doguName: pvc.Labels[v2.DoguLabelName],
			pvcName:  pvc.Name,
		})
	}

	return doguVolumes, nil
}
