package migration

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestNewJobProvider(t *testing.T) {
	t.Run("should create job provider successfully", func(t *testing.T) {
		// given
		pvcClient := newMockPvcClient(t)
		deps := JobProviderDependencies{
			JobContainerConfig: configuration.JobContainer{
				JobConfigMap:      "test-config-map",
				JobServiceAccount: "test-service-account",
				Image: configuration.ContainerImage{
					Registry:   "registry.example.com",
					Repository: "test-repo",
					Tag:        "latest",
				},
				ImagePullPolicy: "IfNotPresent",
				ImagePullSecrets: []configuration.ImagePullSecret{
					{Name: "test-secret"},
				},
				Resources: configuration.ResourceRequirements{
					Limits: configuration.ResourceList{
						CPU:    "100m",
						Memory: "128Mi",
					},
					Requests: configuration.ResourceList{
						CPU:    "50m",
						Memory: "64Mi",
					},
				},
			},
			SSHConfig: configuration.SSH{
				User:              "test-user",
				PrivateSSHKeyPath: "/path/to/key",
				SecretName:        "test-secret",
				SecretDataKey:     "test-key",
			},
			APIKey:    "test-api-key",
			PVCClient: pvcClient,
		}

		// when
		provider, err := newJobProvider(deps)

		// then
		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, deps.SSHConfig, provider.sshConfig)
		assert.Equal(t, pvcClient, provider.pvcClient)
		assert.Equal(t, "test-config-map", provider.jobConfigMap)
		assert.Equal(t, "test-service-account", provider.serviceAccount)
	})

	t.Run("should return error when createJobSpec fails", func(t *testing.T) {
		// given
		pvcClient := newMockPvcClient(t)
		deps := JobProviderDependencies{
			JobContainerConfig: configuration.JobContainer{
				JobConfigMap:      "test-config-map",
				JobServiceAccount: "test-service-account",
				Image: configuration.ContainerImage{
					Registry:   "registry.example.com",
					Repository: "test-repo",
					Tag:        "latest",
				},
				ImagePullPolicy: "InvalidPolicy", // This will cause parseImagePullPolicy to fail
				Resources: configuration.ResourceRequirements{
					Limits: configuration.ResourceList{
						CPU:    "100m",
						Memory: "128Mi",
					},
					Requests: configuration.ResourceList{
						CPU:    "50m",
						Memory: "64Mi",
					},
				},
			},
			SSHConfig: configuration.SSH{
				User:              "test-user",
				PrivateSSHKeyPath: "/path/to/key",
				SecretName:        "test-secret",
				SecretDataKey:     "test-key",
			},
			APIKey:    "test-api-key",
			PVCClient: pvcClient,
		}

		// when
		provider, err := newJobProvider(deps)

		// then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create specification for job")
		assert.Nil(t, provider)
	})
}

