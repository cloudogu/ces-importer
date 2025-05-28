# Running `ces-importer`

## Before the start

There are some requirements before the deployment:
- K8s secrets:
  - one that contains the exporter system's API key
     - the name of the secret and the data field that will hold the value must be configured in these two values:
        - `config.exporter.apiSecretName` contains the name of the k8s secret resource, f. i. `ces-exporter-secret`
        - `config.exporter.apiSecretDataKey` contains the name of the key which holds the value `apiKey`
     - creation example:
       `kubectl -n ecosystem create secret generic ces-exporter-secret --from-literal=apiKey=ApiKey-example-123`
  - one that contains the SSH private key which will be used to access the exporter system via SSH
     - the name of the secret and the data field that will hold the value must be configured in these two values:
        - `config.importer.sshSecretName` contains the name of the k8s secret resource, f. i. `ces-importer-secret`
        - `config.importer.sshSecretDataKey` contains the name of the key which holds the value `privateKey`
     - creation example:
       `kubectl -n ecosystem create secret generic ces-importer-secret --from-file=privateKey=yourPrivateKeyHere`
  - one, that is required for pulling images of ces-importer-Deployment
     - that one must also be referenced in the respective field in the `values.yaml` file
- a Helm Chart `values.yaml` file filled-in with the relevant configuration data 

## Installation als CES-Komponente

Der ces-importer kann als CES-Komponente installiert und konfiguriert werden.
Ein Beispiel ist hier zu sehen:

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Component
metadata:
  name: ces-importer
  namespace: ecosystem
spec:
  name: ces-importer
  namespace: k8s
  version: 0.0.1
  valuesYamlOverwrite: |
    config:
      logging:
        level: DEBUG
      api:
        host: classic-ces.exporter
        skipTLSVerify: false
      migration:
        regularSchedule: "0/5 * * * *"
        finalSchedule: ""
        changeFQDN: false
```

The following image describes `ces-importer`:

![a diagram of two sysemts "exporter" as source und "importer" as target. Persons with the role administrator apply a Helm Chart and secrets to the importer cluster, which then takes up the work. First, an endpoint will be asked for further system specific Exporter host configuration data. Then a Job resource will be created which does the actual data import work.](images/ces-importer.drawio.png "Configuration points of ces-importer and how the configuration points end up in the runtime process.")
