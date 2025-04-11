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