func TestJobProvider_createImportJob(t *testing.T) {
	t.Run("should create import job successfully", func(t *testing.T) {
		// given
		ctx := context.Background()
		pvcClient := newMockPvcClient(t)
		pvcClient.EXPECT().GetDoguVolumes(ctx).Return([]doguPVC{
			{doguName: "jenkins", pvcName: "jenkins-data"},
			{doguName: "cas", pvcName: "cas-data"},
		}, nil)

		provider := jobProvider{
			jobSpec: jobSpec{
				imageURL:         "registry.example.com/test-repo:latest",
				imagePullPolicy:  "IfNotPresent",
				imagePullSecrets: []v1.LocalObjectReference{{Name: "test-secret"}},
				restartPolicy:    v1.RestartPolicyNever,
				env:              []v1.EnvVar{{Name: "API_KEY", Value: "test-api-key"}},
				jobConfigMap:     "test-config-map",
				serviceAccount:   "test-service-account",
			},
			pvcClient: pvcClient,
			sshConfig: configuration.SSH{
				User:              "test-user",
				PrivateSSHKeyPath: "/path/to/key",
				SecretName:        "test-secret",
				SecretDataKey:     "test-key",
			},
			doguVolumeBasePath: "/volumes",
		}

		// when
		job, err := provider.createImportJob(ctx)

		// then
		assert.NoError(t, err)
		assert.NotNil(t, job)

		assert.Contains(t, job.Name, jobName)
		assert.Equal(t, fmt.Sprintf("%s-container", jobName), job.Spec.Template.Spec.Containers[0].Name)
		assert.Equal(t, "registry.example.com/test-repo:latest", job.Spec.Template.Spec.Containers[0].Image)
		assert.Equal(t, v1.PullPolicy("IfNotPresent"), job.Spec.Template.Spec.Containers[0].ImagePullPolicy)
		assert.Equal(t, "test-service-account", job.Spec.Template.Spec.ServiceAccountName)

		// Verify that the volumes are set up correctly
		assert.Equal(t, 4, len(job.Spec.Template.Spec.Volumes))                    // 1 config volume + 2 dogu volumes + 1 ssh key volume
		assert.Equal(t, 4, len(job.Spec.Template.Spec.Containers[0].VolumeMounts)) // 1 config mount + 2 dogu mounts + 1 ssh key mount

		// Verify env for fqdn change is NOT set
		assert.Equal(t, 1, len(job.Spec.Template.Spec.Containers[0].Env))
		assert.Equal(t, "API_KEY", job.Spec.Template.Spec.Containers[0].Env[0].Name)
		assert.Equal(t, "test-api-key", job.Spec.Template.Spec.Containers[0].Env[0].Value)
	})

	t.Run("should create import job with FQDN Change successfully", func(t *testing.T) {
		// given
		ctx := context.Background()
		ctx = SetFinalMigration(ctx)
		ctx = SetTriggerFQDNChange(ctx)

		pvcClient := newMockPvcClient(t)
		pvcClient.EXPECT().GetDoguVolumes(ctx).Return([]doguPVC{
			{doguName: "jenkins", pvcName: "jenkins-data"},
			{doguName: "cas", pvcName: "cas-data"},
		}, nil)

		provider := jobProvider{
			jobSpec: jobSpec{
				imageURL:         "registry.example.com/test-repo:latest",
				imagePullPolicy:  "IfNotPresent",
				imagePullSecrets: []v1.LocalObjectReference{{Name: "test-secret"}},
				restartPolicy:    v1.RestartPolicyNever,
				env:              []v1.EnvVar{{Name: "API_KEY", Value: "test-api-key"}},
				jobConfigMap:     "test-config-map",
				serviceAccount:   "test-service-account",
			},
			pvcClient: pvcClient,
			sshConfig: configuration.SSH{
				User:              "test-user",
				PrivateSSHKeyPath: "/path/to/key",
				SecretName:        "test-secret",
				SecretDataKey:     "test-key",
			},
			doguVolumeBasePath: "/volumes",
		}

		// when
		job, err := provider.createImportJob(ctx)

		// then
		assert.NoError(t, err)
		assert.NotNil(t, job)

		assert.Contains(t, job.Name, jobName)
		assert.Equal(t, fmt.Sprintf("%s-container", jobName), job.Spec.Template.Spec.Containers[0].Name)
		assert.Equal(t, "registry.example.com/test-repo:latest", job.Spec.Template.Spec.Containers[0].Image)
		assert.Equal(t, v1.PullPolicy("IfNotPresent"), job.Spec.Template.Spec.Containers[0].ImagePullPolicy)
		assert.Equal(t, "test-service-account", job.Spec.Template.Spec.ServiceAccountName)

		// Verify that the volumes are set up correctly
		assert.Equal(t, 4, len(job.Spec.Template.Spec.Volumes))                    // 1 config volume + 2 dogu volumes + 1 ssh key volume
		assert.Equal(t, 4, len(job.Spec.Template.Spec.Containers[0].VolumeMounts)) // 1 config mount + 2 dogu mounts + 1 ssh key mount

		// Verify env for fqdn change is set
		assert.Equal(t, 2, len(job.Spec.Template.Spec.Containers[0].Env))
		assert.Equal(t, "TRIGGER_FQDN_CHANGE", job.Spec.Template.Spec.Containers[0].Env[1].Name)
		assert.Equal(t, "true", job.Spec.Template.Spec.Containers[0].Env[1].Value)
	})

	t.Run("should return error when GetDoguVolumes fails", func(t *testing.T) {
		// given
		ctx := context.Background()
		pvcClient := newMockPvcClient(t)
		pvcClient.EXPECT().GetDoguVolumes(ctx).Return([]doguPVC{}, assert.AnError)

		provider := jobProvider{
			jobSpec: jobSpec{
				imageURL:         "registry.example.com/test-repo:latest",
				imagePullPolicy:  "IfNotPresent",
				imagePullSecrets: []v1.LocalObjectReference{{Name: "test-secret"}},
				restartPolicy:    v1.RestartPolicyNever,
				env:              []v1.EnvVar{{Name: "API_KEY", Value: "test-api-key"}},
				jobConfigMap:     "test-config-map",
				serviceAccount:   "test-service-account",
			},
			pvcClient: pvcClient,
			sshConfig: configuration.SSH{
				User:              "test-user",
				PrivateSSHKeyPath: "/path/to/key",
				SecretName:        "test-secret",
				SecretDataKey:     "test-key",
			},
			doguVolumeBasePath: "/volumes",
		}

		// when
		job, err := provider.createImportJob(ctx)

		// then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get dogu volumes")
		assert.Nil(t, job)
	})
}

