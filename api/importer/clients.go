package importer

import (
	"fmt"
	backupEcosystem "github.com/cloudogu/k8s-backup-operator/pkg/api/ecosystem"
	componentEcoClient "github.com/cloudogu/k8s-component-operator/pkg/api/ecosystem"
	doguLibClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"k8s.io/client-go/kubernetes"
	batchv1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

var (
	getK8sClientsSet    = kubernetes.NewForConfig
	getEcoSystemClient  = doguLibClient.NewForConfig
	getComponentsClient = componentEcoClient.NewForConfig
	getBackupClient     = backupEcosystem.NewForConfig
)

type K8sClients struct {
	Pvc            corev1.PersistentVolumeClaimInterface
	Pod            corev1.PodInterface
	Job            batchv1.JobInterface
	ConfigMap      corev1.ConfigMapInterface
	Secret         corev1.SecretInterface
	Dogu           doguLibClient.DoguInterface
	DoguControl    *DoguControl
	Component      componentEcoClient.ComponentInterface
	BackupSchedule backupEcosystem.BackupScheduleInterface
}

func CreateK8SClientSet(k8sRestConfig *rest.Config, namespace string) (K8sClients, error) {
	k8sClientSet, err := getK8sClientsSet(k8sRestConfig)
	if err != nil {
		return K8sClients{}, fmt.Errorf("failed to create k8s client set: %w", err)
	}

	k8sCoreClient := k8sClientSet.CoreV1()
	k8sPVCClient := k8sCoreClient.PersistentVolumeClaims(namespace)
	k8sPodClient := k8sCoreClient.Pods(namespace)
	k8sConfigMapClient := k8sCoreClient.ConfigMaps(namespace)
	k8sSecretClient := k8sCoreClient.Secrets(namespace)

	k8sJobClient := k8sClientSet.BatchV1().Jobs(namespace)

	ecoSystemClient, err := getEcoSystemClient(k8sRestConfig)
	if err != nil {
		return K8sClients{}, fmt.Errorf("failed to create ecosystem client: %w", err)
	}

	k8sDoguClient := ecoSystemClient.Dogus(namespace)
	doguControl := NewDoguControl(k8sDoguClient)

	v1Alpha1Client, err := getComponentsClient(k8sRestConfig)
	if err != nil {
		return K8sClients{}, fmt.Errorf("failed to create component client: %w", err)
	}

	k8sComponentClient := v1Alpha1Client.Components(namespace)

	backupClient, err := getBackupClient(k8sRestConfig)
	if err != nil {
		return K8sClients{}, fmt.Errorf("failed to create ecosystem backup client: %w", err)
	}

	backupScheduleClient := backupClient.BackupSchedules(namespace)

	return K8sClients{
		Pvc:            k8sPVCClient,
		Pod:            k8sPodClient,
		Job:            k8sJobClient,
		ConfigMap:      k8sConfigMapClient,
		Secret:         k8sSecretClient,
		Dogu:           k8sDoguClient,
		DoguControl:    doguControl,
		Component:      k8sComponentClient,
		BackupSchedule: backupScheduleClient,
	}, nil
}
