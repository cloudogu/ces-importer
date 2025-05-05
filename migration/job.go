package migration

import (
	"fmt"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path"
)

type volumeMounts struct {
	volumes []v1.Volume
	mounts  []v1.VolumeMount
}

func createVolumeMounts(pvcList []doguPVC) volumeMounts {
	volumes := make([]v1.Volume, 0, len(pvcList))
	mounts := make([]v1.VolumeMount, 0, len(pvcList))

	for _, pvc := range pvcList {
		volumeName := fmt.Sprintf("%s-data", pvc.doguName)

		volume := v1.Volume{
			Name: volumeName,
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc.pvcName,
					ReadOnly:  false,
				},
			},
		}

		volumes = append(volumes, volume)

		mount := v1.VolumeMount{
			Name:      volumeName,
			MountPath: path.Join("/data", pvc.doguName),
			ReadOnly:  false,
		}

		mounts = append(mounts, mount)
	}

	return volumeMounts{
		volumes: volumes,
		mounts:  mounts,
	}
}

func createImportJob() *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: batchv1.JobSpec{
			Parallelism:           nil,
			Completions:           nil,
			ActiveDeadlineSeconds: nil,
			PodFailurePolicy:      nil,
			SuccessPolicy:         nil,
			BackoffLimit:          nil,
			BackoffLimitPerIndex:  nil,
			MaxFailedIndexes:      nil,
			Selector:              nil,
			ManualSelector:        nil,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: v1.PodSpec{
					Volumes:                       nil,
					InitContainers:                nil,
					Containers:                    nil,
					EphemeralContainers:           nil,
					RestartPolicy:                 "",
					TerminationGracePeriodSeconds: nil,
					ActiveDeadlineSeconds:         nil,
					DNSPolicy:                     "",
					NodeSelector:                  nil,
					ServiceAccountName:            "",
					DeprecatedServiceAccount:      "",
					AutomountServiceAccountToken:  nil,
					NodeName:                      "",
					HostNetwork:                   false,
					HostPID:                       false,
					HostIPC:                       false,
					ShareProcessNamespace:         nil,
					SecurityContext:               nil,
					ImagePullSecrets:              nil,
					Hostname:                      "",
					Subdomain:                     "",
					Affinity:                      nil,
					SchedulerName:                 "",
					Tolerations:                   nil,
					HostAliases:                   nil,
					PriorityClassName:             "",
					Priority:                      nil,
					DNSConfig:                     nil,
					ReadinessGates:                nil,
					RuntimeClassName:              nil,
					EnableServiceLinks:            nil,
					PreemptionPolicy:              nil,
					Overhead:                      nil,
					TopologySpreadConstraints:     nil,
					SetHostnameAsFQDN:             nil,
					OS:                            nil,
					HostUsers:                     nil,
					SchedulingGates:               nil,
					ResourceClaims:                nil,
					Resources:                     nil,
				},
			},
			TTLSecondsAfterFinished: nil,
			CompletionMode:          nil,
			Suspend:                 nil,
			PodReplacementPolicy:    nil,
			ManagedBy:               nil,
		},
	}
}
