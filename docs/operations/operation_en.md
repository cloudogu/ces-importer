# Running `ces-importer`

## Before the start

There are some requirements before the deployment:

- a k8s secret that contains the exporter system's API key
   - the name of the secret and the data field that will hold the value must be configured in these two values:
      - `config.exporter.apiSecretName` contains the name of the k8s secret resource, f. i. `ces-exporter-secret`
      - `config.exporter.apiSecretDataKey` contains the name of the key which holds the value `apiKey`
   - creation example:
     `kubectl -n ecosystem create secret generic ces-exporter-secret --from-literal=apiKey=ApiKey-example-123`
- a k8s secret that contains the SSH private key which will be used to access the exporter system via SSH
   - the name of the secret and the data field that will hold the value must be configured in these two values:
      - `config.importer.sshSecretName` contains the name of the k8s secret resource, f. i. `ces-importer-secret`
      - `config.importer.sshSecretDataKey` contains the name of the key which holds the value `privateKey`
   - creation example:
     `kubectl -n ecosystem create secret generic ces-importer-secret --from-file=privateKey=yourPrivateKeyHere`

## Deploying the application with Helm

================
TBD
==========
