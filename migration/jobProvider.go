package migration

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-importer/configuration"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/url"
	"os"
	"path"
	"strconv"
	"time"
)

const jobName = "migration-job"

var jobConfigPath = fmt.Sprintf("/etc/%s/config", jobName)

type volumeMounts struct {
	volumes []v1.Volume
	mounts  []v1.VolumeMount
}

func collectVolumeMounts(vmList ...volumeMounts) volumeMounts {
	var result volumeMounts

	for _, vm := range vmList {
		result.volumes = append(result.volumes, vm.volumes...)
		result.mounts = append(result.mounts, vm.mounts...)
	}

	return result
}

func createConfigVolumeMount(configMapName string) volumeMounts {
	configVolumeName := "config-volume"

	configVolume := v1.Volume{
		Name: configVolumeName,
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{
					Name: configMapName,
				},
			},
		},
	}

	configVolumeMount := v1.VolumeMount{
		Name:      configVolumeName,
		MountPath: jobConfigPath,
		ReadOnly:  true,
	}

	return volumeMounts{
		volumes: []v1.Volume{configVolume},
		mounts:  []v1.VolumeMount{configVolumeMount},
	}
}

func createDoguVolumeMounts(volumeBasePath string, pvcList []doguPVC) volumeMounts {
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
			MountPath: path.Join(volumeBasePath, pvc.doguName),
			ReadOnly:  false,
		}

		mounts = append(mounts, mount)
	}

	return volumeMounts{
		volumes: volumes,
		mounts:  mounts,
	}
}

func createSSHPrivateKeyMount(secretName, secretDataKey, privateSSHKeyPath string) volumeMounts {
	permissions := int32(0400)
	sshVolumeName := "ssh-private-key-volume"

	secretVolume := v1.Volume{
		Name: sshVolumeName,
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName:  secretName,
				DefaultMode: &permissions,
				Items: []v1.KeyToPath{
					{
						Key:  secretDataKey,
						Path: path.Base(privateSSHKeyPath),
					},
				},
			},
		},
	}

	secretVolumeMount := v1.VolumeMount{
		Name:      sshVolumeName,
		MountPath: path.Dir(privateSSHKeyPath),
		ReadOnly:  true,
	}

	return volumeMounts{
		volumes: []v1.Volume{secretVolume},
		mounts:  []v1.VolumeMount{secretVolumeMount},
	}
}

type jobSpec struct {
	imageURL         string
	imagePullPolicy  v1.PullPolicy
	imagePullSecrets []v1.LocalObjectReference
	resources        v1.ResourceRequirements
	restartPolicy    v1.RestartPolicy
	env              []v1.EnvVar
	serviceAccount   string
	jobConfigMap     string
}

type jobProviderDependencies struct {
	JobContainerConfig configuration.JobContainer
	SSHConfig          configuration.SSH
	APIKey             string
	DoguVolumeBasePath string
	PVCClient          pvcClient
}

func newJobProvider(deps jobProviderDependencies) (*jobProvider, error) {
	jSpec, err := createJobSpec(deps.JobContainerConfig, deps.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create specification for job: %w", err)
	}

	return &jobProvider{
		jobSpec:            jSpec,
		sshConfig:          deps.SSHConfig,
		pvcClient:          deps.PVCClient,
		doguVolumeBasePath: deps.DoguVolumeBasePath,
	}, nil
}

func createJobSpec(jcCfg configuration.JobContainer, apiKey string) (jobSpec, error) {
	requirements, err := parseRequirements(jcCfg)
	if err != nil {
		return jobSpec{}, fmt.Errorf("failed to parse resource requirements: %w", err)
	}

	image, err := parseImage(jcCfg)
	if err != nil {
		return jobSpec{}, fmt.Errorf("failed to parse image: %w", err)
	}

	imagePullPolicy, err := parseImagePullPolicy(jcCfg.ImagePullPolicy)
	if err != nil {
		return jobSpec{}, fmt.Errorf("failed to parse image pull policy: %w", err)
	}

	imagePullSecrets := parseImagePullSecrets(jcCfg)

	envs := createContainerEnv(apiKey)

	return jobSpec{
		imageURL:         image,
		imagePullPolicy:  imagePullPolicy,
		imagePullSecrets: imagePullSecrets,
		resources:        requirements,
		restartPolicy:    v1.RestartPolicyNever,
		env:              envs,
		jobConfigMap:     jcCfg.JobConfigMap,
		serviceAccount:   jcCfg.JobServiceAccount,
	}, nil
}

