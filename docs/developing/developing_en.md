# Developing `ces-importer`

## Build & Push Container

```shell
docker build -t registry.cloudogu.com/testing/ces-importer:0.0.1 . 

docker push registry.cloudogu.com/testing/ces-importer:0.0.1
```

## Build & Push Helm-Chart

```shell
make helm-package

helm push target/k8s/helm/ces-importer-0.0.1.tgz oci://registry.cloudogu.com/testing/ces-importer-helm
```

## Install a local Helm Chart

```shell
make helm-package

# Example secrets
# kubectl -n ecosystem create secret generic ces-importer-secret --from-file=privateKey=yourPrivateKeyHere
# kubectl -n ecosystem create secret generic ces-exporter-secret --from-literal=apiKey=ApiKey-example-123

helm install -n ecosystem -f myvalues.yaml ces-importer target/k8s/helm/ces-importer-0.0.1.tgz --version 0.0.1
```

## Completely remove a Helm Chart

```shell
helm uninstall -n ecosystem ces-importer
```
