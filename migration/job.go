package migration

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-importer/configuration"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path"
)

type volumeMounts struct {
	volumes []v1.Volume
	mounts  []v1.VolumeMount
}

func createDoguVolumeMounts(pvcList []doguPVC) volumeMounts {
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

func createSSHPrivateKeyMount(privateSSHKeyPath string) volumeMounts {
	permissions := int32(0400)

	secretVolume := v1.Volume{
		Name: "ssh-privateKey",
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName:  "TODO",
				DefaultMode: &permissions,
			},
		},
	}

	secretVolumeMount := v1.VolumeMount{
		Name:      "ssh-privateKey",
		MountPath: privateSSHKeyPath,
		ReadOnly:  true,
	}

	return volumeMounts{
		volumes: []v1.Volume{secretVolume},
		mounts:  []v1.VolumeMount{secretVolumeMount},
	}
}

type pvcClient interface {
	GetDoguVolumes(ctx context.Context) ([]doguPVC, error)
}

type JobProvider struct {
	apiConfig configuration.API
	sshConfig configuration.SSH
	pvcClient
}

func (j JobProvider) createImportJob(ctx context.Context) (*batchv1.Job, error) {
	backoffLimit := int32(0) // Allow no retries for the job before failing the job

	pvcList, err := j.GetDoguVolumes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get dogu volumes: %w", err)
	}

	var jobVolumeMounts volumeMounts

	doguVolumeMounts := createDoguVolumeMounts(pvcList)
	jobVolumeMounts.volumes = append(jobVolumeMounts.volumes, doguVolumeMounts.volumes...)
	jobVolumeMounts.mounts = append(jobVolumeMounts.mounts, doguVolumeMounts.mounts...)

	sshKeyVolumeMount := createSSHPrivateKeyMount(j.sshConfig.PrivateSSHKeyPath)
	jobVolumeMounts.volumes = append(jobVolumeMounts.volumes, sshKeyVolumeMount.volumes...)
	jobVolumeMounts.mounts = append(jobVolumeMounts.mounts, sshKeyVolumeMount.mounts...)

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: "migration-job",
			Labels: map[string]string{
				"app.kubernetes.io/name":      "migration-job",
				"app.kubernetes.io/instance":  "migration-job-1",
				"app.kubernetes.io/component": "job",
				"app.kubernetes.io/part-of":   "migration",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: v1.PodSpec{
					Volumes: jobVolumeMounts.volumes,
					Containers: []v1.Container{
						{
							Name:         "",
							Image:        "",
							Env:          []v1.EnvVar{},
							VolumeMounts: jobVolumeMounts.mounts,
							// This is important for dev purposes but not the best decision for productive code later
							ImagePullPolicy: "Always",
						},
					},
					RestartPolicy:      v1.RestartPolicyNever,
					ServiceAccountName: "TODO",
					ImagePullSecrets:   nil,
				},
			},
		},
	}, nil
}