func parseRequirements(jcCfg configuration.JobContainer) (v1.ResourceRequirements, error) {
	limitCPU, err := resource.ParseQuantity(jcCfg.Resources.Limits.CPU)
	if err != nil {
		return v1.ResourceRequirements{}, fmt.Errorf("failed to parse cpu limit: %w", err)
	}

	limitMemory, err := resource.ParseQuantity(jcCfg.Resources.Limits.Memory)
	if err != nil {
		return v1.ResourceRequirements{}, fmt.Errorf("failed to parse memory limit: %w", err)
	}

	requestCPU, err := resource.ParseQuantity(jcCfg.Resources.Requests.CPU)
	if err != nil {
		return v1.ResourceRequirements{}, fmt.Errorf("failed to parse cpu request: %w", err)
	}

	requestMemory, err := resource.ParseQuantity(jcCfg.Resources.Requests.Memory)
	if err != nil {
		return v1.ResourceRequirements{}, fmt.Errorf("failed to parse memory request: %w", err)
	}

	return v1.ResourceRequirements{
		Limits: v1.ResourceList{
			v1.ResourceCPU:    limitCPU,
			v1.ResourceMemory: limitMemory,
		},
		Requests: v1.ResourceList{
			v1.ResourceCPU:    requestCPU,
			v1.ResourceMemory: requestMemory,
		},
	}, nil
}

func parseImage(jcCfg configuration.JobContainer) (string, error) {
	imageURL, err := url.Parse(path.Join(jcCfg.Image.Registry, jcCfg.Image.Repository))
	if err != nil {
		return "", fmt.Errorf("failed to parse image url: %w", err)
	}

	imageURLWithTag := fmt.Sprintf("%s:%s", imageURL, jcCfg.Image.Tag)

	return imageURLWithTag, nil
}

func parseImagePullPolicy(policy string) (v1.PullPolicy, error) {
	switch policy {
	case string(v1.PullAlways):
		return v1.PullAlways, nil
	case string(v1.PullNever):
		return v1.PullNever, nil
	case string(v1.PullIfNotPresent):
		return v1.PullIfNotPresent, nil
	default:
		return "", fmt.Errorf("invalid image pull policy: %s", policy)
	}
}

func parseImagePullSecrets(jcCfg configuration.JobContainer) []v1.LocalObjectReference {
	var result []v1.LocalObjectReference

	for _, secret := range jcCfg.ImagePullSecrets {
		result = append(result, v1.LocalObjectReference{
			Name: secret.Name,
		})
	}

	return result
}

func createContainerEnv(apiKey string) []v1.EnvVar {
	return []v1.EnvVar{
		{
			Name:  "API_KEY",
			Value: apiKey,
		},
		{
			Name:  configuration.EnvBaseConfigPathKey,
			Value: jobConfigPath,
		},
		{
			Name:  configuration.EnvImporterNamespaceKey,
			Value: os.Getenv(configuration.EnvImporterNamespaceKey),
		},
	}
}

type jobProvider struct {
	jobSpec
	pvcClient
	sshConfig          configuration.SSH
	doguVolumeBasePath string
}

func (j jobProvider) createImportJob(ctx context.Context) (*batchv1.Job, error) {
	backoffLimit := int32(0) // Allow no retries for the job before failing the job

	pvcList, err := j.GetDoguVolumes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get dogu volumes: %w", err)
	}

	jobConfigVolumeMount := createConfigVolumeMount(j.jobConfigMap)
	doguVolumeMounts := createDoguVolumeMounts(j.doguVolumeBasePath, pvcList)
	sshKeyVolumeMount := createSSHPrivateKeyMount(j.sshConfig.SecretName, j.sshConfig.SecretDataKey, j.sshConfig.PrivateSSHKeyPath)

	jobVolumeMounts := collectVolumeMounts(jobConfigVolumeMount, doguVolumeMounts, sshKeyVolumeMount)

	unixTimeStr := strconv.FormatInt(time.Now().Unix(), 10)

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: jobName,
			Labels: map[string]string{
				"app.kubernetes.io/name":      jobName,
				"app.kubernetes.io/instance":  fmt.Sprintf("%s-%s", jobName, unixTimeStr),
				"app.kubernetes.io/component": "job",
				"app.kubernetes.io/part-of":   "migration",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Volumes: jobVolumeMounts.volumes,
					Containers: []v1.Container{
						{
							Name:            fmt.Sprintf("%s-container", jobName),
							Image:           j.imageURL,
							Env:             j.env,
							VolumeMounts:    jobVolumeMounts.mounts,
							ImagePullPolicy: j.imagePullPolicy,
						},
					},
					RestartPolicy:      j.restartPolicy,
					ServiceAccountName: j.serviceAccount,
					ImagePullSecrets:   j.imagePullSecrets,
				},
			},
		},
	}, nil
}
