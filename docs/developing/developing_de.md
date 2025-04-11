# `ces-importer` entwickeln

## Container bauen & pushen

```Shell
docker build -t registry.cloudogu.com/testing/ces-importer:0.0.1 .

docker push registry.cloudogu.com/testing/ces-importer:0.0.1
```

## Helm-Chart erstellen & pushen

```shell
make helm-package

helm push target/k8s/helm/ces-importer-0.0.1.tgz oci://registry.cloudogu.com/testing/ces-importer-helm
```