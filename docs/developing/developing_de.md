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

## Lokales Helm-Chart installieren

```shell
make helm-package

helm install -n ecosystem -f myvalues.yaml ces-importer target/k8s/helm/ces-importer-0.0.1.tgz --version 0.0.1
```

## Helm Chart wieder komplett deinstallieren

```shell
helm uninstall -n ecosystem ces-importer
```
