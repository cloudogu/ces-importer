# Running `ces-importer`

## Before the start

There are some requirements before the deployment:
- K8s secrets:
  - one that contains the exporter system's API key
     - the name of the secret and the data field that will hold the value must be configured in these two values:
        - `config.exporter.api.secretName` contains the name of the k8s secret resource, f. i. `ces-exporter-secret`
        - `config.exporter.api.secretDataKey` contains the name of the key which holds the value `apiKey`
     - creation example:
       `kubectl -n ecosystem create secret generic ces-exporter-secret --from-literal=apiKey=ApiKey-example-123`
  - one that contains the SSH private key which will be used to access the exporter system via SSH
     - the name of the secret and the data field that will hold the value must be configured in these two values:
        - `config.importer.ssh.secretName` contains the name of the k8s secret resource, f. i. `ces-importer-secret`
        - `config.importer.ssh.secretDataKey` contains the name of the key which holds the value `privateKey`
     - creation example:
       `kubectl -n ecosystem create secret generic ces-importer-secret --from-file=privateKey=yourPrivateKeyHere`
- one that contains the exporter system's API key
    - the name of the secret and the data field that will hold the value must be configured in these two values:
        - `config.exporter.smtp.secretName` contains the name of the k8s secret resource, f. i. `ces-exporter-secret`
        - `config.exporter.smtp.secretDataKey` contains the name of the key which holds the value `mailPassword`
    - creation example:
      `kubectl -n ecosystem create secret generic ces-exporter-secret --from-literal=mailPassword=yourMailPasswordHere`
  - one, that is required for pulling images of ces-importer-Deployment
     - that one must also be referenced in the respective field in the `values.yaml` file
- a Helm Chart `values.yaml` file filled-in with the relevant configuration data 

## Deploying the application with Helm

```shell
# template only
helm template -n ecosystem -f myvalues.yaml ces-importer oci://registry.cloudogu.com/testing/ces-importer-helm/ces-importer --version 0.0.1

# install
helm install -n ecosystem -f myvalues.yaml ces-importer oci://registry.cloudogu.com/testing/ces-importer-helm/ces-importer --version 0.0.1

helm uninstall -n ecosystem ces-importer
```

## Example `values.yaml`

The following is an abridged version of a possible `values.yaml`. Please see a full YAML file along with comments in the
path `k8s/helm/values.yaml`.

```yaml
main:
  image: registry.cloudogu.com/testing/ces-importer:0.0.1
  imagePullSecrets:
    - name: "ces-exporter-registries"
  imagePullPolicy: IfNotPresent
job:
  image: registry.cloudogu.com/testing/ces-importer:0.0.1
  imagePullSecrets:
    - name: "ces-exporter-registries"
config:
  exporter:
    host: my.classic.ces.exporter.net
    apiSecretName: "ces-exporter-secret"
    apiSecretDataKey: "apiKey"
  importer:
    sshSecretName: "ces-importer-secret"
    sshSecretDataKey: "privateKey"
  migration:
    regularSchedule: "0 4 * * *"
    finalSchedule: "2025-04-03 12:34:56Z"
```

The following image describes `ces-importer`:

![a diagram of two sysemts "exporter" as source und "importer" as target. Persons with the role administrator apply a Helm Chart and secrets to the importer cluster, which then takes up the work. First, an endpoint will be asked for further system specific Exporter host configuration data. Then a Job resource will be created which does the actual data import work.](images/ces-importer.drawio.png "Configuration points of ces-importer and how the configuration points end up in the runtime process.")