func Test_createJobSpec(t *testing.T) {
	t.Run("should create job spec successfully", func(t *testing.T) {
		// given
		jcCfg := configuration.JobContainer{
			JobConfigMap:      "test-config-map",
			JobServiceAccount: "test-service-account",
			Image: configuration.ContainerImage{
				Registry:   "registry.example.com",
				Repository: "test-repo",
				Tag:        "latest",
			},
			ImagePullPolicy: "IfNotPresent",
			ImagePullSecrets: []configuration.ImagePullSecret{
				{Name: "test-secret"},
			},
			Resources: configuration.ResourceRequirements{
				Limits: configuration.ResourceList{
					CPU:    "100m",
					Memory: "128Mi",
				},
				Requests: configuration.ResourceList{
					CPU:    "50m",
					Memory: "64Mi",
				},
			},
		}
		apiKey := "test-api-key"

		// when
		spec, err := createJobSpec(jcCfg, apiKey)

		// then
		assert.NoError(t, err)
		assert.Equal(t, "test-config-map", spec.jobConfigMap)
		assert.Equal(t, "test-service-account", spec.serviceAccount)
		assert.Contains(t, spec.imageURL, "registry.example.com/test-repo:latest")
		assert.Equal(t, "IfNotPresent", string(spec.imagePullPolicy))
		assert.Len(t, spec.imagePullSecrets, 1)
		assert.Equal(t, "test-secret", spec.imagePullSecrets[0].Name)

		// Check that the environment variables are set correctly
		assert.Len(t, spec.env, 3)
		assert.Equal(t, "API_KEY", spec.env[0].Name)
		assert.Equal(t, apiKey, spec.env[0].Value)
	})

	t.Run("should return error when parseRequirements fails", func(t *testing.T) {
		tests := []struct {
			name          string
			resourceInput configuration.ResourceRequirements
		}{
			{
				name: "invalid CPU limit",
				resourceInput: configuration.ResourceRequirements{
					Limits: configuration.ResourceList{
						CPU:    "invalid",
						Memory: "128Mi",
					},
					Requests: configuration.ResourceList{
						CPU:    "50m",
						Memory: "64Mi",
					},
				},
			},
			{
				name: "invalid memory limit",
				resourceInput: configuration.ResourceRequirements{
					Limits: configuration.ResourceList{
						CPU:    "100m",
						Memory: "invalid",
					},
					Requests: configuration.ResourceList{
						CPU:    "50m",
						Memory: "64Mi",
					},
				},
			},
			{
				name: "invalid CPU request",
				resourceInput: configuration.ResourceRequirements{
					Limits: configuration.ResourceList{
						CPU:    "100m",
						Memory: "128Mi",
					},
					Requests: configuration.ResourceList{
						CPU:    "invalid",
						Memory: "64Mi",
					},
				},
			},
			{
				name: "invalid memory request",
				resourceInput: configuration.ResourceRequirements{
					Limits: configuration.ResourceList{
						CPU:    "100m",
						Memory: "128Mi",
					},
					Requests: configuration.ResourceList{
						CPU:    "50m",
						Memory: "invalid",
					},
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				jcCfg := configuration.JobContainer{
					Resources: tt.resourceInput,
				}
				spec, err := createJobSpec(jcCfg, "test-api-key")

				assert.Error(t, err)
				assert.Equal(t, jobSpec{}, spec)
			})
		}
	})

	t.Run("should return error when parseImage fails", func(t *testing.T) {
		// given
		jcCfg := configuration.JobContainer{
			Image: configuration.ContainerImage{
				Registry:   "registry.example.com",
				Repository: "invalid%repo",
				Tag:        "latest",
			},
			Resources: configuration.ResourceRequirements{
				Limits: configuration.ResourceList{
					CPU:    "100m",
					Memory: "128Mi",
				},
				Requests: configuration.ResourceList{
					CPU:    "50m",
					Memory: "64Mi",
				},
			},
		}
		apiKey := "test-api-key"

		// when
		spec, err := createJobSpec(jcCfg, apiKey)

		// then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse image")
		assert.Equal(t, jobSpec{}, spec)
	})

	t.Run("test_createJobSpec_imagePullPolicy", func(t *testing.T) {
		tests := []struct {
			name            string
			imagePullPolicy string
			expectedPolicy  v1.PullPolicy
			expErr          bool
		}{
			{
				name:            "Always",
				imagePullPolicy: "Always",
				expectedPolicy:  v1.PullAlways,
			},
			{
				name:            "Never",
				imagePullPolicy: "Never",
				expectedPolicy:  v1.PullNever,
			},
			{
				name:            "IfNotPresent",
				imagePullPolicy: "IfNotPresent",
				expectedPolicy:  v1.PullIfNotPresent,
			},
			{
				name:            "empty policy",
				imagePullPolicy: "",
				expectedPolicy:  "",
				expErr:          true,
			},
			{
				name:            "invalid policy",
				imagePullPolicy: "InvalidPolicy",
				expectedPolicy:  "",
				expErr:          true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// given
				jcCfg := configuration.JobContainer{
					Image: configuration.ContainerImage{
						Registry:   "registry.example.com",
						Repository: "test-repo",
						Tag:        "latest",
					},
					ImagePullPolicy: tt.imagePullPolicy,
					Resources: configuration.ResourceRequirements{
						Limits: configuration.ResourceList{
							CPU:    "100m",
							Memory: "128Mi",
						},
						Requests: configuration.ResourceList{
							CPU:    "50m",
							Memory: "64Mi",
						},
					},
				}
				apiKey := "test-api-key"

				// when
				spec, err := createJobSpec(jcCfg, apiKey)

				// then
				if tt.expErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
				assert.Equal(t, tt.expectedPolicy, spec.imagePullPolicy)
			})
		}
	})
}
